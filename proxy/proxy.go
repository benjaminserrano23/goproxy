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

// Registry maps middleware names to their constructors.
var registry = map[string]middleware.Middleware{}

func init() {
	registry["logging"] = middleware.Logging()
	registry["security"] = middleware.SecurityHeaders()
	registry["ratelimit"] = middleware.RateLimit(60, time.Minute)
	registry["cache"] = middleware.Cache(5*time.Minute, 1000)
	registry["cors"] = middleware.CORS("*")
}

// BuildMux creates an http.ServeMux from the config routes.
// Returns an error if any route has an invalid upstream URL.
func BuildMux(cfg *config.Config) (*http.ServeMux, error) {
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
