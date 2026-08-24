package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/config"
	"github.com/zeromicro/go-zero/rest"
)

func TestHealthCheckAddressUsesConfiguredPort(t *testing.T) {
	cfg := config.Config{}
	cfg.Host = "0.0.0.0"
	cfg.Port = 18080
	address, err := healthCheckAddress(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if address != "127.0.0.1:18080" {
		t.Fatalf("unexpected health address %q", address)
	}
}

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

func TestFrontendRoutesStartWithoutDuplicates(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate test port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("failed to release test port: %v", err)
	}

	var cfg config.Config
	cfg.Host = "127.0.0.1"
	cfg.Port = port
	server := rest.MustNewServer(cfg.RestConf)
	RegisterHandlers(server)
	startResult := make(chan any, 1)
	go func() {
		defer func() { startResult <- recover() }()
		server.Start()
	}()

	client := &http.Client{Timeout: 250 * time.Millisecond}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case recovered := <-startResult:
			if recovered != nil {
				t.Fatalf("server panicked while registering frontend routes: %v", recovered)
			}
			t.Fatal("server stopped before becoming ready")
		default:
		}

		response, requestErr := client.Get(fmt.Sprintf("http://127.0.0.1:%d/manager/", port))
		if requestErr == nil {
			response.Body.Close()
			if response.StatusCode != http.StatusOK {
				server.Stop()
				t.Fatalf("GET /manager/ returned %d", response.StatusCode)
			}
			server.Stop()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	server.Stop()
	t.Fatal("frontend server did not become ready")
}

func TestTCPHealthCheck(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create test listener: %v", err)
	}
	address := listener.Addr().String()
	if err := checkTCPHealth(address, time.Second); err != nil {
		listener.Close()
		t.Fatalf("health check rejected a listening server: %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("failed to close test listener: %v", err)
	}
	if err := checkTCPHealth(address, 100*time.Millisecond); err == nil {
		t.Fatal("health check accepted a closed server")
	}
}
