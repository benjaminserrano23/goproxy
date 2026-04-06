package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type logEntry struct {
	Timestamp  string `json:"timestamp"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Status     int    `json:"status"`
	DurationMs int64  `json:"duration_ms"`
	RemoteAddr string `json:"remote_addr"`
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// Logging returns structured JSON request/response logging middleware.
func Logging() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(sw, r)

			entry := logEntry{
				Timestamp:  start.UTC().Format(time.RFC3339),
				Method:     r.Method,
				Path:       r.URL.Path,
				Status:     sw.status,
				DurationMs: time.Since(start).Milliseconds(),
				RemoteAddr: r.RemoteAddr,
			}

			data, _ := json.Marshal(entry)
			log.Println(string(data))
		})
	}
}
