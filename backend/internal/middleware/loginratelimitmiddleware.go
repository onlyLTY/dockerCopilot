package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// LoginRateLimit 对登录接口按客户端 IP 做失败次数限制，防止暴力猜测 secretKey。
// 成功登录会清零该 IP 的失败计数。
type LoginRateLimit struct {
	mu       sync.Mutex
	failures map[string]*loginAttempt
	maxFail  int
	window   time.Duration
	banFor   time.Duration
}

type loginAttempt struct {
	count       int
	windowStart time.Time
	bannedUntil time.Time
}

func NewLoginRateLimit() *LoginRateLimit {
	return &LoginRateLimit{
		failures: make(map[string]*loginAttempt),
		maxFail:  5,
		window:   time.Minute,
		banFor:   2 * time.Minute,
	}
}

// Handle 作为 rest.Middleware 使用：仅应挂在 POST /api/auth。
func (m *LoginRateLimit) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if m.isBanned(ip) {
			httpx.WriteJson(w, http.StatusTooManyRequests, types.Resp{
				Code: http.StatusTooManyRequests,
				Msg:  "登录尝试过于频繁，请稍后再试",
				Data: map[string]interface{}{},
			})
			return
		}

		rw := &statusCapture{ResponseWriter: w, status: http.StatusOK}
		next(rw, r)

		// LoginHandler：失败写 HTTP 401/400，成功 200
		switch {
		case rw.status == http.StatusOK:
			m.Clear(ip)
		case rw.status == http.StatusUnauthorized || rw.status == http.StatusBadRequest:
			m.RecordFailure(ip)
		case rw.bizCode == 401:
			m.RecordFailure(ip)
		}
	}
}

func (m *LoginRateLimit) isBanned(ip string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(time.Now())
	a, ok := m.failures[ip]
	if !ok {
		return false
	}
	return time.Now().Before(a.bannedUntil)
}

func (m *LoginRateLimit) RecordFailure(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	m.cleanupLocked(now)
	a, ok := m.failures[ip]
	if !ok {
		m.failures[ip] = &loginAttempt{count: 1, windowStart: now}
		return
	}
	if now.Sub(a.windowStart) > m.window {
		a.count = 1
		a.windowStart = now
		a.bannedUntil = time.Time{}
		return
	}
	a.count++
	if a.count >= m.maxFail {
		a.bannedUntil = now.Add(m.banFor)
	}
}

func (m *LoginRateLimit) Clear(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.failures, ip)
}

func (m *LoginRateLimit) cleanupLocked(now time.Time) {
	for ip, a := range m.failures {
		expiredBan := a.bannedUntil.IsZero() || now.After(a.bannedUntil)
		expiredWindow := now.Sub(a.windowStart) > m.window
		if expiredBan && expiredWindow {
			delete(m.failures, ip)
		}
	}
}

func clientIP(r *http.Request) string {
	// 仅信任直连 RemoteAddr，避免 X-Forwarded-For 被伪造绕过限流。
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}

type statusCapture struct {
	http.ResponseWriter
	status  int
	bizCode int
	wrote   bool
}

func (s *statusCapture) WriteHeader(code int) {
	if !s.wrote {
		s.status = code
		s.wrote = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusCapture) Write(b []byte) (int, error) {
	if !s.wrote {
		s.status = http.StatusOK
		s.wrote = true
	}
	if s.bizCode == 0 {
		s.bizCode = peekJSONCode(b)
	}
	return s.ResponseWriter.Write(b)
}

func peekJSONCode(b []byte) int {
	const key = `"code"`
	idx := strings.Index(string(b), key)
	if idx < 0 {
		return 0
	}
	rest := strings.TrimLeft(string(b[idx+len(key):]), " \t\n\r:")
	n := 0
	for i := 0; i < len(rest); i++ {
		c := rest[i]
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}
