package auth

import (
	"fmt"
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
