package settingstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsUseOneFileAndPreserveFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "appSettings.json")
	t.Setenv("APP_SETTINGS_PATH", path)

	legacyPath := filepath.Join(dir, "backupSettings.json")
	if err := os.WriteFile(legacyPath, []byte(`{"retention":99}`), 0600); err != nil {
		t.Fatal(err)
	}
	if got := GetRetention(); got != 10 {
		t.Fatalf("legacy backup settings were read: retention = %d", got)
	}

	if _, err := SetUpdateCheckInterval("6h"); err != nil {
		t.Fatal(err)
	}
	if _, err := SetRetention(25); err != nil {
		t.Fatal(err)
	}
	if _, err := SetLogLevel("debug"); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stored Settings
	if err := json.Unmarshal(content, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.UpdateCheckInterval != "6h" || stored.Retention != 25 || stored.LogLevel != "debug" {
		t.Fatalf("settings were not preserved: %+v", stored)
	}
	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatalf("legacy backup settings fixture disappeared: %v", err)
	}
}

func TestRetentionValidation(t *testing.T) {
	if ValidRetention(0) || ValidRetention(101) {
		t.Fatal("out-of-range retention accepted")
	}
	if !ValidRetention(1) || !ValidRetention(100) {
		t.Fatal("valid retention rejected")
	}
}
