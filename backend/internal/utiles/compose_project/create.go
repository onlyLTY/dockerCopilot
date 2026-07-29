package compose_project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func CreateProject(svcCtx *svc.ServiceContext, name, filename, content string) (string, string, error) {
	if err := ValidateProjectName(name); err != nil {
		return "", "", err
	}
	if err := ValidateComposeFilename(filename); err != nil {
		return "", "", err
	}
	if len(svcCtx.Config.Compose.ScanPaths) == 0 || svcCtx.Config.Compose.ScanPaths[0] == "" {
		return "", "", fmt.Errorf("未配置 Compose 扫描目录")
	}
	root, err := filepath.Abs(svcCtx.Config.Compose.ScanPaths[0])
	if err != nil {
		return "", "", err
	}
	projectRoot := filepath.Join(root, name)
	if err := ensurePathWithin(root, projectRoot); err != nil {
		return "", "", err
	}
	if _, err := os.Stat(projectRoot); err == nil {
		return "", "", fmt.Errorf("Compose 项目已存在")
	} else if !os.IsNotExist(err) {
		return "", "", err
	}
	// 仅创建项目目录本身（比扫描根深一层）。项目名已被 ValidateProjectName 限制
	// 为单段名称，不会包含路径分隔符，因此无需 MkdirAll 递归创建父目录。
	// 递归创建会去 stat/mkdir 扫描根（如 /compose，通常是 bind mount 挂载点），
	// 而部分文件系统（Windows Docker Desktop 的 bind mount）对挂载点的 stat 行为
	// 不一致，导致 MkdirAll 冒出 "mkdir /compose: file exists"。改用单层 Mkdir 规避。
	if err := os.Mkdir(projectRoot, 0750); err != nil {
		return "", "", err
	}
	version, err := SaveProjectFile(svcCtx, projectRoot, filename, content, "")
	if err != nil {
		os.Remove(projectRoot)
		return "", "", err
	}
	return ProjectID(projectRoot), version, nil
}
