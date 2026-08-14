package compose_project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onlyLTY/dockerCopilot/internal/config"
	"github.com/onlyLTY/dockerCopilot/internal/svc"
)

func TestBackupProjectWritesComposeOnlyAtRoot(t *testing.T) {
	root := t.TempDir()
	backupRoot := t.TempDir()
	compose := "name: cliproxyapi\nservices:\n  app:\n    image: example/app\n"
	if err := os.WriteFile(filepath.Join(root, "compose.yaml"), []byte(compose), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "compose.override.yaml"), []byte("ignored"), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := BackupProject(&svc.ServiceContext{Config: config.Config{Compose: config.ComposeConfig{BackupDir: backupRoot}}}, root)
	if err != nil {
		t.Fatal(err)
	}
	if result["filename"] != "cliproxyapi_compose.yaml" {
		t.Fatalf("unexpected backup filename: %#v", result["filename"])
	}
	content, err := os.ReadFile(filepath.Join(backupRoot, "cliproxyapi_compose.yaml"))
	if err != nil || string(content) != compose {
		t.Fatalf("unexpected backup content: %q, %v", content, err)
	}
	if _, err := os.Stat(filepath.Join(backupRoot, ProjectID(root))); !os.IsNotExist(err) {
		t.Fatalf("unexpected project backup directory: %v", err)
	}
}

func TestBackupProjectCopiesEnvAndConfigByProjectDirectory(t *testing.T) {
	root := t.TempDir()
	backupRoot := t.TempDir()
	files := map[string]string{
		"compose.yaml": "services:\n  app:\n    container_name: cliproxyapi\n    image: example/app\n",
		".env":         "APP_ENV=test\n",
		"config.yaml":  "port: 8080\n",
		"compose.yml":  "ignored\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	result, err := BackupProject(&svc.ServiceContext{Config: config.Config{Compose: config.ComposeConfig{BackupDir: backupRoot}}}, root)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(backupRoot, ProjectID(root))
	if result["filename"] != "cliproxyapi_compose.yaml" {
		t.Fatalf("unexpected backup filename: %#v", result["filename"])
	}
	for name, want := range map[string]string{
		"cliproxyapi_compose.yaml": files["compose.yaml"],
		".env":                     files[".env"],
		"config.yaml":              files["config.yaml"],
	} {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(content) != want {
			t.Fatalf("backup %s = %q, %v", name, content, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "compose.yml")); !os.IsNotExist(err) {
		t.Fatalf("unexpected override backup: %v", err)
	}
}

func TestBackupProjectCopiesSingleOptionalFileAtRoot(t *testing.T) {
	root := t.TempDir()
	backupRoot := t.TempDir()
	for name, content := range map[string]string{
		"compose.yaml": "services:\n  app:\n    image: example/app\n",
		".env":         "APP_ENV=test\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := BackupProject(&svc.ServiceContext{Config: config.Config{Compose: config.ComposeConfig{BackupDir: backupRoot}}}, root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(backupRoot, ".env")); err != nil {
		t.Fatalf("single optional file should be stored at root: %v", err)
	}
}

func TestBackupProjectFallsBackToProjectNameForMultipleContainerNames(t *testing.T) {
	root := t.TempDir()
	backupRoot := t.TempDir()
	content := "name: fallback-project\nservices:\n  one:\n    container_name: first\n    image: example/one\n  two:\n    container_name: second\n    image: example/two\n"
	if err := os.WriteFile(filepath.Join(root, "compose.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := BackupProject(&svc.ServiceContext{Config: config.Config{Compose: config.ComposeConfig{BackupDir: backupRoot}}}, root)
	if err != nil {
		t.Fatal(err)
	}
	if result["filename"] != "fallback-project_compose.yaml" {
		t.Fatalf("expected project name fallback, got %#v", result["filename"])
	}
}
func TestBackupProjectSupportsComposeFilenameVariants(t *testing.T) {
	for _, filename := range []string{"compose.yml", "docker-compose.yaml", "docker-compose.yml"} {
		t.Run(filename, func(t *testing.T) {
			root := t.TempDir()
			backupRoot := t.TempDir()
			content := "name: variant-project\nservices:\n  app:\n    image: example/app\n"
			if err := os.WriteFile(filepath.Join(root, filename), []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			result, err := BackupProject(&svc.ServiceContext{Config: config.Config{Compose: config.ComposeConfig{BackupDir: backupRoot}}}, root)
			if err != nil {
				t.Fatal(err)
			}
			if result["filename"] != "variant-project_compose.yaml" {
				t.Fatalf("unexpected backup filename: %#v", result["filename"])
			}
			if got, err := os.ReadFile(filepath.Join(backupRoot, "variant-project_compose.yaml")); err != nil || string(got) != content {
				t.Fatalf("unexpected backup content: %q, %v", got, err)
			}
		})
	}
}
func TestSaveProjectFileDoesNotCreateAutomaticVersionBackup(t *testing.T) {
	root := t.TempDir()
	backupRoot := t.TempDir()
	old := "services:\n  app:\n    image: old\n"
	if err := os.WriteFile(filepath.Join(root, "compose.yaml"), []byte(old), 0600); err != nil {
		t.Fatal(err)
	}
	version := ContentVersion([]byte(old))
	newContent := "services:\n  app:\n    image: new\n"
	ctx := &svc.ServiceContext{Config: config.Config{Compose: config.ComposeConfig{BackupDir: backupRoot}}}
	if _, err := SaveProjectFile(ctx, root, "compose.yaml", newContent, version); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(backupRoot)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("automatic backup was created: %s", strings.Join(names, ", "))
	}
}
