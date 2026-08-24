package utiles

import (
	"os"
	"regexp"
	"strings"
	"sync"
)

var containerIDPattern = regexp.MustCompile(`(?i)(?:^|[^0-9a-f])([0-9a-f]{12,64})(?:[^0-9a-f]|$)`)
var selfContainerCandidatesOnce sync.Once
var cachedSelfContainerCandidates []string

// IsSelfContainerID reports whether id identifies the container in which
// dockerCopilot is running. Updating that container through its own process
// would stop the process before it can create the replacement.
func IsSelfContainerID(id string) bool {
	selfContainerCandidatesOnce.Do(func() {
		cachedSelfContainerCandidates = selfContainerIDCandidates()
	})
	return isSelfContainerID(id, cachedSelfContainerCandidates)
}

func isSelfContainerID(id string, candidates []string) bool {
	id = normalizeContainerID(id)
	if len(id) < 12 {
		return false
	}
	for _, candidate := range candidates {
		candidate = normalizeContainerID(candidate)
		if len(candidate) < 12 {
			continue
		}
		if strings.HasPrefix(id, candidate) || strings.HasPrefix(candidate, id) {
			return true
		}
	}
	return false
}

func selfContainerIDCandidates() []string {
	candidates := make([]string, 0, 6)
	if configured := strings.TrimSpace(os.Getenv("DOCKER_COPILOT_CONTAINER_ID")); configured != "" {
		candidates = append(candidates, configured)
	}
	if hostname, err := os.Hostname(); err == nil {
		candidates = append(candidates, hostname)
	}
	for _, path := range []string{"/proc/self/cgroup", "/proc/self/mountinfo"} {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, match := range containerIDPattern.FindAllStringSubmatch(string(content), -1) {
			if len(match) == 2 {
				candidates = append(candidates, match[1])
			}
		}
	}
	return candidates
}

func normalizeContainerID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "sha256:")
	if match := containerIDPattern.FindStringSubmatch(value); len(match) == 2 {
		return strings.ToLower(match[1])
	}
	return value
}
