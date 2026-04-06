package middleware

import "net/http"

// Middleware wraps an http.Handler with additional behavior.
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares in order: first middleware is outermost.
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	// Apply in reverse so the first middleware wraps outermost
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
