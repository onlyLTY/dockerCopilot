package auth

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoginAttemptLimiterBlocksAndResets(t *testing.T) {
	limiter := &loginAttemptLimiter{attempts: make(map[string]loginAttempt)}
	now := time.Unix(1000, 0)
	for index := 0; index < maxLoginFailures; index++ {
		limiter.record("client", false, now.Add(time.Duration(index)*time.Second))
	}
	if allowed, retry := limiter.allow("client", now.Add(10*time.Second)); allowed || retry <= 0 {
		t.Fatalf("blocked client was allowed: allowed=%v retry=%s", allowed, retry)
	}
	limiter.record("client", true, now.Add(11*time.Second))
	if allowed, _ := limiter.allow("client", now.Add(12*time.Second)); !allowed {
		t.Fatal("successful login did not reset limiter")
	}
}

func TestLoginAttemptLimiterDoesNotTrackUncheckedClients(t *testing.T) {
	limiter := &loginAttemptLimiter{attempts: make(map[string]loginAttempt)}
	if allowed, _ := limiter.allow("new-client", time.Now()); !allowed {
		t.Fatal("new client was unexpectedly blocked")
	}
	if len(limiter.attempts) != 0 {
		t.Fatalf("allow check allocated an entry: %d", len(limiter.attempts))
	}
}

func TestLoginAttemptLimiterCapsTrackedClients(t *testing.T) {
	limiter := &loginAttemptLimiter{attempts: make(map[string]loginAttempt)}
	now := time.Unix(1000, 0)
	for index := 0; index < maxLoginClients+1; index++ {
		limiter.record(fmt.Sprintf("client-%d", index), false, now.Add(time.Duration(index)*time.Nanosecond))
	}
	if len(limiter.attempts) != maxLoginClients {
		t.Fatalf("tracked %d clients, want %d", len(limiter.attempts), maxLoginClients)
	}
	if _, exists := limiter.attempts["client-0"]; exists {
		t.Fatal("oldest client was not evicted")
	}
}

func TestLoginClientKeyIgnoresForwardedHeaderFromUntrustedPeer(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
	request := httptest.NewRequest("POST", "/api/auth", nil)
	request.RemoteAddr = "192.0.2.10:12345"
	request.Header.Set("X-Forwarded-For", "198.51.100.25")
	if key := loginClientKey(request); key != "192.0.2.10" {
		t.Fatalf("client key = %q, want direct peer", key)
	}
}

func TestLoginClientKeyUsesRightmostUntrustedForwardedAddress(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
	request := httptest.NewRequest("POST", "/api/auth", nil)
	request.RemoteAddr = "10.0.0.8:12345"
	request.Header.Set("X-Forwarded-For", "203.0.113.99, 198.51.100.25")
	if key := loginClientKey(request); key != "198.51.100.25" {
		t.Fatalf("client key = %q, want rightmost untrusted hop", key)
	}
}

func TestLoginClientKeyWalksTrustedProxyChain(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8, 172.16.0.0/12")
	request := httptest.NewRequest("POST", "/api/auth", nil)
	request.RemoteAddr = "10.0.0.8:12345"
	request.Header.Set("X-Forwarded-For", "198.51.100.25, 172.16.0.4")
	if key := loginClientKey(request); key != "198.51.100.25" {
		t.Fatalf("client key = %q, want original untrusted client", key)
	}
}

func TestLoginClientKeyRejectsMalformedForwardedChain(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
	request := httptest.NewRequest("POST", "/api/auth", nil)
	request.RemoteAddr = "10.0.0.8:12345"
	request.Header.Set("X-Forwarded-For", "198.51.100.25, invalid")
	if key := loginClientKey(request); key != "10.0.0.8" {
		t.Fatalf("client key = %q, want safe direct-peer fallback", key)
	}
}
