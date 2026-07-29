package logstore

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadRecentParsesPlainJSONAndGzipLogs(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOG_DIR", dir)
	oldPath := filepath.Join(dir, "old.log")
	newPath := filepath.Join(dir, "new.log")
	if err := os.WriteFile(oldPath, []byte("2026-07-29T04:00:00Z\tinfo\told\nnot structured\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte(`{"@timestamp":"2026-07-29T05:00:00Z","content":"new"}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gzPath := filepath.Join(dir, "rotated.log.gz")
	file, err := os.Create(gzPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := gzip.NewWriter(file)
	_, _ = writer.Write([]byte("2026-07-29T06:00:00Z\tdebug\tgz\n"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := os.Chtimes(oldPath, now.Add(-2*time.Hour), now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newPath, now.Add(-time.Hour), now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	entries, err := ReadRecent(4)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 4 {
		t.Fatalf("expected four entries, got %#v", entries)
	}
	if entries[0].Message != "gz" || entries[1].Message != "new" || entries[2].Message != "not structured" || entries[3].Message != "old" {

		t.Fatalf("unexpected recent entries: %#v", entries)
	}
}

func TestReadRecentReturnsEmptyForMissingDirectory(t *testing.T) {
	t.Setenv("LOG_DIR", filepath.Join(t.TempDir(), "missing"))
	entries, err := ReadRecent(10)
	if err != nil {
		t.Fatal(err)
	}
	if entries == nil {
		t.Fatal("expected non-nil entries")
	}
	if len(entries) != 0 {
		t.Fatalf("expected no entries, got %#v", entries)
	}
}
