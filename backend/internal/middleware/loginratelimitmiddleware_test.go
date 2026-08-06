package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoginRateLimitPersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "loginAttempts.json")
	t.Setenv("LOGIN_ATTEMPTS_PATH", path)

	m1 := NewLoginRateLimit()
	ip := "203.0.113.10"
	for i := 0; i < 5; i++ {
		m1.RecordFailure(ip)
	}
	if !m1.isBanned(ip) {
		t.Fatal("expected ban after 5 failures")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected persist file: %v", err)
	}

	// 模拟重启：新实例从同一文件加载
	m2 := NewLoginRateLimit()
	if !m2.isBanned(ip) {
		t.Fatal("ban should survive restart")
	}

	// 成功登录清零并落盘
	m2.Clear(ip)
	if m2.isBanned(ip) {
		t.Fatal("clear should unban")
	}
	m3 := NewLoginRateLimit()
	if m3.isBanned(ip) {
		t.Fatal("clear should persist across restart")
	}
}

func TestLoginRateLimitExpiredEntriesNotRestoredAsBan(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "loginAttempts.json")
	t.Setenv("LOGIN_ATTEMPTS_PATH", path)

	// 写入已过期的 ban 记录
	old := time.Now().Add(-10 * time.Minute)
	payload := loginAttemptsFile{Entries: map[string]loginAttemptDTO{
		"198.51.100.1": {
			Count:       5,
			WindowStart: old.Unix(),
			BannedUntil: old.Add(2 * time.Minute).Unix(),
		},
	}}
	data, _ := json.Marshal(payload)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	m := NewLoginRateLimit()
	if m.isBanned("198.51.100.1") {
		t.Fatal("expired ban should not apply after load")
	}
}

func TestLoginRateLimitMiddlewareRecordsUnauthorized(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LOGIN_ATTEMPTS_PATH", filepath.Join(dir, "loginAttempts.json"))

	m := NewLoginRateLimit()
	handler := m.Handle(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":401,"msg":"无效的secretKey"}`))
	})

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
		req.RemoteAddr = "192.0.2.1:12345"
		rr := httptest.NewRecorder()
		handler(rr, req)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	rr := httptest.NewRecorder()
	handler(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rr.Code)
	}
}
