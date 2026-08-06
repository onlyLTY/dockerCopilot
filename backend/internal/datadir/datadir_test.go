package datadir

import (
	"path/filepath"
	"testing"
)

func TestRootDefault(t *testing.T) {
	t.Setenv("DATA_DIR", "")
	SetFromConfig("")
	t.Cleanup(func() { SetFromConfig("") })
	if Root() != "/data" {
		t.Fatalf("Root() = %q, want /data", Root())
	}
}

func TestRootEnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DATA_DIR", dir)
	SetFromConfig(filepath.Join(t.TempDir(), "from-yaml")) // yaml 应被 env 盖住
	t.Cleanup(func() { SetFromConfig("") })

	got := Root()
	want, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Root() = %q, want %q", got, want)
	}
	wantIcons := filepath.Join(want, "icon", "icons")
	if IconDir() != wantIcons {
		t.Fatalf("IconDir() = %q, want %q", IconDir(), wantIcons)
	}
	wantSettings := filepath.Join(want, "config", "appSettings.json")
	if AppSettingsPath() != wantSettings {
		t.Fatalf("AppSettingsPath() = %q, want %q", AppSettingsPath(), wantSettings)
	}
	wantLogin := filepath.Join(want, "config", "loginAttempts.json")
	if LoginAttemptsPath() != wantLogin {
		t.Fatalf("LoginAttemptsPath() = %q, want %q", LoginAttemptsPath(), wantLogin)
	}
}

func TestRootFromConfig(t *testing.T) {
	t.Setenv("DATA_DIR", "")
	dir := t.TempDir()
	SetFromConfig(dir)
	t.Cleanup(func() { SetFromConfig("") })

	want, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	if Root() != want {
		t.Fatalf("Root() = %q, want %q", Root(), want)
	}
	if BackupsDir() != filepath.Join(want, "backups") {
		t.Fatalf("BackupsDir() = %q, want under %q", BackupsDir(), want)
	}
}
