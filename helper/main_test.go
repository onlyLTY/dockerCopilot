package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProxyDocumentPreservesOtherFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "daemon.json")
	oldPath, backup := daemonPath, backupPath
	defer func() { daemonPath, backupPath = oldPath, backup }()
	daemonPath = path
	backupPath = filepath.Join(dir, "backup", "daemon.json.bak")
	original := []byte(`{"debug":true,"hosts":["unix:///var/run/docker.sock"],"proxies":{"http-proxy":"http://old:1","custom":"keep"}}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeConfig(original, []byte(`{"debug":true,"hosts":["unix:///var/run/docker.sock"],"proxies":{"http-proxy":"http://new:2","custom":"keep"}}`)); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatal(err)
	}
	if got["debug"] != true || got["hosts"] == nil {
		t.Fatalf("non-proxy daemon fields changed: %+v", got)
	}
	proxies := got["proxies"].(map[string]any)
	if proxies["http-proxy"] != "http://new:2" || proxies["custom"] != "keep" {
		t.Fatalf("proxy fields changed unexpectedly: %+v", proxies)
	}
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("backup was not created: %v", err)
	}
}

func TestValidateProxyRejectsCredentials(t *testing.T) {
	if err := validateProxy(proxyValues{HTTPProxy: "http://user:pass@proxy:7890"}); err == nil {
		t.Fatal("expected proxy credentials to be rejected")
	}
}

func TestApplyRejectsStaleHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "daemon.json")
	oldPath, backup := daemonPath, backupPath
	defer func() { daemonPath, backupPath = oldPath, backup }()
	daemonPath = path
	backupPath = filepath.Join(dir, "backup", "daemon.json.bak")
	original := []byte(`{"debug":true,"proxies":{"http-proxy":"http://old:1"}}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}

	h := &helper{operations: make(map[string]*restartOperation)}
	req := httptest.NewRequest(http.MethodPost, "/proxy/apply", strings.NewReader(`{"httpProxy":"http://new:2","hash":"stale"}`))
	w := httptest.NewRecorder()
	h.apply(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("stale hash status = %d, want %d", w.Code, http.StatusConflict)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string(original) {
		t.Fatalf("stale hash changed daemon config: %s", content)
	}
}
