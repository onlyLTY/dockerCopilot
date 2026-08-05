package utiles

import (
	"path/filepath"
	"strings"
)

// IsDockerSocketPath 判断路径是否指向 Docker socket（含路径片段匹配）。
func IsDockerSocketPath(path string) bool {
	return strings.Contains(filepath.ToSlash(path), "docker.sock")
}

// IsSensitiveHostPath 判定是否为极高危敏感宿主机路径。
// 与 Compose 部署门禁、备份恢复拦截共用同一规则。
// 比较统一用 ToSlash，避免 Windows 上 Clean("/") 变成 "\\" 导致漏判。
func IsSensitiveHostPath(path string) bool {
	slash := filepath.ToSlash(filepath.Clean(path))
	switch slash {
	case "/", "/etc", "/proc", "/sys", "/var/run", "/var/run/docker.sock":
		return true
	default:
		return strings.HasPrefix(slash, "/etc/") ||
			strings.HasPrefix(slash, "/proc/") ||
			strings.HasPrefix(slash, "/sys/")
	}
}

// BindSource 从 Docker bind 字符串 host:container[:mode] 取宿主机源路径。
// Linux 备份/Compose 为主；Windows 盘符场景不在此展开。
func BindSource(bind string) string {
	parts := strings.Split(bind, ":")
	if len(parts) == 0 {
		return bind
	}
	return parts[0]
}
