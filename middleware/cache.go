package middleware

import (
	"bytes"
	"net/http"
	"sync"
	"time"
)

type cacheEntry struct {
	body       []byte
	header     http.Header
	status     int
	expiration time.Time
	insertedAt time.Time
}

type cacheWriter struct {
	http.ResponseWriter
	buf    *bytes.Buffer
	status int
}

func (cw *cacheWriter) WriteHeader(code int) {
	cw.status = code
	cw.ResponseWriter.WriteHeader(code)
}

func (cw *cacheWriter) Write(b []byte) (int, error) {
	cw.buf.Write(b)
	return cw.ResponseWriter.Write(b)
}

// Cache returns a middleware that caches GET responses for the given TTL.
// maxEntries limits the cache size; when full, the oldest entry is evicted.
func Cache(ttl time.Duration, maxEntries int) Middleware {
	var mu sync.RWMutex
	entries := make(map[string]*cacheEntry)

	evictOldest := func() {
		var oldestKey string
		var oldestTime time.Time
		for k, v := range entries {
			if oldestKey == "" || v.insertedAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.insertedAt
			}
		}
		if oldestKey != "" {
			delete(entries, oldestKey)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			key := r.URL.String()

			mu.RLock()
			entry, exists := entries[key]
			mu.RUnlock()

			if exists && time.Now().Before(entry.expiration) {
				for k, v := range entry.header {
					w.Header()[k] = v
				}
				w.Header().Set("X-Cache", "HIT")
				w.WriteHeader(entry.status)
				w.Write(entry.body)
				return
			}

			buf := &bytes.Buffer{}
			cw := &cacheWriter{ResponseWriter: w, buf: buf, status: http.StatusOK}
			w.Header().Set("X-Cache", "MISS")

			next.ServeHTTP(cw, r)

			if cw.status >= 200 && cw.status < 300 {
				now := time.Now()
				mu.Lock()
				// Evict expired entries first
				for k, v := range entries {
					if now.After(v.expiration) {
						delete(entries, k)
					}
				}
				// Evict oldest if still at capacity
				if len(entries) >= maxEntries {
					evictOldest()
				}
				entries[key] = &cacheEntry{
					body:       buf.Bytes(),
					header:     cw.Header().Clone(),
					status:     cw.status,
					expiration: now.Add(ttl),
					insertedAt: now,
				}
				mu.Unlock()
			}
		})
	}
}
