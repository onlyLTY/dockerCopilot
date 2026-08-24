package auth

import (
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	loginFailureWindow = 5 * time.Minute
	loginBlockDuration = 15 * time.Minute
	maxLoginFailures   = 5
	maxLoginClients    = 10_000
)

type loginAttempt struct {
	failures     int
	windowStart  time.Time
	blockedUntil time.Time
	lastSeen     time.Time
}

type loginAttemptLimiter struct {
	mu       sync.Mutex
	attempts map[string]loginAttempt
}

var authLimiter = &loginAttemptLimiter{attempts: make(map[string]loginAttempt)}

func (l *loginAttemptLimiter) allow(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cleanup(now)
	attempt, exists := l.attempts[key]
	if !exists {
		return true, 0
	}
	attempt.lastSeen = now
	l.attempts[key] = attempt
	if now.Before(attempt.blockedUntil) {
		return false, attempt.blockedUntil.Sub(now)
	}
	return true, 0
}

func (l *loginAttemptLimiter) record(key string, success bool, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cleanup(now)
	if success {
		delete(l.attempts, key)
		return
	}
	attempt, exists := l.attempts[key]
	if !exists && len(l.attempts) >= maxLoginClients {
		l.evictOldest()
	}
	if attempt.windowStart.IsZero() || now.Sub(attempt.windowStart) > loginFailureWindow {
		attempt.failures = 0
		attempt.windowStart = now
	}
	attempt.failures++
	attempt.lastSeen = now
	if attempt.failures >= maxLoginFailures {
		attempt.blockedUntil = now.Add(loginBlockDuration)
		attempt.failures = 0
		attempt.windowStart = time.Time{}
	}
	l.attempts[key] = attempt
}

func (l *loginAttemptLimiter) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	for key, attempt := range l.attempts {
		if oldestKey == "" || attempt.lastSeen.Before(oldestTime) {
			oldestKey = key
			oldestTime = attempt.lastSeen
		}
	}
	if oldestKey != "" {
		delete(l.attempts, oldestKey)
	}
}

func (l *loginAttemptLimiter) cleanup(now time.Time) {
	for key, attempt := range l.attempts {
		if now.Sub(attempt.lastSeen) > 30*time.Minute && now.After(attempt.blockedUntil) {
			delete(l.attempts, key)
		}
	}
}

func loginClientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if r.RemoteAddr == "" {
		return "unknown"
	}
	return r.RemoteAddr
}
