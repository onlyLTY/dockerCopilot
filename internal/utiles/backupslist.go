package utiles

import (
	"github.com/onlyLTY/oneKeyUpdate/zspace/internal/svc"
	"os"
	"path/filepath"
)

func BackupList(ctx *svc.ServiceContext) ([]string, error) {
	var backupList []string
	dir := `D:\MyProject\oneKeyUpdateGo` // 指定您的目录
	entries, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			backupList = append(backupList, entry.Name())
		}
	}

	return backupList, nil
}
