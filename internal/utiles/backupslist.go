package utiles

import (
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"os"
	"path/filepath"
)

func BackupList(ctx *svc.ServiceContext) ([]string, error) {
	var backupList []string
	dir := `/data/backups` // 指定您的目录
	entries, err := os.ReadDir(dir)
	if err != nil {
		logx.Error(err)
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			backupList = append(backupList, entry.Name())
		}
	}

	return backupList, nil
}
