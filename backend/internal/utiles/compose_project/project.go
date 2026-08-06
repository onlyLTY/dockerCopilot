package compose_project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	composecli "github.com/compose-spec/compose-go/v2/cli"
	composeTypes "github.com/compose-spec/compose-go/v2/types"
	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	appTypes "github.com/onlyLTY/dockerCopilot/internal/types"
)

const (
	statusUsing   = "using"
	statusStopped = "stopped"
	statusUnused  = "unused"
	statusUnknown = "unknown"
)

var composeFilenames = map[string]bool{
	"compose.yaml":        true,
	"compose.yml":         true,
	"docker-compose.yaml": true,
	"docker-compose.yml":  true,
}

type projectFiles struct {
	root  string
	files []string
}

func ScanProjects(ctx context.Context, svcCtx *svc.ServiceContext) (*appTypes.ComposeProjectsResponse, error) {
	if err := svcCtx.RequireDocker(); err != nil {
		return nil, err
	}
	groups, err := discoverFiles(svcCtx)
	if err != nil {
		return nil, err
	}
	containers, err := svcCtx.DockerClient.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	result := &appTypes.ComposeProjectsResponse{
		Projects: make([]appTypes.ComposeProject, 0, len(groups)),
	}
	for _, group := range groups {
		project := appTypes.ComposeProject{
			ID:         ProjectID(group.root),
			Root:       displayRoot(svcCtx, group.root),
			Files:      make([]appTypes.ComposeFile, 0, len(group.files)),
			Containers: make([]appTypes.ComposeContainer, 0),
			Ports:      make([]appTypes.ComposePort, 0),
			Warnings:   make([]string, 0),
		}
		for _, file := range group.files {
			info, statErr := os.Stat(file)
			composeFile := appTypes.ComposeFile{
				Name: filepath.Base(file),
				Path: file,
			}
			if statErr != nil {
				composeFile.Error = statErr.Error()
				project.Warnings = append(project.Warnings, "无法读取 "+file)
			} else {
				composeFile.Size = info.Size()
				composeFile.ModifiedAt = info.ModTime()
			}
			project.Files = append(project.Files, composeFile)
		}

		parsed, parseErr := loadProject(group)
		if parseErr != nil {
			project.Status = statusUnknown
			project.Warnings = append(project.Warnings, parseErr.Error())
		} else {
			project.Name = parsed.Name
			project.Image = parsed.Image
			for i := range project.Files {
				project.Files[i].Valid = true
			}
			project.Containers = matchingContainers(ctx, svcCtx, parsed.Name, group.root, containers)
			for _, c := range project.Containers {
				project.Ports = append(project.Ports, c.Ports...)
			}
			project.Status = projectStatus(project.Containers)
		}
		result.Projects = append(result.Projects, project)
	}

	sort.Slice(result.Projects, func(i, j int) bool { return result.Projects[i].ID < result.Projects[j].ID })
	for _, project := range result.Projects {
		result.Summary.Total++
		switch project.Status {
		case statusUsing:
			result.Summary.Using++
		case statusStopped:
			result.Summary.Stopped++
		case statusUnused:
			result.Summary.Unused++
		default:
			result.Summary.Unknown++
		}
	}
	return result, nil
}

func ListPorts(ctx context.Context, svcCtx *svc.ServiceContext) (*appTypes.PortsResponse, error) {
	if err := svcCtx.RequireDocker(); err != nil {
		return nil, err
	}
	containers, err := svcCtx.DockerClient.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}
	result := &appTypes.PortsResponse{Ports: make([]appTypes.PortUsage, 0), Conflicts: make([]string, 0), Warnings: make([]string, 0)}
	counts := make(map[string]int)
	for _, item := range containers {
		inspect, err := svcCtx.DockerClient.ContainerInspect(ctx, item.ID)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("容器 %s 端口信息读取失败: %v", firstName(item.Names), err))
			continue
		}
		project := item.Labels["com.docker.compose.project"]
		ports := containerPorts(inspect)
		image := ""
		if inspect.Config != nil {
			image = inspect.Config.Image
		}
		for _, port := range ports {
			usage := appTypes.PortUsage{
				Project: project, ContainerID: item.ID, ContainerName: firstName(item.Names),
				Image: image, State: inspect.State.Status, HostIP: port.HostIP, HostPort: port.HostPort,
				ContainerPort: port.ContainerPort, Protocol: port.Protocol, Published: port.Published,
			}
			if usage.Published {
				usage.ConflictKey = strings.Join([]string{usage.HostIP, usage.HostPort, usage.Protocol}, ":")
				counts[usage.ConflictKey]++
			}
			result.Ports = append(result.Ports, usage)
		}
	}
	for key, count := range counts {
		if count > 1 {
			result.Conflicts = append(result.Conflicts, key)
		}
	}
	sort.Strings(result.Conflicts)
	return result, nil
}

func displayRoot(svcCtx *svc.ServiceContext, root string) string {
	for _, mapping := range svcCtx.Config.Compose.PathMappings {
		host, err := filepath.Abs(mapping.HostPath)
		if err == nil && filepath.Clean(host) == filepath.Clean(root) {
			return filepath.ToSlash(mapping.ContainerPath)
		}
	}
	return filepath.ToSlash(root)
}
func discoverFiles(svcCtx *svc.ServiceContext) ([]projectFiles, error) {
	var groups []projectFiles
	seen := make(map[string][]string)
	paths := svcCtx.Config.Compose.ScanPaths
	if len(paths) == 0 {
		return nil, fmt.Errorf("未配置 Compose 扫描目录")
	}
	for _, root := range paths {
		if root == "" {
			continue
		}
		rootAbs, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(rootAbs); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		maxDepth := svcCtx.Config.Compose.MaxDepth
		err = filepath.Walk(rootAbs, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() {
				if maxDepth > 0 && path != rootAbs {
					rel, _ := filepath.Rel(rootAbs, path)
					if depth := len(strings.Split(rel, string(filepath.Separator))); depth > maxDepth {
						return filepath.SkipDir
					}
				}
				return nil
			}
			if composeFilenames[strings.ToLower(info.Name())] {
				dir := filepath.Dir(path)
				seen[dir] = append(seen[dir], path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	for root, files := range seen {
		sort.Strings(files)
		groups = append(groups, projectFiles{root: root, files: files})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].root < groups[j].root })
	return groups, nil
}

func loadProject(group projectFiles) (*composeProject, error) {
	project, err := LoadProject(context.Background(), group.root, group.files)
	if err != nil {
		return nil, err
	}
	return &composeProject{Name: project.Name, Image: firstServiceImage(project)}, nil
}

func firstServiceImage(project *composeTypes.Project) string {
	if project == nil {
		return ""
	}
	names := make([]string, 0, len(project.Services))
	for name := range project.Services {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if image := strings.TrimSpace(project.Services[name].Image); image != "" {
			return image
		}
	}
	return ""
}

// LoadProject loads the same normalized Compose model used by deployment and validation.
func LoadProject(ctx context.Context, root string, files []string) (*composeTypes.Project, error) {
	if root == "" || len(files) == 0 {
		return nil, fmt.Errorf("Compose 项目文件不完整")
	}
	// 仅 WithDotEnv：${VAR} 来自项目目录 .env，不注入面板进程环境（避免 secretKey 等泄露进业务容器）。
	options, err := composecli.NewProjectOptions(
		files,
		composecli.WithWorkingDirectory(root),
		composecli.WithDotEnv,
		composecli.WithResolvedPaths(true),
		composecli.WithDiscardEnvFile,
	)
	if err != nil {
		return nil, err
	}
	return composecli.ProjectFromOptions(ctx, options)
}

type composeProject struct {
	Name  string
	Image string
}

func matchingContainers(ctx context.Context, svcCtx *svc.ServiceContext, projectName, root string, containers []dockerTypes.Container) []appTypes.ComposeContainer {
	result := make([]appTypes.ComposeContainer, 0)
	for _, item := range containers {
		if item.Labels["com.docker.compose.project"] != projectName || !sameComposeRoot(item.Labels, root) {
			continue
		}
		inspect, err := svcCtx.DockerClient.ContainerInspect(ctx, item.ID)
		if err != nil {
			continue
		}
		result = append(result, appTypes.ComposeContainer{
			ID: item.ID, Name: firstName(item.Names), Service: item.Labels["com.docker.compose.service"],
			State: inspect.State.Status, Ports: containerPorts(inspect),
		})
	}
	return result
}

func sameComposeRoot(labels map[string]string, root string) bool {
	labelRoot := labels["com.docker.compose.project.working_dir"]
	if labelRoot == "" {
		return true
	}
	abs, err := filepath.Abs(labelRoot)
	return err == nil && filepath.Clean(abs) == filepath.Clean(root)
}

func projectStatus(containers []appTypes.ComposeContainer) string {
	if len(containers) == 0 {
		return statusUnused
	}
	for _, item := range containers {
		if item.State == "running" {
			return statusUsing
		}
	}
	return statusStopped
}

func firstName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return strings.TrimPrefix(names[0], "/")
}

func containerPorts(inspect dockerTypes.ContainerJSON) []appTypes.ComposePort {
	ports := make([]appTypes.ComposePort, 0)
	seen := make(map[string]bool)
	unpublished := make(map[string]int)
	publishedTargets := make(map[string]bool)
	add := func(hostIP, hostPort, containerPort, protocol string, published bool) {
		if containerPort == "" {
			return
		}
		targetKey := containerPort + ":" + protocol
		// Published bindings that differ only by host IP (IPv4 0.0.0.0, IPv6 ::,
		// or an empty wildcard) map to the same host port and would otherwise
		// appear as duplicate rows. Collapse them by keying on the host port
		// alone for published entries; keep the host IP in the key for
		// unpublished entries so exposed-only targets stay distinct.
		key := hostIP + ":" + hostPort + ":" + targetKey
		if published {
			key = hostPort + ":" + targetKey
		}
		if seen[key] {
			return
		}
		// Docker can report an exposed target as well as its published binding.
		// Replace the unbound entry when the real binding is discovered.
		if published {
			if index, ok := unpublished[targetKey]; ok {
				ports[index] = appTypes.ComposePort{HostIP: hostIP, HostPort: hostPort, ContainerPort: containerPort, Protocol: protocol, Published: true}
				delete(unpublished, targetKey)
				seen[key] = true
				publishedTargets[targetKey] = true
				return
			}
			publishedTargets[targetKey] = true
		} else if hostIP == "" && hostPort == "" && publishedTargets[targetKey] {
			return
		}
		ports = append(ports, appTypes.ComposePort{HostIP: hostIP, HostPort: hostPort, ContainerPort: containerPort, Protocol: protocol, Published: published})
		seen[key] = true
		if !published && hostIP == "" && hostPort == "" {
			unpublished[targetKey] = len(ports) - 1
		}
	}
	if inspect.NetworkSettings != nil {
		for target, bindings := range inspect.NetworkSettings.Ports {
			protocol, containerPort := nat.SplitProtoPort(string(target))
			if len(bindings) == 0 {
				add("", "", containerPort, protocol, false)
				continue
			}
			for _, binding := range bindings {
				add(binding.HostIP, binding.HostPort, containerPort, protocol, binding.HostPort != "")
			}
		}
	}
	if inspect.HostConfig != nil {
		for target, bindings := range inspect.HostConfig.PortBindings {
			protocol, containerPort := nat.SplitProtoPort(string(target))
			for _, binding := range bindings {
				add(binding.HostIP, binding.HostPort, containerPort, protocol, binding.HostPort != "")
			}
		}
	}
	if inspect.Config != nil {
		for target := range inspect.Config.ExposedPorts {
			protocol, containerPort := nat.SplitProtoPort(string(target))
			add("", "", containerPort, protocol, false)
		}
	}
	return ports
}
