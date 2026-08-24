package utiles

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var backupFilenamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func BackupDir() string {
	backupDir := os.Getenv("BACKUP_DIR")
	if backupDir == "" {
		backupDir = "/data/backups"
	}
	return backupDir
}

func backupDirectoryPath() (string, error) {
	return filepath.Abs(filepath.Clean(BackupDir()))
}

func ResolveBackupPath(filename string, allowedExts ...string) (string, error) {
	name := strings.TrimSpace(filename)
	if name == "" {
		return "", fmt.Errorf("非法文件名")
	}
	if name != filepath.Base(name) {
		return "", fmt.Errorf("非法文件名")
	}
	if strings.ContainsAny(name, `/\`) || !backupFilenamePattern.MatchString(name) {
		return "", fmt.Errorf("非法文件名")
	}

	if len(allowedExts) > 0 {
		ext := strings.ToLower(filepath.Ext(name))
		allowed := false
		for _, candidate := range allowedExts {
			if ext == strings.ToLower(candidate) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("非法文件名")
		}
	}

	base, err := backupDirectoryPath()
	if err != nil {
		return "", fmt.Errorf("备份目录无效: %w", err)
	}
	fullPath := filepath.Clean(filepath.Join(base, name))
	basePrefix := base + string(os.PathSeparator)
	if fullPath != base && !strings.HasPrefix(fullPath, basePrefix) {
		return "", fmt.Errorf("非法文件名")
	}

	return fullPath, nil
}

// ReadBackupDownload returns an authenticated download payload while reusing
// the same filename, symlink and size checks as restore operations.
func ReadBackupDownload(filename string) ([]byte, string, error) {
	fullPath, err := ResolveBackupPath(filename, ".json", ".yaml", ".yml")
	if err != nil {
		return nil, "", err
	}
	content, err := readBackupFile(fullPath)
	if err != nil {
		return nil, "", err
	}
	return content, filepath.Base(fullPath), nil
}
