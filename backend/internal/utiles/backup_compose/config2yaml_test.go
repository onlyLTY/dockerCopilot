package backupCompose

import (
	"testing"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	dockerTypes "github.com/docker/docker/api/types"
	containerTypes "github.com/docker/docker/api/types/container"
	mountTypes "github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
)

func TestFormatNetworkServiceConfigHandlesEmptyBindings(t *testing.T) {
	port, err := nat.NewPort("tcp", "8080")
	if err != nil {
		t.Fatal(err)
	}
	containerJSON := dockerTypes.ContainerJSON{
		ContainerJSONBase: &dockerTypes.ContainerJSONBase{HostConfig: &containerTypes.HostConfig{PortBindings: nat.PortMap{port: {}}}},
		Config:            &containerTypes.Config{},
	}
	var service composeTypes.ServiceConfig
	formatNetworkServiceConfig(containerJSON, &service)
	if len(service.Ports) != 1 || service.Ports[0].Published != "" {
		t.Fatalf("unexpected ports: %#v", service.Ports)
	}
}

func TestFormatVolumeServiceConfigPreservesWritableMount(t *testing.T) {
	containerJSON := dockerTypes.ContainerJSON{
		Mounts: []dockerTypes.MountPoint{{Type: mountTypes.TypeBind, Source: "/data", Destination: "/app", RW: true}},
	}
	var service composeTypes.ServiceConfig
	formatVolumeServiceConfig(containerJSON, &service)
	if len(service.Volumes) != 1 || service.Volumes[0].ReadOnly {
		t.Fatalf("writable mount was marked read-only: %#v", service.Volumes)
	}
}
