package middleware

import (
	"net/http"
	"sync"
	"time"
)

type rateLimitEntry struct {
	tokens     float64
	lastRefill time.Time
}

// RateLimit returns a token bucket rate limiting middleware.
// limit: max requests, window: time period.
func RateLimit(limit int, window time.Duration) Middleware {
	var mu sync.Mutex
	buckets := make(map[string]*rateLimitEntry)
	refillRate := float64(limit) / window.Seconds()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr

			mu.Lock()
			now := time.Now()
			entry, exists := buckets[key]
			if !exists {
				buckets[key] = &rateLimitEntry{tokens: float64(limit - 1), lastRefill: now}
				mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			elapsed := now.Sub(entry.lastRefill)
			entry.tokens += elapsed.Seconds() * refillRate
			if entry.tokens > float64(limit) {
				entry.tokens = float64(limit)
			}
			entry.lastRefill = now

			if entry.tokens < 1 {
				mu.Unlock()
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			entry.tokens--
			mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}
