package backupstore

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/datadir"
	"github.com/onlyLTY/dockerCopilot/internal/settingstore"
	"github.com/zeromicro/go-zero/core/logx"
)

func Directory() string {
	if dir := os.Getenv("BACKUP_DIR"); dir != "" {
		return dir
	}
	return datadir.BackupsDir()
}

func GetRetention() (int, error) {
	return settingstore.GetRetention(), nil
}

func SetRetention(value int) (int, error) {
	retention, err := settingstore.SetRetention(value)
	if err != nil {
		return 0, err
	}
	if err := Retain(retention); err != nil {
		logx.Errorf("保存备份保留设置成功，但清理旧备份失败: %v", err)
	}
	return retention, nil
}

func Retain(limit int) error {
	if !settingstore.ValidRetention(limit) {
		limit = settingstore.GetRetention()
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
		if len(files) <= limit {
			continue
		}
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
	var suffix [3]byte
	if _, err := io.ReadFull(rand.Reader, suffix[:]); err != nil {
		return "backup-" + time.Now().UTC().Format("20060102T150405") + "-" + fmt.Sprintf("%06x", uint64(time.Now().UnixNano())&0xffffff) + ext
	}
	return "backup-" + time.Now().UTC().Format("20060102T150405") + "-" + hex.EncodeToString(suffix[:]) + ext
}

// CreateBackupFile 生成唯一文件名并写入备份；已存在则不覆盖（O_EXCL 重试）。
func CreateBackupFile(dir, ext string, content []byte, perm os.FileMode) (string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	for attempt := 0; attempt < 10; attempt++ {
		name := TimestampedName(ext)
		path := filepath.Join(dir, name)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			return "", err
		}
		_, writeErr := file.Write(content)
		closeErr := file.Close()
		if writeErr != nil {
			_ = os.Remove(path)
			return "", writeErr
		}
		if closeErr != nil {
			_ = os.Remove(path)
			return "", closeErr
		}
		return name, nil
	}
	return "", fmt.Errorf("无法生成唯一备份文件名")
}
