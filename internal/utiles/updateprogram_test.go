package utiles

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecompressTarGzRejectsTraversal(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "update.tar.gz")
	writeTestTarGz(t, archivePath, "../outside", "owned")
	destination := t.TempDir()

	if err := decompressTarGz(archivePath, destination); err == nil {
		t.Fatal("expected traversal archive to be rejected")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(destination), "outside")); !os.IsNotExist(err) {
		t.Fatalf("archive escaped destination: %v", err)
	}
}

func TestDecompressTarGzExtractsRegularFile(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "update.tar.gz")
	writeTestTarGz(t, archivePath, "dockerCopilot-new", "binary")
	destination := t.TempDir()

	if err := decompressTarGz(archivePath, destination); err != nil {
		t.Fatalf("decompressTarGz returned error: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(destination, "dockerCopilot-new"))
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(content) != "binary" {
		t.Fatalf("unexpected extracted content %q", content)
	}
}

func TestVerifySHA256(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "archive")
	checksumPath := filepath.Join(dir, "archive.sha256")
	content := []byte("verified update")
	if err := os.WriteFile(archivePath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(content)
	if err := os.WriteFile(checksumPath, []byte(fmt.Sprintf("%x  archive\n", digest)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifySHA256(archivePath, checksumPath); err != nil {
		t.Fatalf("valid checksum rejected: %v", err)
	}
	if err := os.WriteFile(archivePath, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifySHA256(archivePath, checksumPath); err == nil {
		t.Fatal("tampered archive accepted")
	}
}

func TestDownloadFileEnforcesLimitAndStatus(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/large":
			_, _ = w.Write([]byte(strings.Repeat("x", 9)))
		default:
			http.Error(w, "missing", http.StatusNotFound)
		}
	}))
	defer server.Close()

	largePath := filepath.Join(t.TempDir(), "large")
	if err := downloadFile(context.Background(), server.Client(), server.URL+"/large", largePath, 8); err == nil {
		t.Fatal("oversized response accepted")
	}
	if _, err := os.Stat(largePath); !os.IsNotExist(err) {
		t.Fatalf("partial oversized file was not removed: %v", err)
	}
	if err := downloadFile(context.Background(), server.Client(), server.URL+"/missing", filepath.Join(t.TempDir(), "missing"), 8); err == nil {
		t.Fatal("non-200 response accepted")
	}
}

func writeTestTarGz(t *testing.T, path, name, content string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
