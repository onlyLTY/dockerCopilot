package compose_project

import (
	"testing"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

func TestContainerPortsMergesDockerPortSources(t *testing.T) {
	bound, _ := nat.NewPort("tcp", "8080")
	exposed, _ := nat.NewPort("tcp", "9090")
	inspect := dockerTypes.ContainerJSON{
		ContainerJSONBase: &dockerTypes.ContainerJSONBase{HostConfig: &container.HostConfig{PortBindings: nat.PortMap{bound: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: "18080"}}}}},
		Config:            &container.Config{ExposedPorts: nat.PortSet{exposed: struct{}{}}},
		NetworkSettings:   &dockerTypes.NetworkSettings{NetworkSettingsBase: dockerTypes.NetworkSettingsBase{Ports: nat.PortMap{bound: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: "18080"}}}}},
	}
	ports := containerPorts(inspect)
	if len(ports) != 2 {
		t.Fatalf("expected two unique ports, got %#v", ports)
	}
	if ports[0].ContainerPort != "8080" || !ports[0].Published {
		t.Fatalf("expected published 8080 mapping, got %#v", ports[0])
	}
}

func TestContainerPortsIncludesHostBindingWithoutNetworkSettings(t *testing.T) {
	bound, _ := nat.NewPort("udp", "5353")
	inspect := dockerTypes.ContainerJSON{ContainerJSONBase: &dockerTypes.ContainerJSONBase{HostConfig: &container.HostConfig{PortBindings: nat.PortMap{bound: []nat.PortBinding{{HostPort: "15353"}}}}}}
	ports := containerPorts(inspect)
	if len(ports) != 1 || ports[0].ContainerPort != "5353" || ports[0].Protocol != "udp" {
		t.Fatalf("expected host binding fallback, got %#v", ports)
	}
}
