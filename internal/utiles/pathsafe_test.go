package utiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBackupPathRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BACKUP_DIR", dir)
	for _, name := range []string{"../outside.json", `..\outside.json`, "/tmp/outside.json", "backup.txt", "foo..json"} {
		if _, err := ResolveBackupPath(name); err == nil {
			t.Fatalf("expected %q to be rejected", name)
		}
	}
}

func TestResolveBackupPathAllowsRegularBackup(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BACKUP_DIR", dir)
	path, err := ResolveBackupPath("backup-2026-01-01.json")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(path) != dir {
		t.Fatalf("unexpected path: %s", path)
	}
}

func TestResolveBackupPathRejectsSymlinkOutside(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	t.Setenv("BACKUP_DIR", dir)
	target := filepath.Join(outside, "backup.json")
	if err := os.WriteFile(target, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "backup.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := ResolveBackupPath("backup.json"); err == nil {
		t.Fatal("expected outside symlink to be rejected")
	}
}
