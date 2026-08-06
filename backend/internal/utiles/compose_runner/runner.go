package compose_runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	composecli "github.com/compose-spec/compose-go/v2/cli"
	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	dockerMsg "github.com/docker/docker/pkg/jsonmessage"
	"gopkg.in/yaml.v3"
)

type Result struct {
	Output string `json:"output"`
}

// Available checks the Docker daemon through the injected API client. It does
// not inspect PATH and does not require a Docker CLI or Compose plugin binary.
func Available(ctx context.Context, dockerClient client.APIClient) bool {
	if dockerClient == nil {
		return false
	}
	_, err := dockerClient.ServerVersion(ctx)
	return err == nil
}

func Config(ctx context.Context, dockerClient client.APIClient, projectDir string, files []string, timeout time.Duration) (Result, error) {
	if projectDir == "" || len(files) == 0 {
		return Result{}, fmt.Errorf("Compose 执行参数不完整")
	}
	project, err := loadProject(ctx, projectDir, files, timeout)
	if err != nil {
		return Result{}, err
	}
	if dockerClient == nil {
		return Result{}, fmt.Errorf("Docker 客户端不可用")
	}
	data, err := yaml.Marshal(project)
	if err != nil {
		return Result{}, fmt.Errorf("Compose 配置序列化失败: %w", err)
	}
	return Result{Output: sanitizeOutput(string(data))}, nil
}

// Up deploys a compose project natively through the Docker Engine API. It does
// NOT use github.com/docker/compose/v2 (which drags in buildkit/buildx/containerd)
// nor the docker CLI. Supported: image-based services with the common fields
// (ports, volumes, env, restart, healthcheck, resources, user, privileged, etc.),
// multi-service projects with depends_on ordering, and named networks/volumes.
// NOT supported: build: (image must already exist or be pullable), secrets,
// configs, profiles, and swarm multi-replica orchestration.
func Up(ctx context.Context, dockerClient client.APIClient, projectDir string, files []string, timeout time.Duration, pullImages bool) (Result, error) {
	if projectDir == "" || len(files) == 0 {
		return Result{}, fmt.Errorf("Compose 执行参数不完整")
	}
	if dockerClient == nil {
		return Result{}, fmt.Errorf("Docker 客户端不可用")
	}
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	project, err := loadProject(runCtx, projectDir, files, timeout)
	if err != nil {
		return Result{}, err
	}

	out := &strings.Builder{}
	// 记录每一步耗时：以上一条日志为基准计算增量，方便前端展示每步花费的时间。
	stepStart := time.Now()
	logf := func(format string, args ...interface{}) {
		elapsed := time.Since(stepStart)
		stepStart = time.Now()
		fmt.Fprintf(out, format+" (耗时 %s)\n", append(args, formatDuration(elapsed))...)
	}

	// Reject build-only services early: without an image we cannot proceed and
	// building is explicitly out of scope (that is what pulls in buildkit).
	for name, svc := range project.Services {
		if svc.Image == "" {
			return Result{Output: sanitizeOutput(out.String())}, fmt.Errorf("服务 %s 未指定 image，当前不支持从源码构建（build:）", name)
		}
	}

	// 1. networks: create declared networks that do not exist yet, plus a
	// project default network when no custom network is declared.
	networkNames, defaultNetwork, err := ensureNetworks(runCtx, dockerClient, project, logf)
	if err != nil {
		return Result{Output: sanitizeOutput(out.String())}, err
	}

	// 2. volumes: create declared named volumes that do not exist yet.
	volumeNames, err := ensureVolumes(runCtx, dockerClient, project, logf)
	if err != nil {
		return Result{Output: sanitizeOutput(out.String())}, err
	}

	// 3. deploy services in depends_on order.
	order, err := topoSort(project.Services)
	if err != nil {
		return Result{Output: sanitizeOutput(out.String())}, err
	}
	for _, serviceName := range order {
		svc := project.Services[serviceName]
		if err := deployService(runCtx, dockerClient, project.Name, projectDir, serviceName, svc, defaultNetwork, networkNames, volumeNames, pullImages, logf); err != nil {
			return Result{Output: sanitizeOutput(out.String())}, err
		}
	}

	if runCtx.Err() != nil {
		return Result{Output: sanitizeOutput(out.String())}, runCtx.Err()
	}
	return Result{Output: sanitizeOutput(out.String())}, nil
}

// ensureNetworks creates any declared networks missing from the daemon and
// returns a map from compose network key to real network name, plus the name of
// the project default network (created when the project declares no networks).
func ensureNetworks(ctx context.Context, cli client.APIClient, project *composeTypes.Project, logf func(string, ...interface{})) (map[string]string, string, error) {
	names := map[string]string{}
	defaultNetwork := ""

	for key, cfg := range project.Networks {
		realName := cfg.Name
		if realName == "" {
			realName = project.Name + "_" + key
		}
		names[key] = realName
		if bool(cfg.External) {
			continue
		}
		if err := createNetworkIfMissing(ctx, cli, realName, project.Name, cfg, logf); err != nil {
			return nil, "", err
		}
	}

	if len(project.Networks) == 0 {
		defaultNetwork = project.Name + "_default"
		if err := createNetworkIfMissing(ctx, cli, defaultNetwork, project.Name, composeTypes.NetworkConfig{}, logf); err != nil {
			return nil, "", err
		}
	}
	return names, defaultNetwork, nil
}

func createNetworkIfMissing(ctx context.Context, cli client.APIClient, name, projectName string, cfg composeTypes.NetworkConfig, logf func(string, ...interface{})) error {
	if _, err := cli.NetworkInspect(ctx, name, network.InspectOptions{}); err == nil {
		return nil
	}
	driver := cfg.Driver
	if driver == "" {
		driver = "bridge"
	}
	opts := network.CreateOptions{
		Driver:     driver,
		Internal:   cfg.Internal,
		Attachable: cfg.Attachable,
		Labels: map[string]string{
			labelProject: projectName,
			labelVersion: "native",
		},
	}
	if len(cfg.DriverOpts) > 0 {
		opts.Options = map[string]string(cfg.DriverOpts)
	}
	if _, err := cli.NetworkCreate(ctx, name, opts); err != nil {
		return fmt.Errorf("创建网络 %s 失败: %w", name, err)
	}
	logf("创建网络 %s", name)
	return nil
}

// ensureVolumes creates any declared named volumes missing from the daemon and
// returns a map from compose volume key to real volume name.
func ensureVolumes(ctx context.Context, cli client.APIClient, project *composeTypes.Project, logf func(string, ...interface{})) (map[string]string, error) {
	names := map[string]string{}
	for key, cfg := range project.Volumes {
		realName := cfg.Name
		if realName == "" {
			realName = project.Name + "_" + key
		}
		names[key] = realName
		if bool(cfg.External) {
			continue
		}
		if _, err := cli.VolumeInspect(ctx, realName); err == nil {
			continue
		}
		opts := volume.CreateOptions{
			Name:   realName,
			Driver: cfg.Driver,
			Labels: map[string]string{
				labelProject: project.Name,
				labelVersion: "native",
			},
		}
		if len(cfg.DriverOpts) > 0 {
			opts.DriverOpts = map[string]string(cfg.DriverOpts)
		}
		if _, err := cli.VolumeCreate(ctx, opts); err != nil {
			return nil, fmt.Errorf("创建数据卷 %s 失败: %w", realName, err)
		}
		logf("创建数据卷 %s", realName)
	}
	return names, nil
}

// deployService pulls the image if needed, translates the service, and creates
// (or recreates when the config hash changed) and starts the container.
func deployService(ctx context.Context, cli client.APIClient, projectName, root, serviceName string, svc composeTypes.ServiceConfig, defaultNetwork string, networkNames, volumeNames map[string]string, pullImages bool, logf func(string, ...interface{})) error {
	t, err := translateService(projectName, root, serviceName, svc, defaultNetwork, networkNames, volumeNames)
	if err != nil {
		return fmt.Errorf("服务 %s 配置转换失败: %w", serviceName, err)
	}

	// pull image when it is not present locally
	present, err := imageExists(ctx, cli, svc.Image)
	if err != nil {
		return err
	}
	if pullImages || !present || strings.EqualFold(svc.PullPolicy, "always") {
		logf("拉取镜像 %s", svc.Image)
		if err := pullImage(ctx, cli, svc.Image); err != nil {
			return fmt.Errorf("服务 %s 拉取镜像失败: %w", serviceName, err)
		}
	}

	// look for an existing container with the same name
	existing, err := findContainerByName(ctx, cli, t.name)
	if err != nil {
		return err
	}
	newHash := t.config.Labels[labelConfigHash]
	if existing != nil {
		oldHash := existing.Labels[labelConfigHash]
		if !pullImages && oldHash == newHash && existing.State == "running" {
			logf("服务 %s 配置未变化，跳过", serviceName)
			return nil
		}
		logf("服务 %s 配置变化，重建容器", serviceName)
		if err := removeContainer(ctx, cli, existing.ID); err != nil {
			return fmt.Errorf("服务 %s 移除旧容器失败: %w", serviceName, err)
		}
	}

	created, err := cli.ContainerCreate(ctx, t.config, t.hostConfig, t.network, nil, t.name)
	if err != nil {
		return fmt.Errorf("服务 %s 创建容器失败: %w", serviceName, err)
	}
	if err := cli.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("服务 %s 启动容器失败: %w", serviceName, err)
	}
	logf("服务 %s 已启动 (%s)", serviceName, t.name)
	return nil
}

func imageExists(ctx context.Context, cli client.APIClient, ref string) (bool, error) {
	_, _, err := cli.ImageInspectWithRaw(ctx, ref)
	if err == nil {
		return true, nil
	}
	if client.IsErrNotFound(err) {
		return false, nil
	}
	// treat other errors as "not present" so we attempt a pull, but surface
	// genuinely broken clients through the pull path instead.
	return false, nil
}

func pullImage(ctx context.Context, cli client.APIClient, ref string) error {
	reader, err := cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()
	// drain the jsonmessage stream so the pull completes and errors surface
	decoder := json.NewDecoder(reader)
	for {
		var msg dockerMsg.JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if msg.Error != nil {
			return fmt.Errorf(msg.Error.Message)
		}
	}
}

func findContainerByName(ctx context.Context, cli client.APIClient, name string) (*containerSummary, error) {
	args := filters.NewArgs()
	args.Add("name", "^/"+name+"$")
	list, err := cli.ContainerList(ctx, container.ListOptions{All: true, Filters: args})
	if err != nil {
		return nil, fmt.Errorf("查询容器 %s 失败: %w", name, err)
	}
	for _, c := range list {
		for _, n := range c.Names {
			if strings.TrimPrefix(n, "/") == name {
				return &containerSummary{ID: c.ID, State: c.State, Labels: c.Labels}, nil
			}
		}
	}
	return nil, nil
}

type containerSummary struct {
	ID     string
	State  string
	Labels map[string]string
}

func removeContainer(ctx context.Context, cli client.APIClient, id string) error {
	timeout := 10
	_ = cli.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout})
	return cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
}

// topoSort orders services so that dependencies (depends_on) start first.
// Returns an error on cyclic dependencies. Services with equal ordering are
// sorted by name for deterministic output.
func topoSort(services composeTypes.Services) ([]string, error) {
	visited := map[string]int{} // 0=unvisited,1=visiting,2=done
	var order []string
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)

	var visit func(string) error
	visit = func(name string) error {
		switch visited[name] {
		case 2:
			return nil
		case 1:
			return fmt.Errorf("服务依赖存在循环: %s", name)
		}
		visited[name] = 1
		svc, ok := services[name]
		if ok {
			deps := make([]string, 0, len(svc.DependsOn))
			for dep := range svc.DependsOn {
				deps = append(deps, dep)
			}
			sort.Strings(deps)
			for _, dep := range deps {
				if _, exists := services[dep]; !exists {
					continue
				}
				if err := visit(dep); err != nil {
					return err
				}
			}
		}
		visited[name] = 2
		order = append(order, name)
		return nil
	}

	for _, name := range names {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	return order, nil
}

// serviceConfigHash produces a stable hash of the service definition so repeat
// deployments can detect configuration drift and recreate only when needed.
func serviceConfigHash(svc composeTypes.ServiceConfig) string {
	data, err := json.Marshal(svc)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func loadProject(ctx context.Context, projectDir string, files []string, timeout time.Duration) (*composeTypes.Project, error) {
	loadCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		loadCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	// 仅 WithDotEnv：与 compose_project.LoadProject 一致，不读取面板进程环境变量。
	options, err := composecli.NewProjectOptions(
		files,
		composecli.WithWorkingDirectory(projectDir),
		composecli.WithDotEnv,
		composecli.WithResolvedPaths(true),
		composecli.WithDiscardEnvFile,
	)
	if err != nil {
		return nil, err
	}
	return composecli.ProjectFromOptions(loadCtx, options)
}

// formatDuration 输出人类可读的耗时，毫秒级用 ms，其余保留一位小数的秒。
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func sanitizeOutput(output string) string {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		for _, marker := range []string{"PASSWORD=", "TOKEN=", "SECRET=", "password:", "token:", "secret:"} {
			if index := strings.Index(strings.ToLower(line), strings.ToLower(marker)); index >= 0 {
				line = line[:index] + marker + "[REDACTED]"
			}
		}
		lines[i] = line
	}
	text := strings.Join(lines, "\n")
	if len(text) > 64*1024 {
		return text[:64*1024] + "..."
	}
	return text
}
