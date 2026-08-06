package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/onlyLTY/dockerCopilot/internal/datadir"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// LoginRateLimit 对登录接口按客户端 IP 做失败次数限制，防止暴力猜测 secretKey。
// 成功登录会清零该 IP 的失败计数。
// 状态持久化到 DATA_DIR/config/loginAttempts.json（可用 LOGIN_ATTEMPTS_PATH 覆盖），重启后仍生效。
type LoginRateLimit struct {
	mu       sync.Mutex
	failures map[string]*loginAttempt
	maxFail  int
	window   time.Duration
	banFor   time.Duration
	path     string
}

type loginAttempt struct {
	count       int
	windowStart time.Time
	bannedUntil time.Time
}

// 落盘 DTO（Unix 秒，便于跨重启与可读）。
type loginAttemptsFile struct {
	Entries map[string]loginAttemptDTO `json:"entries"`
}

type loginAttemptDTO struct {
	Count       int   `json:"count"`
	WindowStart int64 `json:"windowStart"`
	BannedUntil int64 `json:"bannedUntil,omitempty"`
}

func NewLoginRateLimit() *LoginRateLimit {
	m := &LoginRateLimit{
		failures: make(map[string]*loginAttempt),
		maxFail:  5,
		window:   time.Minute,
		banFor:   2 * time.Minute,
		path:     loginAttemptsStorePath(),
	}
	m.load()
	return m
}

func loginAttemptsStorePath() string {
	if p := strings.TrimSpace(os.Getenv("LOGIN_ATTEMPTS_PATH")); p != "" {
		return p
	}
	return datadir.LoginAttemptsPath()
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
	now := time.Now()
	m.cleanupLocked(now)
	a, ok := m.failures[ip]
	if !ok {
		m.failures[ip] = &loginAttempt{count: 1, windowStart: now}
	} else if now.Sub(a.windowStart) > m.window {
		a.count = 1
		a.windowStart = now
		a.bannedUntil = time.Time{}
	} else {
		a.count++
		if a.count >= m.maxFail {
			a.bannedUntil = now.Add(m.banFor)
		}
	}
	m.persistLocked()
	m.mu.Unlock()
}

func (m *LoginRateLimit) Clear(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.failures[ip]; !ok {
		return
	}
	delete(m.failures, ip)
	m.persistLocked()
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

func (m *LoginRateLimit) load() {
	if m.path == "" {
		return
	}
	content, err := os.ReadFile(m.path)
	if err != nil {
		if !os.IsNotExist(err) {
			logx.Errorf("读取登录限流文件失败 %s: %v", m.path, err)
		}
		return
	}
	var stored loginAttemptsFile
	if err := json.Unmarshal(content, &stored); err != nil {
		logx.Errorf("解析登录限流文件失败 %s: %v", m.path, err)
		return
	}
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	for ip, dto := range stored.Entries {
		ip = strings.TrimSpace(ip)
		if ip == "" || dto.Count <= 0 {
			continue
		}
		a := &loginAttempt{
			count:       dto.Count,
			windowStart: time.Unix(dto.WindowStart, 0),
		}
		if dto.BannedUntil > 0 {
			a.bannedUntil = time.Unix(dto.BannedUntil, 0)
		}
		m.failures[ip] = a
	}
	m.cleanupLocked(now)
	// 若清理掉过期项，回写缩小文件（失败不影响启动）
	if len(m.failures) != len(stored.Entries) {
		m.persistLocked()
	}
}

// persistLocked 原子写入当前 failures。调用方必须已持有 m.mu。
func (m *LoginRateLimit) persistLocked() {
	if m.path == "" {
		return
	}
	now := time.Now()
	m.cleanupLocked(now)
	payload := loginAttemptsFile{Entries: make(map[string]loginAttemptDTO, len(m.failures))}
	for ip, a := range m.failures {
		dto := loginAttemptDTO{
			Count:       a.count,
			WindowStart: a.windowStart.Unix(),
		}
		if !a.bannedUntil.IsZero() {
			dto.BannedUntil = a.bannedUntil.Unix()
		}
		payload.Entries[ip] = dto
	}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		logx.Errorf("序列化登录限流失败: %v", err)
		return
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0750); err != nil {
		logx.Errorf("创建登录限流目录失败: %v", err)
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(m.path), ".loginAttempts-*.tmp")
	if err != nil {
		logx.Errorf("创建登录限流临时文件失败: %v", err)
		return
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		logx.Errorf("设置登录限流文件权限失败: %v", err)
		return
	}
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		logx.Errorf("写入登录限流失败: %v", err)
		return
	}
	if err := tmp.Close(); err != nil {
		logx.Errorf("关闭登录限流临时文件失败: %v", err)
		return
	}
	if err := os.Rename(tmpName, m.path); err != nil {
		logx.Errorf("保存登录限流失败: %v", err)
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
