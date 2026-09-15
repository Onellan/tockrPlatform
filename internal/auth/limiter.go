package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"strings"
	"sync"
	"time"
)

type LoginLimiterConfig struct {
	FailureThreshold int
	BaseBackoff      time.Duration
	MaxBackoff       time.Duration
	EntryTTL         time.Duration
	MaxEntries       int
}

type loginEntry struct {
	failures     int
	blockedUntil time.Time
	lastSeen     time.Time
}

type LoginLimiter struct {
	mu      sync.Mutex
	entries map[string]loginEntry
	config  LoginLimiterConfig
}

func NewLoginLimiter(config LoginLimiterConfig) *LoginLimiter {
	if config.FailureThreshold < 1 {
		config.FailureThreshold = 5
	}
	if config.BaseBackoff <= 0 {
		config.BaseBackoff = 2 * time.Second
	}
	if config.MaxBackoff < config.BaseBackoff {
		config.MaxBackoff = time.Minute
	}
	if config.EntryTTL <= 0 {
		config.EntryTTL = 15 * time.Minute
	}
	if config.MaxEntries < 1 {
		config.MaxEntries = 4096
	}
	return &LoginLimiter{entries: make(map[string]loginEntry), config: config}
}

func LoginAbuseKey(identity, remoteAddr string) string {
	host := strings.TrimSpace(remoteAddr)
	if parsed, _, err := net.SplitHostPort(host); err == nil && parsed != "" {
		host = parsed
	}
	if host == "" {
		host = "unknown"
	}
	value := strings.ToLower(strings.TrimSpace(identity)) + "\x00" + host
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (l *LoginLimiter) RetryAfter(key string, now time.Time) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked(now)
	entry, ok := l.entries[key]
	if !ok || !now.Before(entry.blockedUntil) {
		return 0, false
	}
	entry.lastSeen = now
	l.entries[key] = entry
	return entry.blockedUntil.Sub(now), true
}

func (l *LoginLimiter) RecordFailure(key string, now time.Time) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked(now)
	entry, ok := l.entries[key]
	if !ok && len(l.entries) >= l.config.MaxEntries {
		l.evictOldestLocked()
	}
	entry.failures++
	entry.lastSeen = now
	if entry.failures >= l.config.FailureThreshold {
		delay := l.config.BaseBackoff
		for i := l.config.FailureThreshold; i < entry.failures && delay < l.config.MaxBackoff; i++ {
			delay *= 2
			if delay > l.config.MaxBackoff {
				delay = l.config.MaxBackoff
			}
		}
		entry.blockedUntil = now.Add(delay)
		l.entries[key] = entry
		return delay, true
	}
	l.entries[key] = entry
	return 0, false
}

func (l *LoginLimiter) RecordSuccess(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

func (l *LoginLimiter) pruneLocked(now time.Time) {
	for key, entry := range l.entries {
		if now.Sub(entry.lastSeen) >= l.config.EntryTTL {
			delete(l.entries, key)
		}
	}
}

func (l *LoginLimiter) evictOldestLocked() {
	var key string
	var oldest time.Time
	for candidate, entry := range l.entries {
		if key == "" || entry.lastSeen.Before(oldest) {
			key, oldest = candidate, entry.lastSeen
		}
	}
	if key != "" {
		delete(l.entries, key)
	}
}
