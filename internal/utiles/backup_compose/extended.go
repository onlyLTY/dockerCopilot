package backupCompose

import (
	"regexp"
	"sort"
	"strings"
	"time"

	composeType "github.com/compose-spec/compose-go/types"
	"github.com/docker/docker/api/types/container"
)

var sensitiveEnvironmentKeyPattern = regexp.MustCompile(`(?i)(password|passwd|secret|token|api[_-]?key|private[_-]?key|credential|cookie|session|auth)`)

func isSensitiveEnvironmentKey(key string) bool {
	return sensitiveEnvironmentKeyPattern.MatchString(strings.TrimSpace(key))
}

func durationPointer(value time.Duration) *composeType.Duration {
	if value <= 0 {
		return nil
	}
	duration := composeType.Duration(value)
	return &duration
}

func uint64Pointer(value int) *uint64 {
	if value <= 0 {
		return nil
	}
	converted := uint64(value)
	return &converted
}

func extraHosts(values []string) composeType.HostsList {
	result := make(composeType.HostsList)
	for _, value := range values {
		host, address, found := strings.Cut(value, "=")
		if !found {
			host, address, found = strings.Cut(value, ":")
		}
		if found && strings.TrimSpace(host) != "" && strings.TrimSpace(address) != "" {
			result[strings.TrimSpace(host)] = strings.TrimSpace(address)
		}
	}
	return result
}

func tmpfsList(values map[string]string) composeType.StringList {
	keys := make([]string, 0, len(values))
	for path := range values {
		keys = append(keys, path)
	}
	sort.Strings(keys)
	result := make(composeType.StringList, 0, len(values))
	for _, path := range keys {
		if options := strings.TrimSpace(values[path]); options != "" {
			result = append(result, path+":"+options)
		} else {
			result = append(result, path)
		}
	}
	return result
}

func formatDevices(devices []container.DeviceMapping) []string {
	result := make([]string, 0, len(devices))
	for _, device := range devices {
		value := device.PathOnHost + ":" + device.PathInContainer
		if device.CgroupPermissions != "" {
			value += ":" + device.CgroupPermissions
		}
		result = append(result, value)
	}
	return result
}

func formatUlimits(ulimits []*container.Ulimit) map[string]*composeType.UlimitsConfig {
	result := make(map[string]*composeType.UlimitsConfig, len(ulimits))
	for _, ulimit := range ulimits {
		if ulimit == nil || ulimit.Name == "" {
			continue
		}
		result[ulimit.Name] = &composeType.UlimitsConfig{Soft: int(ulimit.Soft), Hard: int(ulimit.Hard)}
	}
	return result
}

func filterGeneratedAliases(aliases []string, rawContainerName, containerID string) []string {
	containerName := strings.TrimPrefix(rawContainerName, "/")
	result := make([]string, 0, len(aliases))
	seen := make(map[string]struct{})
	for _, alias := range aliases {
		alias = strings.TrimSpace(alias)
		if alias == "" || alias == containerName || strings.HasPrefix(containerID, alias) || strings.HasPrefix(alias, containerID) {
			continue
		}
		if _, exists := seen[alias]; exists {
			continue
		}
		seen[alias] = struct{}{}
		result = append(result, alias)
	}
	return result
}
