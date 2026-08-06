package compose_runner

import (
	"fmt"
	"sort"
	"strings"
	"time"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"

	"github.com/docker/go-connections/nat"
)

// Compose 标准 label：ScanProjects（compose_project）据此把容器归属回项目；
// 缺失则部署出的容器不会出现在项目列表中。
const (
	labelProject         = "com.docker.compose.project"
	labelService         = "com.docker.compose.service"
	labelWorkingDir      = "com.docker.compose.project.working_dir"
	labelConfigHash      = "com.docker.compose.config-hash"
	labelOneoff          = "com.docker.compose.oneoff"
	labelVersion         = "com.docker.compose.version"
	labelContainerNumber = "com.docker.compose.container-number"
)

// translated 聚合 ContainerCreate 所需的三份配置，以及解析后的容器名。
type translated struct {
	name       string
	config     *container.Config
	hostConfig *container.HostConfig
	network    *network.NetworkingConfig
}

// translateService 将 compose ServiceConfig 转为 Docker Engine API 结构。
// root 为项目工作目录（写入 compose label）；服务未声明网络时挂到 defaultNetwork。
func translateService(projectName, root, serviceName string, svc composeTypes.ServiceConfig, defaultNetwork string, networkNames, volumeNames map[string]string) (translated, error) {
	name := svc.ContainerName
	if name == "" {
		name = projectName + "-" + serviceName + "-1"
	}

	cfg := &container.Config{
		Image:      svc.Image,
		Hostname:   svc.Hostname,
		Domainname: svc.DomainName,
		User:       svc.User,
		Tty:        svc.Tty,
		OpenStdin:  svc.StdinOpen,
		WorkingDir: svc.WorkingDir,
		StopSignal: svc.StopSignal,
		Labels:     buildLabels(projectName, root, serviceName, svc),
	}
	if len(svc.Command) > 0 {
		cfg.Cmd = []string(svc.Command)
	}
	if len(svc.Entrypoint) > 0 {
		cfg.Entrypoint = []string(svc.Entrypoint)
	}
	if env := environmentSlice(svc.Environment); len(env) > 0 {
		cfg.Env = env
	}
	if hc := translateHealthcheck(svc.HealthCheck); hc != nil {
		cfg.Healthcheck = hc
	}

	hostCfg := &container.HostConfig{
		Privileged: svc.Privileged,
		CapAdd:     svc.CapAdd,
		CapDrop:    svc.CapDrop,
		DNS:        []string(svc.DNS),
		DNSSearch:  []string(svc.DNSSearch),
		DNSOptions: svc.DNSOpts,
		ExtraHosts: svc.ExtraHosts.AsList(":"),
		GroupAdd:   svc.GroupAdd,
		ShmSize:    int64(svc.ShmSize),
		Runtime:    svc.Runtime,
	}
	if svc.NetworkMode != "" {
		hostCfg.NetworkMode = container.NetworkMode(svc.NetworkMode)
	}
	if svc.Pid != "" {
		hostCfg.PidMode = container.PidMode(svc.Pid)
	}
	if svc.Ipc != "" {
		hostCfg.IpcMode = container.IpcMode(svc.Ipc)
	}
	if svc.PidsLimit != 0 {
		limit := svc.PidsLimit
		hostCfg.PidsLimit = &limit
	}
	if len(svc.SecurityOpt) > 0 {
		hostCfg.SecurityOpt = svc.SecurityOpt
	}
	if svc.Init != nil {
		hostCfg.Init = svc.Init
	}
	hostCfg.RestartPolicy = translateRestart(svc.Restart)
	applyResources(&hostCfg.Resources, svc)

	// host 网络模式下端口映射无效，跳过。
	if !isHostNetwork(svc.NetworkMode) {
		exposed, bindings, err := translatePorts(svc.Ports)
		if err != nil {
			return translated{}, err
		}
		if len(exposed) > 0 {
			cfg.ExposedPorts = exposed
		}
		if len(bindings) > 0 {
			hostCfg.PortBindings = bindings
		}
	}

	binds, mounts, err := translateVolumes(svc.Volumes, volumeNames)
	if err != nil {
		return translated{}, err
	}
	if len(binds) > 0 {
		hostCfg.Binds = binds
	}
	if len(mounts) > 0 {
		hostCfg.Mounts = mounts
	}

	netCfg := translateNetworks(svc, defaultNetwork, networkNames)

	return translated{name: name, config: cfg, hostConfig: hostCfg, network: netCfg}, nil
}

func buildLabels(projectName, root, serviceName string, svc composeTypes.ServiceConfig) map[string]string {
	labels := map[string]string{}
	for k, v := range svc.Labels {
		labels[k] = v
	}
	labels[labelProject] = projectName
	labels[labelService] = serviceName
	labels[labelWorkingDir] = root
	labels[labelOneoff] = "False"
	labels[labelContainerNumber] = "1"
	labels[labelConfigHash] = serviceConfigHash(svc)
	return labels
}

// environmentSlice 将 compose MappingWithEquals 转为 Docker 期望的 KEY=VALUE 切片。
// nil 值表示透传宿主环境，写成裸 KEY（无 =value）；显式值已由 WithDotEnv 等解析。
func environmentSlice(env composeTypes.MappingWithEquals) []string {
	if len(env) == 0 {
		return nil
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(env))
	for _, k := range keys {
		v := env[k]
		if v == nil {
			out = append(out, k)
			continue
		}
		out = append(out, k+"="+*v)
	}
	return out
}

func translateRestart(restart string) container.RestartPolicy {
	// compose 取值：no / always / on-failure[:max] / unless-stopped
	if restart == "" {
		return container.RestartPolicy{}
	}
	name := restart
	var maxRetry int
	if strings.HasPrefix(restart, "on-failure") {
		parts := strings.SplitN(restart, ":", 2)
		name = "on-failure"
		if len(parts) == 2 {
			fmt.Sscanf(parts[1], "%d", &maxRetry)
		}
	}
	return container.RestartPolicy{
		Name:              container.RestartPolicyMode(name),
		MaximumRetryCount: maxRetry,
	}
}

// applyResources 将 compose 资源限制写入 docker Resources。
// 同时支持 deploy.resources.limits 与旧式顶层 mem_limit / cpus。
func applyResources(res *container.Resources, svc composeTypes.ServiceConfig) {
	if svc.Deploy != nil && svc.Deploy.Resources.Limits != nil {
		lim := svc.Deploy.Resources.Limits
		if lim.MemoryBytes != 0 {
			res.Memory = int64(lim.MemoryBytes)
		}
		if lim.NanoCPUs != 0 {
			res.NanoCPUs = int64(float64(lim.NanoCPUs.Value()) * 1e9)
		}
		if lim.Pids != 0 {
			pids := lim.Pids
			res.PidsLimit = &pids
		}
	}
	// 旧式字段仅在有值时覆盖（未设置则保留 deploy 结果）。
	if svc.MemLimit != 0 {
		res.Memory = int64(svc.MemLimit)
	}
	if svc.MemReservation != 0 {
		res.MemoryReservation = int64(svc.MemReservation)
	}
	if svc.CPUS != 0 {
		res.NanoCPUs = int64(float64(svc.CPUS) * 1e9)
	}
	if svc.CPUSet != "" {
		res.CpusetCpus = svc.CPUSet
	}
	if svc.CPUShares != 0 {
		res.CPUShares = svc.CPUShares
	}
}

func translateHealthcheck(hc *composeTypes.HealthCheckConfig) *dockerspec.HealthcheckConfig {
	if hc == nil {
		return nil
	}
	if hc.Disable {
		return &dockerspec.HealthcheckConfig{Test: []string{"NONE"}}
	}
	out := &dockerspec.HealthcheckConfig{
		Test: []string(hc.Test),
	}
	if hc.Interval != nil {
		out.Interval = time.Duration(*hc.Interval)
	}
	if hc.Timeout != nil {
		out.Timeout = time.Duration(*hc.Timeout)
	}
	if hc.StartPeriod != nil {
		out.StartPeriod = time.Duration(*hc.StartPeriod)
	}
	if hc.StartInterval != nil {
		out.StartInterval = time.Duration(*hc.StartInterval)
	}
	if hc.Retries != nil {
		out.Retries = int(*hc.Retries)
	}
	return out
}

func translatePorts(ports []composeTypes.ServicePortConfig) (nat.PortSet, nat.PortMap, error) {
	exposed := nat.PortSet{}
	bindings := nat.PortMap{}
	for _, p := range ports {
		proto := p.Protocol
		if proto == "" {
			proto = "tcp"
		}
		port, err := nat.NewPort(proto, fmt.Sprintf("%d", p.Target))
		if err != nil {
			return nil, nil, fmt.Errorf("端口解析失败 %d/%s: %w", p.Target, proto, err)
		}
		exposed[port] = struct{}{}
		if p.Published != "" {
			bindings[port] = append(bindings[port], nat.PortBinding{
				HostIP:   p.HostIP,
				HostPort: p.Published,
			})
		}
	}
	return exposed, bindings, nil
}

// translateVolumes 将 compose volumes 拆成旧式 bind 字符串与类型化 Mounts。
// bind 走 Binds（与现有备份/恢复路径一致）；命名卷与 tmpfs 走 Mounts。
func translateVolumes(volumes []composeTypes.ServiceVolumeConfig, volumeNames map[string]string) ([]string, []mount.Mount, error) {
	var binds []string
	var mounts []mount.Mount
	for _, v := range volumes {
		switch v.Type {
		case "bind", "":
			if v.Source == "" || v.Target == "" {
				return nil, nil, fmt.Errorf("bind 挂载缺少 source 或 target")
			}
			bind := v.Source + ":" + v.Target
			if v.ReadOnly {
				bind += ":ro"
			}
			binds = append(binds, bind)
		case "volume":
			source := v.Source
			if mapped, ok := volumeNames[source]; ok && mapped != "" {
				source = mapped
			}
			mounts = append(mounts, mount.Mount{
				Type:     mount.TypeVolume,
				Source:   source,
				Target:   v.Target,
				ReadOnly: v.ReadOnly,
			})
		case "tmpfs":
			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeTmpfs,
				Target: v.Target,
			})
		default:
			return nil, nil, fmt.Errorf("不支持的挂载类型: %s", v.Type)
		}
	}
	return binds, mounts, nil
}

// translateNetworks 构建 NetworkingConfig。
// 服务未声明网络且非特殊 network_mode 时，挂到项目默认网络。
func translateNetworks(svc composeTypes.ServiceConfig, defaultNetwork string, networkNames map[string]string) *network.NetworkingConfig {
	// host/none/container:/bridge 自行管理 endpoint，不能再附显式 NetworkingConfig。
	if isSpecialNetworkMode(svc.NetworkMode) {
		return nil
	}
	endpoints := map[string]*network.EndpointSettings{}
	if len(svc.Networks) == 0 {
		if defaultNetwork != "" {
			endpoints[defaultNetwork] = &network.EndpointSettings{}
		}
	} else {
		for netKey, netCfg := range svc.Networks {
			realName := netKey
			if mapped, ok := networkNames[netKey]; ok && mapped != "" {
				realName = mapped
			}
			ep := &network.EndpointSettings{}
			if netCfg != nil {
				ep.Aliases = netCfg.Aliases
				if netCfg.Ipv4Address != "" || netCfg.Ipv6Address != "" {
					ep.IPAMConfig = &network.EndpointIPAMConfig{
						IPv4Address: netCfg.Ipv4Address,
						IPv6Address: netCfg.Ipv6Address,
					}
				}
			}
			endpoints[realName] = ep
		}
	}
	if len(endpoints) == 0 {
		return nil
	}
	return &network.NetworkingConfig{EndpointsConfig: endpoints}
}

// isHostNetwork 判断是否 host 网络。
// 用原始字符串比较，不用 NetworkMode.IsHost()：后者在 Windows 恒为 false，
// 会破坏跨平台测试与非 linux 构建。
func isHostNetwork(mode string) bool {
	return mode == "host"
}

// isSpecialNetworkMode：host/none/bridge/container:<name> 自行管理 endpoint，
// 不得再附显式 NetworkingConfig。
func isSpecialNetworkMode(mode string) bool {
	if mode == "" {
		return false
	}
	return mode == "host" || mode == "none" || mode == "bridge" || strings.HasPrefix(mode, "container:")
}
