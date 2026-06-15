package middleware

import (
	"database/sql"
	"fmt"
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
	mu         sync.Mutex
	counters   map[string]*counter
	rate       int
	window     time.Duration
	db         *sql.DB
	trustProxy bool
}

// NewRateLimiter creates a rate limiter
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		counters: make(map[string]*counter),
		rate:     rate,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

// NewSessionRateLimiter creates a limiter keyed by valid session user ID, then client IP.
func NewSessionRateLimiter(db *sql.DB, trustProxy bool, rate int, window time.Duration) *RateLimiter {
	rl := NewRateLimiter(rate, window)
	rl.db = db
	rl.trustProxy = trustProxy
	return rl
}

// Allow checks whether a key remains below the rate limit
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

// Wrap applies rate limiting to an http handler
func (rl *RateLimiter) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.Allow(rl.requestKey(r)) {
			w.Header().Set("Retry-After", retryAfterSeconds(rl.window))
			http.Error(w, "Trop de requêtes. Réessayez plus tard.", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) requestKey(r *http.Request) string {
	if rl.db != nil {
		if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
			var userID string
			err := rl.db.QueryRow(
				`SELECT user_id FROM sessions WHERE session_token = ? AND expires_at > datetime('now')`,
				cookie.Value,
			).Scan(&userID)
			if err == nil && userID != "" {
				return "user:" + userID
			}
		}
	}
	return "ip:" + realIP(r, rl.trustProxy)
}

func retryAfterSeconds(window time.Duration) string {
	seconds := int64(window / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	return fmt.Sprintf("%d", seconds)
}

// cleanup removes expired rate limit counters
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

// realIP gets the client ip address
func realIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			return strings.TrimSpace(ip)
		}
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			if i := strings.Index(forwarded, ","); i != -1 {
				return strings.TrimSpace(forwarded[:i])
			}
			return strings.TrimSpace(forwarded)
		}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
