package auth

import (
	"net"
	"net/http"
	"os"
	"strings"
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
	directIP := remoteIP(r.RemoteAddr)
	if directIP == nil {
		if r.RemoteAddr == "" {
			return "unknown"
		}
		return r.RemoteAddr
	}
	trustedProxies := parseTrustedProxyCIDRs(os.Getenv("TRUSTED_PROXY_CIDRS"))
	if !ipInNetworks(directIP, trustedProxies) {
		return directIP.String()
	}

	forwarded := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for index := len(forwarded) - 1; index >= 0; index-- {
		candidate := net.ParseIP(strings.TrimSpace(forwarded[index]))
		if candidate == nil {
			return directIP.String()
		}
		if !ipInNetworks(candidate, trustedProxies) {
			return candidate.String()
		}
	}
	return directIP.String()
}

func remoteIP(remoteAddress string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		return net.ParseIP(host)
	}
	return net.ParseIP(strings.Trim(remoteAddress, "[]"))
}

func parseTrustedProxyCIDRs(raw string) []*net.IPNet {
	var networks []*net.IPNet
	for _, value := range strings.Split(raw, ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		_, network, err := net.ParseCIDR(value)
		if err == nil {
			networks = append(networks, network)
		}
	}
	return networks
}

func ipInNetworks(ip net.IP, networks []*net.IPNet) bool {
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
