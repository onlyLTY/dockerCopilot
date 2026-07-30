package logwriter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriterFormatsLogEntry(t *testing.T) {
	dir := t.TempDir()
	writer, err := New(dir, 7, false)
	if err != nil {
		t.Fatal(err)
	}
	writer.Info("第一行\n第二行\n")
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "access.log"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "【INFO】") {
		t.Fatalf("expected level and timestamp on one line, got %q", text)
	}
	if !strings.Contains(text, "\n第一行\n第二行\n--------------------------------\n") {
		t.Fatalf("expected multiline content and separator, got %q", text)
	}
}
