package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type counter struct {
	count   int
	resetAt time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	counters map[string]*counter
	rate     int
	window   time.Duration
}

func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		counters: make(map[string]*counter),
		rate:     rate,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	c, exists := rl.counters[key]
	if !exists || now.After(c.resetAt) {
		rl.counters[key] = &counter{count: 1, resetAt: now.Add(rl.window)}
		return true
	}

	if c.count >= rl.rate {
		return false
	}

	c.count++
	return true
}

func (rl *RateLimiter) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.Allow(realIP(r)) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, c := range rl.counters {
			if now.After(c.resetAt) {
				delete(rl.counters, key)
			}
		}
		rl.mu.Unlock()
	}
}

func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if i := strings.Index(forwarded, ","); i != -1 {
			return strings.TrimSpace(forwarded[:i])
		}
		return strings.TrimSpace(forwarded)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
