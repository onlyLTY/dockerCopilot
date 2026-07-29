package backupCompose

import (
	composeType "github.com/compose-spec/compose-go/v2/types"
	dockerTypes "github.com/docker/docker/api/types"
	composeNat "github.com/docker/go-connections/nat"
	"github.com/onlyLTY/dockerCopilot/internal/backupstore"
	"github.com/zeromicro/go-zero/core/logx"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strconv"
	"strings"
)

// DockerConfig2ComposeYaml 将docker config转换为docker-compose.yaml
func DockerConfig2ComposeYaml(containerJSONs []dockerTypes.ContainerJSON) (err error) {
	var c composeYaml
	c.Services = make(map[string]composeType.ServiceConfig, len(containerJSONs))
	for _, containerJSON := range containerJSONs {
		var s composeType.ServiceConfig
		formatBaseServiceConfig(containerJSON, &s)
		formatEnvServiceConfig(containerJSON, &s)
		formatNetworkServiceConfig(containerJSON, &s)
		formatVolumeServiceConfig(containerJSON, &s)
		c.Services[s.Name] = s
	}
	backupDir := backupstore.Directory()
	yamlData, yamlMarshalErr := yaml.Marshal(c)
	if yamlMarshalErr != nil {
		logx.Errorf("Error marshalling data err is: %v", yamlMarshalErr)
		return yamlMarshalErr
	}
	fileName, err := backupstore.CreateBackupFile(backupDir, ".yaml", yamlData, 0644)
	if err != nil {
		logx.Error("Error writing backup file:", err)
		return err
	}
	fullPath := filepath.Join(backupDir, fileName)
	retention, err := backupstore.GetRetention()
	if err != nil {
		logx.Errorf("Error reading backup retention after writing %s: %v", fullPath, err)
		return nil
	}
	if err := backupstore.Retain(retention); err != nil {
		logx.Errorf("Error applying backup retention after writing %s: %v", fullPath, err)
	}
	return nil
}

func formatBaseServiceConfig(containerJSON dockerTypes.ContainerJSON, s *composeType.ServiceConfig) {
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
	return
}

func formatEnvServiceConfig(containerJSON dockerTypes.ContainerJSON, s *composeType.ServiceConfig) {
	s.Environment = composeType.NewMappingWithEquals(containerJSON.Config.Env)
	return
}

func formatNetworkServiceConfig(containerJSON dockerTypes.ContainerJSON, s *composeType.ServiceConfig) {
	s.NetworkMode = string(containerJSON.HostConfig.NetworkMode)
	for containerPort, bindings := range containerJSON.HostConfig.PortBindings {
		proto, port := composeNat.SplitProtoPort(string(containerPort))
		portNum, convertErr := strconv.Atoi(port)
		if convertErr != nil {
			logx.Errorf("Error converting port err is: %v", convertErr)
			continue
		}
		if len(bindings) == 0 {
			s.Ports = append(s.Ports, composeType.ServicePortConfig{Target: uint32(portNum), Protocol: proto})
			continue
		}
		for _, binding := range bindings {
			p := composeType.ServicePortConfig{Target: uint32(portNum), Published: binding.HostPort, Protocol: proto}
			if binding.HostIP != "" && binding.HostIP != "0.0.0.0" {
				p.HostIP = binding.HostIP
			}
			s.Ports = append(s.Ports, p)
		}
	}
}

func formatVolumeServiceConfig(containerJSON dockerTypes.ContainerJSON, s *composeType.ServiceConfig) {
	for _, containerVolume := range containerJSON.Mounts {
		var v composeType.ServiceVolumeConfig
		v.Type = string(containerVolume.Type)
		v.Source = containerVolume.Source
		v.Target = containerVolume.Destination
		v.ReadOnly = !containerVolume.RW

		s.Volumes = append(s.Volumes, v)
	}
}
