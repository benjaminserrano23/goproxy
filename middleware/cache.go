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
func Cache(ttl time.Duration) Middleware {
	var mu sync.RWMutex
	entries := make(map[string]*cacheEntry)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only cache GET requests
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			key := r.URL.String()

			// Check cache
			mu.RLock()
			entry, exists := entries[key]
			mu.RUnlock()

			if exists && time.Now().Before(entry.expiration) {
				// Serve from cache
				for k, v := range entry.header {
					w.Header()[k] = v
				}
				w.Header().Set("X-Cache", "HIT")
				w.WriteHeader(entry.status)
				w.Write(entry.body)
				return
			}

			// Cache miss — proxy and capture
			buf := &bytes.Buffer{}
			cw := &cacheWriter{ResponseWriter: w, buf: buf, status: http.StatusOK}
			w.Header().Set("X-Cache", "MISS")

			next.ServeHTTP(cw, r)

			// Only cache successful responses
			if cw.status >= 200 && cw.status < 300 {
				mu.Lock()
				entries[key] = &cacheEntry{
					body:       buf.Bytes(),
					header:     cw.Header().Clone(),
					status:     cw.status,
					expiration: time.Now().Add(ttl),
				}
				mu.Unlock()
			}
		})
	}
}
