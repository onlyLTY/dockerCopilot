package backupCompose

import (
	"testing"

	composeType "github.com/compose-spec/compose-go/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

func TestFormatVolumePreservesReadOnlyMeaning(t *testing.T) {
	containerJSON := container.InspectResponse{Mounts: []container.MountPoint{
		{Type: mount.TypeBind, Source: "/rw", Destination: "/data", RW: true},
		{Type: mount.TypeBind, Source: "/ro", Destination: "/config", RW: false},
	}}
	var service composeType.ServiceConfig
	formatVolumeServiceConfig(containerJSON, &service)
	if len(service.Volumes) != 2 || service.Volumes[0].ReadOnly || !service.Volumes[1].ReadOnly {
		t.Fatalf("volume permissions were reversed: %+v", service.Volumes)
	}
}

func TestFormatNetworkPreservesAllPortBindings(t *testing.T) {
	containerJSON := container.InspectResponse{ContainerJSONBase: &container.ContainerJSONBase{
		HostConfig: &container.HostConfig{PortBindings: nat.PortMap{
			nat.Port("8080/tcp"): {
				{HostIP: "127.0.0.1", HostPort: "18080"},
				{HostIP: "0.0.0.0", HostPort: "28080"},
			},
			nat.Port("9090/tcp"): {},
		}},
	}}
	var service composeType.ServiceConfig
	compose := composeYaml{Networks: make(map[string]composeType.NetworkConfig)}
	formatNetworkServiceConfig(containerJSON, &service, &compose)
	if len(service.Ports) != 3 {
		t.Fatalf("expected all bindings plus exposed port, got %+v", service.Ports)
	}
	hostIPs := map[string]bool{}
	for _, port := range service.Ports {
		if port.Target == 8080 {
			hostIPs[port.HostIP] = true
		}
	}
	if !hostIPs["127.0.0.1"] || !hostIPs["0.0.0.0"] {
		t.Fatalf("host IP was dropped: %+v", service.Ports)
	}
}

func TestSensitiveEnvironmentValuesAreOmittedByDefault(t *testing.T) {
	containerJSON := container.InspectResponse{Config: &container.Config{
		Env: []string{"APP_MODE=production", "POSTGRES_PASSWORD=super-secret"},
	}}
	var service composeType.ServiceConfig
	formatEnvServiceConfig(containerJSON, &service, Options{})
	if service.Environment["APP_MODE"] == nil || *service.Environment["APP_MODE"] != "production" {
		t.Fatalf("ordinary environment was not preserved: %+v", service.Environment)
	}
	if service.Environment["POSTGRES_PASSWORD"] != nil {
		t.Fatal("sensitive environment value was written to plaintext compose output")
	}
}

func TestSensitiveEnvironmentCanBeExplicitlyIncluded(t *testing.T) {
	containerJSON := container.InspectResponse{Config: &container.Config{Env: []string{"API_TOKEN=value"}}}
	var service composeType.ServiceConfig
	formatEnvServiceConfig(containerJSON, &service, Options{IncludeSensitiveEnvironment: true})
	if service.Environment["API_TOKEN"] == nil || *service.Environment["API_TOKEN"] != "value" {
		t.Fatal("explicit secret inclusion was ignored")
	}
}

func TestFormatNetworkOnlyPinsExplicitIPAMAddresses(t *testing.T) {
	containerJSON := container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{ID: "abcdefabcdef1234", Name: "/app", HostConfig: &container.HostConfig{}},
		Config:            &container.Config{},
		NetworkSettings: &container.NetworkSettings{Networks: map[string]*network.EndpointSettings{
			"dynamic": {IPAddress: "172.20.0.2", GlobalIPv6Address: "fd00::2"},
			"static": {
				IPAddress:  "172.21.0.2",
				IPAMConfig: &network.EndpointIPAMConfig{IPv4Address: "172.21.0.20", IPv6Address: "fd01::20"},
			},
		}},
	}
	var service composeType.ServiceConfig
	compose := composeYaml{Networks: make(map[string]composeType.NetworkConfig)}
	formatNetworkServiceConfig(containerJSON, &service, &compose)
	if service.Networks["dynamic"].Ipv4Address != "" || service.Networks["dynamic"].Ipv6Address != "" {
		t.Fatalf("runtime-assigned addresses were exported as static: %+v", service.Networks["dynamic"])
	}
	if service.Networks["static"].Ipv4Address != "172.21.0.20" || service.Networks["static"].Ipv6Address != "fd01::20" {
		t.Fatalf("explicit IPAM addresses were lost: %+v", service.Networks["static"])
	}
}
