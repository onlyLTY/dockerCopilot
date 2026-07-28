package utiles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func BackupDirectory() string {
	if dir := os.Getenv("BACKUP_DIR"); dir != "" {
		return dir
	}
	return "/data/backups"
}

func ResolveBackupPath(filename string) (string, error) {
	if filename == "" || strings.IndexByte(filename, 0) >= 0 {
		return "", fmt.Errorf("invalid backup filename")
	}
	if filepath.Base(filename) != filename || filepath.IsAbs(filename) || strings.ContainsAny(filename, "/\\") || strings.Contains(filename, "..") {
		return "", fmt.Errorf("invalid backup filename")
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".json" && ext != ".yaml" && ext != ".yml" {
		return "", fmt.Errorf("unsupported backup file type")
	}

	base, err := filepath.Abs(BackupDirectory())
	if err != nil {
		return "", err
	}
	target := filepath.Join(base, filename)
	if err := ensurePathWithin(base, target); err != nil {
		return "", err
	}

	if resolved, err := filepath.EvalSymlinks(target); err == nil {
		if err := ensurePathWithin(base, resolved); err != nil {
			return "", fmt.Errorf("backup path escapes backup directory")
		}
	}
	return target, nil
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
		return fmt.Errorf("path escapes allowed directory")
	}
	return nil
}
