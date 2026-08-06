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

func TestContainerPortsDeduplicatesExposedPortWhenPublished(t *testing.T) {
	bound, _ := nat.NewPort("tcp", "8080")
	inspect := dockerTypes.ContainerJSON{
		ContainerJSONBase: &dockerTypes.ContainerJSONBase{
			HostConfig: &container.HostConfig{PortBindings: nat.PortMap{bound: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: "18080"}}}},
		},
		Config: &container.Config{ExposedPorts: nat.PortSet{bound: struct{}{}}},
	}
	ports := containerPorts(inspect)
	if len(ports) != 1 || !ports[0].Published || ports[0].HostPort != "18080" {
		t.Fatalf("expected exposed port to be replaced by published binding, got %#v", ports)
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

func TestSameComposeRootUsesPathMapping(t *testing.T) {
	mapping := composePathMapping{source: "/data/docker-data", destination: "/compose"}
	labels := map[string]string{"com.docker.compose.project.working_dir": "/data/docker-data/dockercopilot"}
	if !sameComposeRoot(labels, "/compose/dockercopilot", []composePathMapping{mapping}) {
		t.Fatal("expected host and container compose roots to match")
	}
}

func TestSameComposeRootDoesNotMatchSiblingPath(t *testing.T) {
	mapping := composePathMapping{source: "/data/docker-data", destination: "/compose"}
	labels := map[string]string{"com.docker.compose.project.working_dir": "/data/docker-database/dockercopilot"}
	if sameComposeRoot(labels, "/compose/dockercopilot", []composePathMapping{mapping}) {
		t.Fatal("expected unrelated host root not to match")
	}
}

func TestSameComposeRootNormalizesWindowsSeparators(t *testing.T) {
	mapping := composePathMapping{source: `E:\\compose`, destination: "/compose"}
	labels := map[string]string{"com.docker.compose.project.working_dir": `E:\\compose\\dockercopilot`}
	if !sameComposeRoot(labels, "/compose/dockercopilot", []composePathMapping{mapping}) {
		t.Fatal("expected Windows host root to match mapped container root")
	}
}

func TestSameComposeRootRequiresWorkingDirectory(t *testing.T) {
	if sameComposeRoot(map[string]string{}, "/compose/dockercopilot", nil) {
		t.Fatal("expected missing working directory not to match")
	}
}
