package backupCompose

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	composeType "github.com/compose-spec/compose-go/types"
	"github.com/docker/docker/api/types/container"
	composeNat "github.com/docker/go-connections/nat"
	"github.com/zeromicro/go-zero/core/logx"
	"sigs.k8s.io/yaml"
)

type Options struct {
	IncludeSensitiveEnvironment bool
}

// DockerConfig2ComposeYaml 将docker config转换为docker-compose.yaml
func DockerConfig2ComposeYaml(containerJSONs []container.InspectResponse) ([]byte, error) {
	return DockerConfig2ComposeYamlWithOptions(containerJSONs, Options{})
}

func DockerConfig2ComposeYamlWithOptions(containerJSONs []container.InspectResponse, options Options) ([]byte, error) {
	var c composeYaml
	c.Services = make(map[string]composeType.ServiceConfig, len(containerJSONs))
	c.Networks = make(map[string]composeType.NetworkConfig)
	if !options.IncludeSensitiveEnvironment {
		c.Warnings = []string{"Sensitive environment variable values were omitted; provide them when deploying this Compose file."}
	}
	for _, containerJSON := range containerJSONs {
		if containerJSON.Config == nil || containerJSON.HostConfig == nil || containerJSON.NetworkSettings == nil {
			return nil, fmt.Errorf("container %s has incomplete inspect data", containerJSON.ID)
		}
		var s composeType.ServiceConfig
		formatBaseServiceConfig(containerJSON, &s)
		formatEnvServiceConfig(containerJSON, &s, options)
		formatNetworkServiceConfig(containerJSON, &s, &c)
		formatVolumeServiceConfig(containerJSON, &s)
		c.Services[s.Name] = s
	}
	yamlData, yamlMarshalErr := yaml.Marshal(c)
	if yamlMarshalErr != nil {
		return nil, fmt.Errorf("marshal compose yaml: %w", yamlMarshalErr)
	}
	return yamlData, nil
}

func formatBaseServiceConfig(containerJSON container.InspectResponse, s *composeType.ServiceConfig) {
	s.Image = containerJSON.Config.Image
	name, cutNameResult := strings.CutPrefix(containerJSON.Name, "/")
	if !cutNameResult {
		logx.Infof("cutting name is: %s", containerJSON.Name)
	}
	s.ContainerName = name
	s.Name = name
	s.Tty = containerJSON.Config.Tty
	s.StdinOpen = containerJSON.Config.OpenStdin
	s.Hostname = containerJSON.Config.Hostname
	s.DomainName = containerJSON.Config.Domainname
	s.User = containerJSON.Config.User
	s.Platform = containerJSON.Platform
	s.Labels = composeType.Labels(containerJSON.Config.Labels)
	s.StopSignal = containerJSON.Config.StopSignal
	if containerJSON.Config.StopTimeout != nil {
		duration := composeType.Duration(time.Duration(*containerJSON.Config.StopTimeout) * time.Second)
		s.StopGracePeriod = &duration
	}
	if health := containerJSON.Config.Healthcheck; health != nil {
		healthConfig := &composeType.HealthCheckConfig{
			Test:    composeType.HealthCheckTest(health.Test),
			Retries: uint64Pointer(health.Retries),
			Disable: len(health.Test) == 1 && strings.EqualFold(health.Test[0], "NONE"),
		}
		healthConfig.Interval = durationPointer(health.Interval)
		healthConfig.Timeout = durationPointer(health.Timeout)
		healthConfig.StartPeriod = durationPointer(health.StartPeriod)
		healthConfig.StartInterval = durationPointer(health.StartInterval)
		s.HealthCheck = healthConfig
	}
	if len(containerJSON.Config.Entrypoint) > 0 {
		s.Entrypoint = composeType.ShellCommand(containerJSON.Config.Entrypoint)
	}
	s.WorkingDir = containerJSON.Config.WorkingDir
	s.Restart = string(containerJSON.HostConfig.RestartPolicy.Name)
	s.Privileged = containerJSON.HostConfig.Privileged
	s.ReadOnly = containerJSON.HostConfig.ReadonlyRootfs
	s.Init = containerJSON.HostConfig.Init
	s.CapAdd = append([]string(nil), containerJSON.HostConfig.CapAdd...)
	s.CapDrop = append([]string(nil), containerJSON.HostConfig.CapDrop...)
	s.SecurityOpt = append([]string(nil), containerJSON.HostConfig.SecurityOpt...)
	s.GroupAdd = append([]string(nil), containerJSON.HostConfig.GroupAdd...)
	s.DNS = append(composeType.StringList(nil), containerJSON.HostConfig.DNS...)
	s.DNSOpts = append([]string(nil), containerJSON.HostConfig.DNSOptions...)
	s.DNSSearch = append(composeType.StringList(nil), containerJSON.HostConfig.DNSSearch...)
	s.ExtraHosts = extraHosts(containerJSON.HostConfig.ExtraHosts)
	s.Sysctls = composeType.Mapping(containerJSON.HostConfig.Sysctls)
	s.Tmpfs = tmpfsList(containerJSON.HostConfig.Tmpfs)
	s.Ipc = string(containerJSON.HostConfig.IpcMode)
	s.Pid = string(containerJSON.HostConfig.PidMode)
	s.Uts = string(containerJSON.HostConfig.UTSMode)
	s.UserNSMode = string(containerJSON.HostConfig.UsernsMode)
	s.Runtime = containerJSON.HostConfig.Runtime
	s.Isolation = string(containerJSON.HostConfig.Isolation)
	s.Links = append([]string(nil), containerJSON.HostConfig.Links...)
	s.ShmSize = composeType.UnitBytes(containerJSON.HostConfig.ShmSize)
	s.VolumeDriver = containerJSON.HostConfig.VolumeDriver
	s.VolumesFrom = append([]string(nil), containerJSON.HostConfig.VolumesFrom...)
	s.CgroupParent = containerJSON.HostConfig.CgroupParent
	s.Cgroup = string(containerJSON.HostConfig.Cgroup)
	s.Annotations = composeType.Mapping(containerJSON.HostConfig.Annotations)
	s.DeviceCgroupRules = append([]string(nil), containerJSON.HostConfig.DeviceCgroupRules...)
	s.Devices = formatDevices(containerJSON.HostConfig.Devices)
	s.Ulimits = formatUlimits(containerJSON.HostConfig.Ulimits)
	s.CPUShares = containerJSON.HostConfig.CPUShares
	s.CPUPeriod = containerJSON.HostConfig.CPUPeriod
	s.CPUQuota = containerJSON.HostConfig.CPUQuota
	s.CPURTPeriod = containerJSON.HostConfig.CPURealtimePeriod
	s.CPURTRuntime = containerJSON.HostConfig.CPURealtimeRuntime
	s.CPUSet = containerJSON.HostConfig.CpusetCpus
	if containerJSON.HostConfig.NanoCPUs > 0 {
		s.CPUS = float32(float64(containerJSON.HostConfig.NanoCPUs) / 1_000_000_000)
	}
	s.CPUCount = containerJSON.HostConfig.CPUCount
	s.CPUPercent = float32(containerJSON.HostConfig.CPUPercent)
	s.MemLimit = composeType.UnitBytes(containerJSON.HostConfig.Memory)
	s.MemReservation = composeType.UnitBytes(containerJSON.HostConfig.MemoryReservation)
	s.MemSwapLimit = composeType.UnitBytes(containerJSON.HostConfig.MemorySwap)
	if containerJSON.HostConfig.MemorySwappiness != nil {
		s.MemSwappiness = composeType.UnitBytes(*containerJSON.HostConfig.MemorySwappiness)
	}
	if containerJSON.HostConfig.OomKillDisable != nil {
		s.OomKillDisable = *containerJSON.HostConfig.OomKillDisable
	}
	s.OomScoreAdj = int64(containerJSON.HostConfig.OomScoreAdj)
	if containerJSON.HostConfig.PidsLimit != nil {
		s.PidsLimit = *containerJSON.HostConfig.PidsLimit
	}
	if driver := containerJSON.HostConfig.LogConfig.Type; driver != "" {
		s.Logging = &composeType.LoggingConfig{Driver: driver, Options: composeType.Options(containerJSON.HostConfig.LogConfig.Config)}
	}
	if len(containerJSON.Config.Cmd) > 0 {
		s.Command = composeType.ShellCommand(containerJSON.Config.Cmd)
	}
}

func formatEnvServiceConfig(containerJSON container.InspectResponse, s *composeType.ServiceConfig, options Options) {
	s.Environment = composeType.NewMappingWithEquals(containerJSON.Config.Env)
	if options.IncludeSensitiveEnvironment {
		return
	}
	for key := range s.Environment {
		if isSensitiveEnvironmentKey(key) {
			s.Environment[key] = nil
		}
	}
}

func formatNetworkServiceConfig(containerJSON container.InspectResponse, s *composeType.ServiceConfig, c *composeYaml) {
	networkMode := string(containerJSON.HostConfig.NetworkMode)
	if networkMode == "host" || networkMode == "none" || strings.HasPrefix(networkMode, "container:") || strings.HasPrefix(networkMode, "service:") {
		s.NetworkMode = networkMode
	} else if containerJSON.NetworkSettings != nil && len(containerJSON.NetworkSettings.Networks) > 0 {
		s.Networks = make(map[string]*composeType.ServiceNetworkConfig, len(containerJSON.NetworkSettings.Networks))
		for networkName, endpoint := range containerJSON.NetworkSettings.Networks {
			if endpoint == nil || networkName == "" {
				continue
			}
			aliases := filterGeneratedAliases(endpoint.Aliases, containerJSON.Name, containerJSON.ID)
			serviceNetwork := &composeType.ServiceNetworkConfig{Aliases: aliases}
			if endpoint.IPAMConfig != nil {
				serviceNetwork.Ipv4Address = endpoint.IPAMConfig.IPv4Address
				serviceNetwork.Ipv6Address = endpoint.IPAMConfig.IPv6Address
				serviceNetwork.LinkLocalIPs = append([]string(nil), endpoint.IPAMConfig.LinkLocalIPs...)
			}
			s.Networks[networkName] = serviceNetwork
			c.Networks[networkName] = composeType.NetworkConfig{
				Name: networkName, External: composeType.External{External: true},
			}
		}
	} else if networkMode != "" && networkMode != "default" && networkMode != "bridge" {
		s.NetworkMode = networkMode
	}
	for containerPort, bindings := range containerJSON.HostConfig.PortBindings {
		proto, port := composeNat.SplitProtoPort(string(containerPort))
		portNum, convertErr := strconv.ParseUint(port, 10, 16)
		if convertErr != nil {
			logx.Errorf("Error converting port err is: %v", convertErr)
			continue
		}
		targetPort := uint32(portNum)
		if len(bindings) == 0 {
			s.Ports = append(s.Ports, composeType.ServicePortConfig{Target: targetPort, Protocol: proto})
			continue
		}
		for _, binding := range bindings {
			s.Ports = append(s.Ports, composeType.ServicePortConfig{
				Target: targetPort, Published: binding.HostPort,
				HostIP: binding.HostIP, Protocol: proto,
			})
		}
	}
	if containerJSON.Config != nil {
		for exposedPort := range containerJSON.Config.ExposedPorts {
			if _, published := containerJSON.HostConfig.PortBindings[exposedPort]; published {
				continue
			}
			s.Expose = append(s.Expose, string(exposedPort))
		}
	}
}

func formatVolumeServiceConfig(containerJSON container.InspectResponse, s *composeType.ServiceConfig) {
	for _, containerVolume := range containerJSON.Mounts {
		var v composeType.ServiceVolumeConfig
		v.Type = string(containerVolume.Type)
		v.Source = containerVolume.Source
		if string(containerVolume.Type) == "volume" && containerVolume.Name != "" {
			v.Source = containerVolume.Name
		}
		v.Target = containerVolume.Destination
		v.ReadOnly = !containerVolume.RW
		if v.Type == "bind" {
			v.Bind = &composeType.ServiceVolumeBind{
				Propagation: string(containerVolume.Propagation), CreateHostPath: true,
			}
			for _, option := range strings.Split(containerVolume.Mode, ",") {
				switch option {
				case "z", "Z":
					v.Bind.SELinux = option
				case "consistent", "cached", "delegated":
					v.Consistency = option
				}
			}
		}
		s.Volumes = append(s.Volumes, v)
	}
}
