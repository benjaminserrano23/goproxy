package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/benjaminserrano23/goproxy/config"
	"github.com/benjaminserrano23/goproxy/middleware"
)

// BuildMux creates an http.ServeMux from the config routes.
// Returns an error if any route has an invalid upstream URL.
func BuildMux(cfg *config.Config) (*http.ServeMux, error) {
	// Parse ratelimiter window
	window, err := time.ParseDuration(cfg.RateLimiter.Window)
	if err != nil {
		window = time.Minute
	}

	// Build middleware registry with config values
	registry := map[string]middleware.Middleware{
		"logging":   middleware.Logging(),
		"security":  middleware.SecurityHeaders(),
		"ratelimit": middleware.RateLimit(cfg.RateLimiter.URL, cfg.RateLimiter.Limit, window),
		"cache":     middleware.Cache(5*time.Minute, 1000),
		"cors":      middleware.CORS("*"),
	}

	mux := http.NewServeMux()

	for _, route := range cfg.Routes {
		upstream, err := url.Parse(route.Upstream)
		if err != nil {
			return nil, fmt.Errorf("invalid upstream URL %q for path %q: %w", route.Upstream, route.Path, err)
		}

		reverseProxy := httputil.NewSingleHostReverseProxy(upstream)

		// Resolve middleware chain
		var mws []middleware.Middleware
		for _, name := range route.Middlewares {
			mw, ok := registry[strings.TrimSpace(name)]
			if !ok {
				log.Printf("warning: unknown middleware %q in route %q, skipping", name, route.Path)
				continue
			}
			mws = append(mws, mw)
		}

		handler := middleware.Chain(reverseProxy, mws...)
		mux.Handle(route.Path+"/", http.StripPrefix(route.Path, handler))
		mux.Handle(route.Path, handler)
	}

	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	return mux, nil
}
