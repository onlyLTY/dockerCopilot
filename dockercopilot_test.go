package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/onlyLTY/dockerCopilot/internal/config"
)

func TestRuntimeSecurityWarningsAreAdvisory(t *testing.T) {
	t.Setenv("BACKUP_ENCRYPTION_KEY", "")
	var cfg config.Config
	cfg.Auth.AccessSecret = "123456"
	warnings := runtimeSecurityWarnings(cfg)
	if len(warnings) != 2 {
		t.Fatalf("expected short and numeric warnings, got %v", warnings)
	}
	cfg.Auth.AccessSecret = "0123456789abcdef-strong-random-key"
	if warnings := runtimeSecurityWarnings(cfg); len(warnings) != 0 {
		t.Fatalf("strong secret produced warnings: %v", warnings)
	}
}

func TestRuntimeSecurityWarningsAllowUserChosenBackupKey(t *testing.T) {
	t.Setenv("BACKUP_ENCRYPTION_KEY", "too-short")
	var cfg config.Config
	cfg.Auth.AccessSecret = "0123456789abcdef-strong-random-key"
	warnings := runtimeSecurityWarnings(cfg)
	if len(warnings) != 1 {
		t.Fatalf("expected one advisory warning, got %v", warnings)
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := securityHeaders(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodGet, "https://example.test/manager", nil)
	recorder := httptest.NewRecorder()
	handler(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("unexpected response code %d", recorder.Code)
	}
	for _, name := range []string{"Content-Security-Policy", "Strict-Transport-Security", "X-Content-Type-Options", "Cross-Origin-Opener-Policy"} {
		if recorder.Header().Get(name) == "" {
			t.Fatalf("security header %s is missing", name)
		}
	}
}

func TestSecurityHeadersDisableAPICaching(t *testing.T) {
	handler := securityHeaders(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodGet, "http://example.test/api/containers", nil)
	recorder := httptest.NewRecorder()
	handler(recorder, request)
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("API response cache policy is %q", recorder.Header().Get("Cache-Control"))
	}
}
