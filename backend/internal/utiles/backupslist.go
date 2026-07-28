package utiles

import (
	"os"
	"path/filepath"

	"github.com/onlyLTY/dockerCopilot/internal/backupstore"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func BackupList(ctx *svc.ServiceContext) ([]string, error) {
	var backupList []string
	dir := backupstore.Directory()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			backupList = append(backupList, entry.Name())
		} else if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yaml" {
			backupList = append(backupList, entry.Name())
		}
	}

	return backupList, nil
}
