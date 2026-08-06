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
	"github.com/onlyLTY/dockerCopilot/internal/module"
	"gopkg.in/yaml.v3"
)

type Result struct {
	Output string `json:"output"`
}

type ProgressReporter func(message string)

// Available 通过注入的 API 客户端探测 Docker 守护进程是否可用。
// 不检查 PATH，也不依赖 Docker CLI / Compose 插件二进制。
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

// Up 通过 Docker Engine API 原生部署 compose 项目。
// 不使用 github.com/docker/compose/v2（会拖入 buildkit/buildx/containerd），也不调用 docker CLI。
// 支持：基于 image 的常见字段（ports/volumes/env/restart/healthcheck/resources/user/privileged 等）、
// depends_on 多服务排序、命名网络与数据卷。
// 不支持：build:（镜像须已存在或可拉取）、secrets、configs、profiles、swarm 多副本编排。
func Up(ctx context.Context, dockerClient client.APIClient, projectDir string, files []string, timeout time.Duration, pullImages bool) (Result, error) {
	return up(ctx, dockerClient, projectDir, files, timeout, pullImages, nil)
}

func UpWithProgress(ctx context.Context, dockerClient client.APIClient, projectDir string, files []string, timeout time.Duration, pullImages bool, report ProgressReporter) (Result, error) {
	return up(ctx, dockerClient, projectDir, files, timeout, pullImages, report)
}

func up(ctx context.Context, dockerClient client.APIClient, projectDir string, files []string, timeout time.Duration, pullImages bool, report ProgressReporter) (Result, error) {
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
	// 每步耗时由前端基于 TaskStep.StartedAt/EndedAt 的 durationMs 展示；
	// 这里只记录操作信息，不再附加“耗时”（基于相邻日志间隔的计时不准确）。
	logf := func(format string, args ...interface{}) {
		message := fmt.Sprintf(format, args...)
		fmt.Fprintf(out, "%s\n", message)
		if report != nil {
			report(message)
		}
	}

	// 提前拒绝仅 build 的服务：无 image 无法继续，且构建明确不在本实现范围内（会引入 buildkit）。
	for name, svc := range project.Services {
		if svc.Image == "" {
			return Result{Output: sanitizeOutput(out.String())}, fmt.Errorf("服务 %s 未指定 image，当前不支持从源码构建（build:）", name)
		}
	}

	// 顺序：网络 → 数据卷 → 按 depends_on 拓扑部署服务。
	networkNames, defaultNetwork, err := ensureNetworks(runCtx, dockerClient, project, logf)
	if err != nil {
		return Result{Output: sanitizeOutput(out.String())}, err
	}

	volumeNames, err := ensureVolumes(runCtx, dockerClient, project, logf)
	if err != nil {
		return Result{Output: sanitizeOutput(out.String())}, err
	}

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

// ensureNetworks 为缺失的声明网络建网；返回 compose 网络键→真实名，
// 以及项目默认网络名（项目未声明任何网络时会创建）。
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

// ensureVolumes 为缺失的声明命名卷建卷；返回 compose 卷键→真实名。
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

// deployService 按需拉镜像、转换配置，并在配置哈希变化时重建后启动容器。
func deployService(ctx context.Context, cli client.APIClient, projectName, root, serviceName string, svc composeTypes.ServiceConfig, defaultNetwork string, networkNames, volumeNames map[string]string, pullImages bool, logf func(string, ...interface{})) error {
	t, err := translateService(projectName, root, serviceName, svc, defaultNetwork, networkNames, volumeNames)
	if err != nil {
		return fmt.Errorf("服务 %s 配置转换失败: %w", serviceName, err)
	}

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
	// 其它错误也当「本地没有」，走 pull；真正坏掉的客户端会在 pull 路径暴露。
	return false, nil
}

func pullImage(ctx context.Context, cli client.APIClient, imageRef string) error {
	candidates, localName, err := module.ResolvePullCandidates(imageRef)
	if err != nil {
		return err
	}
	var errs []string
	for _, candidate := range candidates {
		reader, pullErr := cli.ImagePull(ctx, candidate, image.PullOptions{})
		if pullErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", candidate, pullErr))
			continue
		}
		drainErr := drainComposePull(reader)
		_ = reader.Close()
		if drainErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", candidate, drainErr))
			continue
		}
		// 加速源拉下的镜像 tag 回原名，容器 Config.Image 保持 compose 中的引用。
		if candidate != localName {
			_ = cli.ImageTag(ctx, candidate, localName)
		}
		return nil
	}
	if len(errs) == 0 {
		return fmt.Errorf("拉取镜像失败：无可用源")
	}
	return fmt.Errorf("拉取镜像失败：%s", strings.Join(errs, "；"))
}

func drainComposePull(reader io.Reader) error {
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

// topoSort 按 depends_on 拓扑排序，保证依赖先启动；有环则报错。
// 同层按服务名排序，保证输出稳定。
func topoSort(services composeTypes.Services) ([]string, error) {
	visited := map[string]int{} // 0=未访问 1=访问中 2=已完成
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

// serviceConfigHash 对服务定义做稳定哈希，重复部署时仅在配置漂移时重建。
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
