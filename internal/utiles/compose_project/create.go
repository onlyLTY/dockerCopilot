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
	if err := os.MkdirAll(projectRoot, 0750); err != nil {
		return "", "", err
	}
	version, err := SaveProjectFile(svcCtx, projectRoot, filename, content, "")
	if err != nil {
		os.Remove(projectRoot)
		return "", "", err
	}
	return ProjectID(projectRoot), version, nil
}
