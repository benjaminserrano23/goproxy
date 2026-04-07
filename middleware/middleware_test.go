package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/benjaminserrano23/goproxy/middleware"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}

func TestChain(t *testing.T) {
	var order []string

	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m1-before")
			next.ServeHTTP(w, r)
			order = append(order, "m1-after")
		})
	}

	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m2-before")
			next.ServeHTTP(w, r)
			order = append(order, "m2-after")
		})
	}

	handler := middleware.Chain(okHandler(), m1, m2)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	expected := []string{"m1-before", "m2-before", "m2-after", "m1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d", len(expected), len(order))
	}
	for i, v := range expected {
		if order[i] != v {
			t.Fatalf("position %d: expected %s, got %s", i, v, order[i])
		}
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := middleware.Chain(okHandler(), middleware.SecurityHeaders())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	checks := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":       "DENY",
		"X-XSS-Protection":      "1; mode=block",
	}
	for header, expected := range checks {
		if got := w.Header().Get(header); got != expected {
			t.Errorf("%s: expected %q, got %q", header, expected, got)
		}
	}
}

func TestRateLimit_AllowsThenDenies(t *testing.T) {
	handler := middleware.Chain(okHandler(), middleware.RateLimit(3, time.Minute))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d should be allowed, got %d", i+1, w.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

func TestCache_HitAndMiss(t *testing.T) {
	callCount := 0
	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write([]byte("response"))
	})

	handler := middleware.Chain(upstream, middleware.Cache(time.Minute, 100))

	// First request — MISS
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Header().Get("X-Cache") != "MISS" {
		t.Fatal("first request should be a cache MISS")
	}

	// Second request — HIT
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Header().Get("X-Cache") != "HIT" {
		t.Fatal("second request should be a cache HIT")
	}

	if callCount != 1 {
		t.Fatalf("upstream should be called once, called %d times", callCount)
	}
}

func TestCache_Eviction(t *testing.T) {
	callCount := 0
	upstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write([]byte("response"))
	})

	// Max 2 entries
	handler := middleware.Chain(upstream, middleware.Cache(time.Minute, 2))

	// Fill cache with 2 entries
	for _, path := range []string{"/a", "/b"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// Add a 3rd — should evict the oldest
	req := httptest.NewRequest(http.MethodGet, "/c", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// /c should now be cached
	req = httptest.NewRequest(http.MethodGet, "/c", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Header().Get("X-Cache") != "HIT" {
		t.Fatal("/c should be a cache HIT")
	}
}

func TestCORS_Headers(t *testing.T) {
	handler := middleware.Chain(okHandler(), middleware.CORS("*"))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected *, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("missing Access-Control-Allow-Methods")
	}
}

func TestCORS_Preflight(t *testing.T) {
	handler := middleware.Chain(okHandler(), middleware.CORS("https://example.com"))
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("preflight should return 204, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("expected https://example.com, got %q", got)
	}
}

func TestLogging_DoesNotBreak(t *testing.T) {
	handler := middleware.Chain(okHandler(), middleware.Logging())
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
