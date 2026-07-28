package backupstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultRetention = 10
	minRetention     = 1
	maxRetention     = 100
)

type Settings struct {
	Retention int `json:"retention"`
}

func Directory() string {
	if dir := os.Getenv("BACKUP_DIR"); dir != "" {
		return dir
	}
	return "/data/backups"
}

func SettingsPath() string { return "/data/config/backupSettings.json" }

func GetRetention() (int, error) {
	content, err := os.ReadFile(SettingsPath())
	if os.IsNotExist(err) {
		return defaultRetention, nil
	}
	if err != nil {
		return 0, err
	}
	var settings Settings
	if err := json.Unmarshal(content, &settings); err != nil {
		return 0, err
	}
	return clamp(settings.Retention), nil
}

func SetRetention(value int) (int, error) {
	if value < minRetention || value > maxRetention {
		return 0, fmt.Errorf("备份保留数量必须在 %d-%d 之间", minRetention, maxRetention)
	}
	if err := os.MkdirAll(filepath.Dir(SettingsPath()), 0755); err != nil {
		return 0, err
	}
	content, err := json.MarshalIndent(Settings{Retention: value}, "", "  ")
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(SettingsPath(), content, 0644); err != nil {
		return 0, err
	}
	if err := Retain(value); err != nil {
		return 0, err
	}
	return value, nil
}

func Retain(limit int) error {
	if limit < minRetention {
		limit = defaultRetention
	}
	entries, err := os.ReadDir(Directory())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	byExt := map[string][]os.DirEntry{".json": {}, ".yaml": {}, ".yml": {}}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "backup-") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if _, ok := byExt[ext]; ok {
			byExt[ext] = append(byExt[ext], entry)
		}
	}
	for ext, files := range byExt {
		sort.Slice(files, func(i, j int) bool { return files[i].Name() > files[j].Name() })
		for _, entry := range files[limit:] {
			if err := os.Remove(filepath.Join(Directory(), entry.Name())); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		_ = ext
	}
	return nil
}

func TimestampedName(ext string) string {
	return "backup-" + time.Now().UTC().Format("2006-01-02T15-04-05.000000000Z") + ext
}

func clamp(value int) int {
	if value < minRetention || value > maxRetention {
		return defaultRetention
	}
	return value
}
