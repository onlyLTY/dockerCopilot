package utiles

import (
	"strings"

	"github.com/docker/docker/api/types/container"
)

const generatedHostnameLength = 12

func clearGeneratedHostname(config *container.Config, containerID string) {
	if config == nil {
		return
	}
	containerID = strings.TrimPrefix(strings.TrimSpace(containerID), "sha256:")
	if len(containerID) >= generatedHostnameLength && config.Hostname == containerID[:generatedHostnameLength] {
		config.Hostname = ""
	}
}
