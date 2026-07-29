package utiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupListSupportsLegacyShortAndCaseInsensitiveNames(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BACKUP_DIR", dir)
	for _, name := range []string{
		"backup-2026-01-01T00-00-00.000000000Z.json",
		"backup-20260101T000000-abcdef.json",
		"backup-20260101T000000-abcdef.yaml",
		"backup-20260101T000000-abcdef.yml",
		"backup-20260101T000000-fedcba.JSON",
		"not-a-backup.txt",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	files, err := BackupList(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 5 {
		t.Fatalf("expected five backup files, got %d: %v", len(files), files)
	}
}
