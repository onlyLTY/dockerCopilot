package utiles

import (
	"testing"

	"github.com/docker/docker/api/types/container"
)

func TestClearGeneratedHostnamePreservesCustomValue(t *testing.T) {
	config := &container.Config{Hostname: "database.internal"}
	clearGeneratedHostname(config, "0123456789abcdef")
	if config.Hostname != "database.internal" {
		t.Fatalf("custom hostname was removed: %q", config.Hostname)
	}
}

func TestClearGeneratedHostnameRemovesContainerIDPrefix(t *testing.T) {
	config := &container.Config{Hostname: "0123456789ab"}
	clearGeneratedHostname(config, "0123456789abcdef")
	if config.Hostname != "" {
		t.Fatalf("generated hostname was retained: %q", config.Hostname)
	}
}
