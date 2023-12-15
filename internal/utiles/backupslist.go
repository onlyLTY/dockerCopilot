package utiles

import (
	"github.com/onlyLTY/dockerCopilot/UGREEN/internal/svc"
	"os"
	"path/filepath"
)

func BackupList(ctx *svc.ServiceContext) ([]string, error) {
	var backupList []string
	dir := os.Getenv("BACKUP_DIR") // 从环境变量中获取备份目录
	if dir == "" {
		dir = "/data/backup" // 如果环境变量未设置，使用默认值
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			backupList = append(backupList, entry.Name()[:len(entry.Name())-5])
		}
	}

	return backupList, nil
}
