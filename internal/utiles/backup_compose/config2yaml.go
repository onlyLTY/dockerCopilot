package backupCompose

import (
	"fmt"
	"strconv"
	"strings"

	composeType "github.com/compose-spec/compose-go/types"
	"github.com/docker/docker/api/types/container"
	composeNat "github.com/docker/go-connections/nat"
	"github.com/zeromicro/go-zero/core/logx"
	"sigs.k8s.io/yaml"
)

// DockerConfig2ComposeYaml 将docker config转换为docker-compose.yaml
func DockerConfig2ComposeYaml(containerJSONs []container.InspectResponse) ([]byte, error) {
	var c composeYaml
	c.Services = make(map[string]composeType.ServiceConfig, len(containerJSONs))
	for _, containerJSON := range containerJSONs {
		if containerJSON.Config == nil || containerJSON.HostConfig == nil || containerJSON.NetworkSettings == nil {
			return nil, fmt.Errorf("container %s has incomplete inspect data", containerJSON.ID)
		}
		var s composeType.ServiceConfig
		formatBaseServiceConfig(containerJSON, &s)
		formatEnvServiceConfig(containerJSON, &s)
		formatNetworkServiceConfig(containerJSON, &s)
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
	if len(containerJSON.Config.Entrypoint) > 0 {
		s.Entrypoint = composeType.ShellCommand(containerJSON.Config.Entrypoint)
	}
	s.WorkingDir = containerJSON.Config.WorkingDir
	s.Restart = string(containerJSON.HostConfig.RestartPolicy.Name)
	s.Privileged = containerJSON.HostConfig.Privileged
	if len(containerJSON.Config.Cmd) > 0 {
		s.Command = composeType.ShellCommand(containerJSON.Config.Cmd)
	}
}

func formatEnvServiceConfig(containerJSON container.InspectResponse, s *composeType.ServiceConfig) {
	s.Environment = composeType.NewMappingWithEquals(containerJSON.Config.Env)
}

func formatNetworkServiceConfig(containerJSON container.InspectResponse, s *composeType.ServiceConfig) {
	s.NetworkMode = string(containerJSON.HostConfig.NetworkMode)
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
}

func formatVolumeServiceConfig(containerJSON container.InspectResponse, s *composeType.ServiceConfig) {
	for _, containerVolume := range containerJSON.Mounts {
		var v composeType.ServiceVolumeConfig
		v.Type = string(containerVolume.Type)
		v.Source = containerVolume.Source
		v.Target = containerVolume.Destination
		v.ReadOnly = !containerVolume.RW
		s.Volumes = append(s.Volumes, v)
	}
}
