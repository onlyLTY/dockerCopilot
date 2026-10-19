package main

import (
	"os"
	"path/filepath"
	"testing"
)

// 回归：v2.2.7 自更新把 Angular 产物放进了 web/browser/，
// 旧实现只认 web/index.html，导致 /manager 返回 404。这里保证两种布局都能定位到前端根。
func TestFrontendRootHandlesStandardLayout(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := frontendRoot(root); got != root {
		t.Fatalf("frontendRoot() = %q, want %q", got, root)
	}
}

func TestFrontendRootFallsBackToNestedBrowserDir(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "browser")
	if err := os.MkdirAll(nested, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "index.html"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := frontendRoot(root); got != nested {
		t.Fatalf("frontendRoot() = %q, want nested %q", got, nested)
	}
}

func TestFrontendRootKeepsMissingLayout(t *testing.T) {
	root := t.TempDir()
	if got := frontendRoot(root); got != root {
		t.Fatalf("frontendRoot() = %q, want unchanged %q", got, root)
	}
}
