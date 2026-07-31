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

func TestContainerUpdateIgnorePersistence(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APP_SETTINGS_PATH", filepath.Join(dir, "appSettings.json"))

	if err := SetContainerUpdateIgnored("db", true); err != nil {
		t.Fatal(err)
	}
	if err := SetContainerUpdateIgnored("db", true); err != nil {
		t.Fatal(err)
	}
	if !IsContainerUpdateIgnored("db") {
		t.Fatal("container update ignore was not persisted")
	}
	if err := RenameContainerUpdateIgnore("db", "database"); err != nil {
		t.Fatal(err)
	}
	if IsContainerUpdateIgnored("db") || !IsContainerUpdateIgnored("database") {
		t.Fatal("container update ignore was not migrated")
	}
	if err := SetContainerUpdateIgnored("database", false); err != nil {
		t.Fatal(err)
	}
	if IsContainerUpdateIgnored("database") {
		t.Fatal("container update ignore was not removed")
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

func TestProxySettingsPersistenceAndValidation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "appSettings.json")
	t.Setenv("APP_SETTINGS_PATH", path)

	if got := GetProxySettings(); got.GithubProxy != "" || got.HTTPProxy != "" || got.HTTPSProxy != "" || got.NoProxy != "" {
		t.Fatalf("unexpected proxy defaults: %+v", got)
	}

	want := ProxySettings{
		GithubProxy: "https://github-proxy.example/",
		HTTPProxy:   "http://127.0.0.1:7890",
		HTTPSProxy:  "http://127.0.0.1:7890",
		NoProxy:     "localhost,127.0.0.1,::1",
	}
	got, err := SetProxySettings(want)
	if err != nil {
		t.Fatal(err)
	}
	if got != want || GetProxySettings() != want {
		t.Fatalf("proxy settings were not persisted: got=%+v want=%+v", got, want)
	}
	var stored Settings
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, &stored); err != nil || !stored.ProxySettingsConfigured {
		t.Fatalf("proxy configuration marker was not persisted: %+v", stored)
	}

	for _, value := range []ProxySettings{
		{HTTPProxy: "proxy.example:7890"},
		{HTTPProxy: "http://user:password@proxy.example:7890"},
		{GithubProxy: "ftp://github-proxy.example/"},
		{NoProxy: "localhost\n127.0.0.1"},
	} {
		if _, err := SetProxySettings(value); err == nil {
			t.Fatalf("invalid proxy settings accepted: %+v", value)
		}
	}
}
