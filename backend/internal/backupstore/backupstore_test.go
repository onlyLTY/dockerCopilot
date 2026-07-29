package backupstore

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestTimestampedNameUsesShortUniqueFormat(t *testing.T) {
	pattern := regexp.MustCompile(`^backup-\d{8}T\d{6}-[0-9a-f]{6}\.json$`)
	first := TimestampedName(".json")
	second := TimestampedName(".json")
	if !pattern.MatchString(first) || !pattern.MatchString(second) {
		t.Fatalf("unexpected backup names: %q, %q", first, second)
	}
	if first == second {
		t.Fatalf("expected unique names, got %q twice", first)
	}
}

func TestCreateBackupFileDoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	name, err := CreateBackupFile(dir, ".json", []byte(`{"ok":true}`), 0600)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != `{"ok":true}` {
		t.Fatalf("unexpected content: %s", content)
	}
	if !strings.HasPrefix(name, "backup-") {
		t.Fatalf("unexpected name: %s", name)
	}
}
