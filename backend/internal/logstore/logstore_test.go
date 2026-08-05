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
	// gz 最新（刚写），然后 new，然后 old
	if err := os.Chtimes(gzPath, now, now); err != nil {
		t.Fatal(err)
	}
	entries, err := ReadRecent(4, "")
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

func TestReadRecentParsesFormattedLogBlocks(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOG_DIR", dir)
	path := filepath.Join(dir, "access.log")
	content := "【INFO】2026-07-29T07:00:00Z\n第一行\n第二行\n--------------------------------\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadRecent(1, "all")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %#v", entries)
	}
	want := Entry{Timestamp: "2026-07-29T07:00:00Z", Level: "info", Message: "第一行\n第二行"}
	if entries[0] != want {
		t.Fatalf("expected %#v, got %#v", want, entries[0])
	}
}

func TestReadRecentReturnsEmptyForMissingDirectory(t *testing.T) {
	t.Setenv("LOG_DIR", filepath.Join(t.TempDir(), "missing"))
	entries, err := ReadRecent(10, "")
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

func TestReadRecentFiltersByLevel(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOG_DIR", dir)
	path := filepath.Join(dir, "mixed.log")
	content := "" +
		"2026-07-29T01:00:00Z\tinfo\ti1\n" +
		"2026-07-29T02:00:00Z\terror\te1\n" +
		"2026-07-29T03:00:00Z\tinfo\ti2\n" +
		"2026-07-29T04:00:00Z\terror\te2\n" +
		"2026-07-29T05:00:00Z\twarn\tw1\n" +
		"2026-07-29T06:00:00Z\terror\te3\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	entries, err := ReadRecent(10, "error")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 errors, got %#v", entries)
	}
	// 最新 error 在前
	if entries[0].Message != "e3" || entries[1].Message != "e2" || entries[2].Message != "e1" {
		t.Fatalf("unexpected order: %#v", entries)
	}
	// limit 截断在过滤之后
	limited, err := ReadRecent(2, "error")
	if err != nil {
		t.Fatal(err)
	}
	if len(limited) != 2 || limited[0].Message != "e3" || limited[1].Message != "e2" {
		t.Fatalf("expected newest 2 errors, got %#v", limited)
	}
}
