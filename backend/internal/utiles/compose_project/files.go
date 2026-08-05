package compose_project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

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
	if !composeFilenames[lower] && !strings.Contains(lower, "override") {
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
		if err := backupVersion(svcCtx, root, filename, current, version); err != nil {
			return "", err
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
	path, err := ProjectFilePath(root, filename)
	if err != nil {
		return nil, "", err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	return content, ContentVersion(content), nil
}

func ListProjectFiles(root string) ([]map[string]interface{}, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	files := make([]map[string]interface{}, 0)
	for _, entry := range entries {
		if entry.IsDir() || (!composeFilenames[strings.ToLower(entry.Name())] && !strings.Contains(strings.ToLower(entry.Name()), "override") && entry.Name() != ".env") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
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
	if err := ValidateComposeFilename(filename); err != nil {
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
	return loader.LoadWithContext(context.Background(), composeTypes.ConfigDetails{
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

func backupVersion(svcCtx *svc.ServiceContext, root, filename string, content []byte, version string) error {
	backupRoot := svcCtx.Config.Compose.BackupDir
	if backupRoot == "" {
		backupRoot = filepath.Join(utiles.BackupDirectory(), "compose-projects")
	}
	dir := filepath.Join(backupRoot, ProjectID(root), "versions")
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	name := stamp + "-" + filepath.Base(filename)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0600); err != nil {
		return err
	}
	manifest := map[string]interface{}{"filename": filename, "version": version, "createdAt": time.Now().UTC(), "size": len(content), "sha256": version}
	data, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	return os.WriteFile(path+".json", data, 0600)
}

func BackupProject(svcCtx *svc.ServiceContext, root string) (map[string]interface{}, error) {
	backupRoot := svcCtx.Config.Compose.BackupDir
	if backupRoot == "" {
		backupRoot = filepath.Join(utiles.BackupDirectory(), "compose-projects")
	}
	dir := filepath.Join(backupRoot, ProjectID(root), "cleanup", time.Now().UTC().Format("20060102T150405.000000000Z"))
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, err
	}
	files, err := ListProjectFiles(root)
	if err != nil {
		return nil, err
	}
	manifest := map[string]interface{}{"projectId": ProjectID(root), "createdAt": time.Now().UTC(), "files": files}
	for _, item := range files {
		name := item["name"].(string)
		content, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(dir, name), content, 0600); err != nil {
			return nil, err
		}
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0600); err != nil {
		return nil, err
	}
	return map[string]interface{}{"path": dir, "manifest": manifest}, nil
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
