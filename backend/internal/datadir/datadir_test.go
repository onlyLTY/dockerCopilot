package datadir

import (
	"path/filepath"
	"testing"
)

func TestRootDefault(t *testing.T) {
	t.Setenv("DATA_DIR", "")
	if Root() != "/data" {
		t.Fatalf("Root() = %q, want /data", Root())
	}
}

func TestRootOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)
	if Root() != dir {
		t.Fatalf("Root() = %q, want %q", Root(), dir)
	}
	wantIcons := filepath.Join(dir, "icon", "icons")
	if IconDir() != wantIcons {
		t.Fatalf("IconDir() = %q, want %q", IconDir(), wantIcons)
	}
	wantSettings := filepath.Join(dir, "config", "appSettings.json")
	if AppSettingsPath() != wantSettings {
		t.Fatalf("AppSettingsPath() = %q, want %q", AppSettingsPath(), wantSettings)
	}
}
