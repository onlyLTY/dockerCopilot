package utiles

import (
	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func BackupList(ctx *svc.ServiceContext) ([]string, error) {
	_ = ctx
	var backupList []string
	dir, err := ensureBackupDir()
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	directory, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	entries, err := directory.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if entry.Type().IsRegular() && (extension == ".json" || extension == ".yaml") {
			backupList = append(backupList, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(backupList)))
	return backupList, nil
}
