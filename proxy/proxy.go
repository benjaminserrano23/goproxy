package proxy

import (
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
	registry["cache"] = middleware.Cache(5 * time.Minute)
}

// BuildMux creates an http.ServeMux from the config routes.
func BuildMux(cfg *config.Config) *http.ServeMux {
	mux := http.NewServeMux()

	for _, route := range cfg.Routes {
		upstream, err := url.Parse(route.Upstream)
		if err != nil {
			panic("invalid upstream URL: " + route.Upstream)
		}

		reverseProxy := httputil.NewSingleHostReverseProxy(upstream)

		// Resolve middleware chain
		var mws []middleware.Middleware
		for _, name := range route.Middlewares {
			if mw, ok := registry[strings.TrimSpace(name)]; ok {
				mws = append(mws, mw)
			}
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

	return mux
}
