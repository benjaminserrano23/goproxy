package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// RateLimit returns a middleware that delegates rate limiting to an external
// ratelimiter-go service via HTTP. Fails open: if the service is unreachable,
// the request is allowed (with a warning log).
func RateLimit(ratelimiterURL string, limit int, window time.Duration) Middleware {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	windowStr := window.String()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr

			body := fmt.Sprintf(
				`{"key":%q,"limit":%d,"window":%q,"algorithm":"token_bucket"}`,
				key, limit, windowStr,
			)

			resp, err := client.Post(
				ratelimiterURL+"/check",
				"application/json",
				strings.NewReader(body),
			)
			if err != nil {
				// Fail open: allow request if ratelimiter is down
				log.Printf("warning: ratelimiter unreachable: %v", err)
				next.ServeHTTP(w, r)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusTooManyRequests {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
