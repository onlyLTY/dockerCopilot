package compose_project

import (
	"fmt"
	"path/filepath"
	"strings"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
)

type Risk struct {
	Level   string `json:"level"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

func InspectRisks(project *composeTypes.Project, root string) []Risk {
	risks := make([]Risk, 0)
	for _, service := range project.Services {
		prefix := "services." + service.Name
		if service.Privileged {
			risks = append(risks, Risk{"high", prefix + ".privileged", "服务启用了 privileged"})
		}
		if service.NetworkMode == "host" {
			risks = append(risks, Risk{"high", prefix + ".network_mode", "服务使用 host 网络"})
		}
		if service.Pid == "host" {
			risks = append(risks, Risk{"high", prefix + ".pid", "服务共享宿主机 PID 命名空间"})
		}
		if len(service.Devices) > 0 {
			risks = append(risks, Risk{"high", prefix + ".devices", "服务使用宿主机设备"})
		}
		if len(service.CapAdd) > 0 {
			risks = append(risks, Risk{"high", prefix + ".cap_add", "服务增加 Linux capabilities"})
		}
		if len(service.SecurityOpt) > 0 {
			risks = append(risks, Risk{"high", prefix + ".security_opt", "服务配置了安全选项"})
		}
		if service.Build != nil {
			risks = append(risks, Risk{"warning", prefix + ".build", "服务包含构建上下文，部署前需要检查路径"})
		}
		for _, volume := range service.Volumes {
			if volume.Type != "bind" {
				continue
			}
			if isSensitiveHostPath(volume.Source) {
				risks = append(risks, Risk{"high", prefix + ".volumes", "服务挂载了敏感宿主机路径"})
			} else if !pathWithin(root, volume.Source) {
				risks = append(risks, Risk{"warning", prefix + ".volumes", "服务挂载路径位于项目目录之外"})
			}
			if strings.Contains(filepath.ToSlash(volume.Source), "docker.sock") {
				risks = append(risks, Risk{"high", prefix + ".volumes", "服务挂载了 Docker socket"})
			}
		}
	}
	return risks
}

func isSensitiveHostPath(path string) bool {
	switch filepath.Clean(path) {
	case "/", "/etc", "/proc", "/sys", "/var/run", "/var/run/docker.sock":
		return true
	default:
		return strings.HasPrefix(filepath.ToSlash(path), "/etc/") || strings.HasPrefix(filepath.ToSlash(path), "/proc/") || strings.HasPrefix(filepath.ToSlash(path), "/sys/")
	}
}

func pathWithin(root, path string) bool {
	if path == "" || !filepath.IsAbs(path) {
		return true
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// IsCriticalRisk 判定是否为极高危（默认即使 confirmWarnings 也不允许，需 AllowHighRisk）。
func IsCriticalRisk(risk Risk) bool {
	msg := strings.ToLower(risk.Message)
	return strings.Contains(msg, "docker socket") ||
		strings.Contains(msg, "privileged") ||
		strings.Contains(msg, "敏感宿主机路径")
}

// ValidateCriticalRisks 极高危硬拦：仅当 allowCritical 为 true（通常来自配置 AllowHighRisk）才放行。
func ValidateCriticalRisks(project *composeTypes.Project, root string, allowCritical bool) error {
	if allowCritical {
		return nil
	}
	for _, risk := range InspectRisks(project, root) {
		if IsCriticalRisk(risk) {
			return fmt.Errorf("存在极高危配置，需在服务端开启 AllowHighRisk 后才能部署: %s", risk.Message)
		}
	}
	return nil
}

func ValidateRisks(project *composeTypes.Project, root string, allowHighRisk bool) error {
	risks := InspectRisks(project, root)
	for _, risk := range risks {
		if risk.Level == "high" && !allowHighRisk {
			return fmt.Errorf("存在高风险配置: %s", risk.Message)
		}
	}
	return nil
}
