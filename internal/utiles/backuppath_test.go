package utiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadBackupDownloadValidatesPathAndContent(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("BACKUP_DIR", directory)
	filename := "backup-test.yaml"
	if err := os.WriteFile(filepath.Join(directory, filename), []byte("services: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	content, returnedName, err := ReadBackupDownload(filename)
	if err != nil {
		t.Fatal(err)
	}
	if returnedName != filename || string(content) != "services: {}\n" {
		t.Fatalf("unexpected download: name=%q content=%q", returnedName, content)
	}
	if _, _, err := ReadBackupDownload("../outside.yaml"); err == nil {
		t.Fatal("path traversal download was accepted")
	}
}
