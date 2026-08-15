package compose_project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/compose-spec/compose-go/v2/loader"
	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/utiles"
	"gopkg.in/yaml.v3"
)

var projectNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)

func ProjectID(root string) string {
	sum := sha256.Sum256([]byte(filepath.Clean(root)))
	return hex.EncodeToString(sum[:8])
}

func ValidateProjectName(name string) error {
	if !projectNamePattern.MatchString(name) {
		return fmt.Errorf("项目名称只能包含小写字母、数字、短横线和下划线，长度不超过 63")
	}
	return nil
}

func ValidateComposeFilename(filename string) error {
	if filename == "" || filepath.Base(filename) != filename || strings.ContainsAny(filename, "/\\") || strings.Contains(filename, "..") {
		return fmt.Errorf("Compose 文件名不合法")
	}
	lower := strings.ToLower(filename)
	if lower == ".env" {
		return nil
	}
	if !IsComposeFile(filename) {
		return fmt.Errorf("不支持的 Compose 文件名")
	}
	return nil
}

func FindProjectRoot(svcCtx *svc.ServiceContext, id string) (string, error) {
	for _, root := range svcCtx.Config.Compose.ScanPaths {
		rootAbs, err := filepath.Abs(root)
		if err != nil {
			return "", err
		}
		if projectRoot, err := findProjectRoot(rootAbs, id); err == nil {
			return projectRoot, nil
		}
	}
	return "", fmt.Errorf("Compose 项目不存在")
}

// RemoveProjectRoot 只删除配置的 Compose 扫描目录下的真实项目目录，
// 不允许删除扫描根目录或符号链接目录。
func RemoveProjectRoot(root string, scanPaths []string) error {
	rootAbs, err := validateProjectRootForRemoval(root, scanPaths)
	if err != nil {
		return err
	}
	return os.RemoveAll(rootAbs)
}

func validateProjectRootForRemoval(root string, scanPaths []string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rootAbs = filepath.Clean(rootAbs)
	info, err := os.Lstat(rootAbs)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("项目目录不是安全的真实目录")
	}
	rootReal, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", err
	}
	rootReal = filepath.Clean(rootReal)

	for _, scanPath := range scanPaths {
		if strings.TrimSpace(scanPath) == "" {
			continue
		}
		scanAbs, err := filepath.Abs(scanPath)
		if err != nil {
			return "", err
		}
		scanReal, err := filepath.EvalSymlinks(filepath.Clean(scanAbs))
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(filepath.Clean(scanReal), rootReal)
		if err != nil {
			continue
		}
		if rel == "." || rel == "" {
			return "", fmt.Errorf("不能删除 Compose 扫描根目录")
		}
		if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
			return rootAbs, nil
		}
	}
	return "", fmt.Errorf("项目目录不在 Compose 扫描范围内")
}

func findProjectRoot(root, id string) (string, error) {
	var found string
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() && ProjectID(path) == id {
			found = path
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("Compose 项目不存在")
	}
	return found, nil
}

func ProjectFilePath(root, filename string) (string, error) {
	if err := ValidateComposeFilename(filename); err != nil {
		return "", err
	}
	path := filepath.Join(root, filename)
	if err := ensurePathWithin(root, path); err != nil {
		return "", err
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("不允许通过符号链接访问 Compose 文件")
	}
	return path, nil
}

func SaveProjectFile(svcCtx *svc.ServiceContext, root, filename, content, expectedVersion string) (string, error) {
	if int64(len(content)) > svcCtx.Config.Compose.MaxFileSize && svcCtx.Config.Compose.MaxFileSize > 0 {
		return "", fmt.Errorf("Compose 文件超过大小限制")
	}
	path, err := ProjectFilePath(root, filename)
	if err != nil {
		return "", err
	}
	if current, err := os.ReadFile(path); err == nil {
		version := ContentVersion(current)
		if expectedVersion != "" && expectedVersion != version {
			return "", fmt.Errorf("文件已被其他操作修改，请重新加载")
		}
	} else if !os.IsNotExist(err) {

		return "", err
	}
	if _, err := ParseComposeContent(root, filename, []byte(content)); err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0750); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(root, ".compose-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return "", err
	}
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return "", err
	}
	return ContentVersion([]byte(content)), nil
}

func ReadProjectFile(root, filename string) ([]byte, string, error) {
	return ReadProjectFileWithLimit(root, filename, 0)
}

func ReadProjectFileWithLimit(root, filename string, maxSize int64) ([]byte, string, error) {
	path, err := ProjectFilePath(root, filename)
	if err != nil {
		return nil, "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", err
	}
	if maxSize > 0 && info.Size() > maxSize {
		return nil, "", fmt.Errorf("Compose 文件超过大小限制")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	return content, ContentVersion(content), nil
}

func ListProjectFiles(root string) ([]map[string]interface{}, error) {
	return ListProjectFilesWithLimit(root, 0)
}

func ListProjectFilesWithLimit(root string, maxSize int64) ([]map[string]interface{}, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	files := make([]map[string]interface{}, 0)
	for _, entry := range entries {
		if entry.IsDir() || (!IsComposeFile(entry.Name()) && !IsComposeEnvFile(entry.Name())) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		if maxSize > 0 && info.Size() > maxSize {
			return nil, fmt.Errorf("文件 %s 超过大小限制", entry.Name())
		}
		content, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			return nil, err
		}
		files = append(files, map[string]interface{}{"name": entry.Name(), "size": info.Size(), "modifiedAt": info.ModTime(), "version": ContentVersion(content)})
	}
	sort.Slice(files, func(i, j int) bool { return files[i]["name"].(string) < files[j]["name"].(string) })
	return files, nil
}

func ParseComposeContent(root, filename string, content []byte) (*composeTypes.Project, error) {
	return ParseComposeContentWithContext(context.Background(), root, filename, content)
}

func ParseComposeContentWithContext(ctx context.Context, root, filename string, content []byte) (*composeTypes.Project, error) {
	if err := ValidateComposeFilename(filename); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := yaml.Unmarshal(content, &raw); err != nil {
		// 不把底层 yaml 细节回传给前端，日志由调用方记录
		return nil, fmt.Errorf("YAML 解析失败，请检查语法")
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("Compose 文件不能为空")
	}
	if _, ok := raw["services"]; !ok {
		return nil, fmt.Errorf("Compose 文件必须包含 services")
	}
	projectName := ""
	if v, ok := raw["name"].(string); ok {
		projectName = v
	}
	if projectName == "" {
		projectName = filepath.Base(root)
	}
	projectName = loader.NormalizeProjectName(projectName)
	if projectName == "" {
		projectName = "app"
	}
	return loader.LoadWithContext(ctx, composeTypes.ConfigDetails{
		WorkingDir:  root,
		ConfigFiles: []composeTypes.ConfigFile{{Filename: filepath.Join(root, filename), Content: content}},
		Environment: composeTypes.Mapping{},
	}, func(o *loader.Options) {
		o.SetProjectName(projectName, true)
	})
}

func ContentVersion(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func BackupProject(svcCtx *svc.ServiceContext, root string) (map[string]interface{}, error) {
	composeFilename, err := findBackupComposeFilename(root)
	if err != nil {
		return nil, err
	}
	composePath, err := ProjectFilePath(root, composeFilename)
	if err != nil {
		return nil, err
	}
	composeContent, err := os.ReadFile(composePath)
	if err != nil {
		return nil, err
	}
	project, err := ParseComposeContent(root, composeFilename, composeContent)
	if err != nil {
		return nil, err
	}
	name := backupName(project)
	backupRoot := svcCtx.Config.Compose.BackupDir
	if backupRoot == "" {
		backupRoot = filepath.Join(utiles.BackupDirectory(), "compose-projects")
	}
	backupDir := backupRoot
	optional := []string{}
	for _, filename := range []string{".env", "config.yaml"} {
		path := filepath.Join(root, filename)
		if info, statErr := os.Lstat(path); statErr == nil && !info.IsDir() {
			if info.Mode()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("不允许备份符号链接文件：%s", filename)
			}
			optional = append(optional, filename)
		} else if statErr != nil && !os.IsNotExist(statErr) {
			return nil, statErr
		}
	}
	if len(optional) == 2 {
		backupDir = filepath.Join(backupRoot, ProjectID(root))
	}
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return nil, err
	}
	backupFilename := name + "_compose.yaml"
	if err := os.WriteFile(filepath.Join(backupDir, backupFilename), composeContent, 0600); err != nil {
		return nil, err
	}
	for _, filename := range optional {
		content, err := os.ReadFile(filepath.Join(root, filename))
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(backupDir, filename), content, 0600); err != nil {
			return nil, err
		}
	}
	files := []string{backupFilename}
	files = append(files, optional...)
	return map[string]interface{}{
		"path":     backupDir,
		"filename": backupFilename,
		"files":    files,
		"version":  ContentVersion(composeContent),
	}, nil
}

func findBackupComposeFilename(root string) (string, error) {
	for _, filename := range []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"} {
		path := filepath.Join(root, filename)
		info, err := os.Lstat(path)
		if err == nil {
			if info.IsDir() {
				return "", fmt.Errorf("Compose 文件不能是目录：%s", filename)
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("不允许备份符号链接文件：%s", filename)
			}
			return filename, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("未找到 Compose 主文件（支持 compose.yaml、compose.yml、docker-compose.yaml、docker-compose.yml）")
}
func backupName(project *composeTypes.Project) string {
	name := ""
	for _, service := range project.Services {
		containerName := strings.TrimSpace(service.ContainerName)
		if containerName == "" {
			continue
		}
		if name != "" && name != containerName {
			return normalizedBackupName(project.Name)
		}
		name = containerName
	}
	if name == "" {
		name = project.Name
	}
	return normalizedBackupName(name)
}

func normalizedBackupName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "/")
	if name == "" {
		return "app"
	}
	name = regexp.MustCompile(`[^a-zA-Z0-9._-]+`).ReplaceAllString(name, "-")
	name = strings.Trim(name, ".-")
	if name == "" || name == "." || name == ".." {
		return "app"
	}
	return name
}

func ensurePathWithin(root, target string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("路径超出项目目录")
	}
	return nil
}
