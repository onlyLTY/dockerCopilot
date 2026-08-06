package utiles

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

func TestValidateRestoreHostConfigCritical(t *testing.T) {
	cases := []struct {
		name string
		hc   *container.HostConfig
	}{
		{"privileged", &container.HostConfig{Privileged: true}},
		{"docker sock bind", &container.HostConfig{Binds: []string{"/var/run/docker.sock:/var/run/docker.sock"}}},
		{"sensitive bind", &container.HostConfig{Binds: []string{"/etc/passwd:/etc/passwd:ro"}}},
		{"docker sock mount", &container.HostConfig{Mounts: []mount.Mount{{Type: mount.TypeBind, Source: "/run/docker.sock", Target: "/var/run/docker.sock"}}}},
		{"sensitive mount", &container.HostConfig{Mounts: []mount.Mount{{Type: mount.TypeBind, Source: "/etc", Target: "/host-etc"}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateRestoreHostConfig("app", tc.hc, false); err == nil {
				t.Fatal("expected block without AllowHighRisk")
			}
			if err := validateRestoreHostConfig("app", tc.hc, true); err != nil {
				t.Fatalf("AllowHighRisk should permit: %v", err)
			}
		})
	}
}

func TestValidateRestoreHostConfigHighRisk(t *testing.T) {
	cases := []struct {
		name string
		hc   *container.HostConfig
	}{
		{"host network", &container.HostConfig{NetworkMode: "host"}},
		{"host pid", &container.HostConfig{PidMode: "host"}},
		{"devices", &container.HostConfig{Resources: container.Resources{Devices: []container.DeviceMapping{{PathOnHost: "/dev/null"}}}}},
		{"cap add", &container.HostConfig{CapAdd: []string{"SYS_ADMIN"}}},
		{"security opt", &container.HostConfig{SecurityOpt: []string{"seccomp=unconfined"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateRestoreHostConfig("app", tc.hc, false); err == nil {
				t.Fatal("expected high-risk block without AllowHighRisk")
			}
			if err := validateRestoreHostConfig("app", tc.hc, true); err != nil {
				t.Fatalf("AllowHighRisk should permit: %v", err)
			}
		})
	}
}

func TestValidateRestoreHostConfigSafe(t *testing.T) {
	hc := &container.HostConfig{
		Binds:         []string{"/data/app:/data:ro"},
		NetworkMode:   "bridge",
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}
	if err := validateRestoreHostConfig("app", hc, false); err != nil {
		t.Fatalf("safe config should pass: %v", err)
	}
	if err := validateRestoreHostConfig("app", nil, false); err != nil {
		t.Fatalf("nil host config should pass: %v", err)
	}
}
