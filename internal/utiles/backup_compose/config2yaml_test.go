package backupCompose

import (
	"testing"

	composeType "github.com/compose-spec/compose-go/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
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
	formatNetworkServiceConfig(containerJSON, &service)
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
