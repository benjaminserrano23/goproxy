# GoProxy

Lightweight reverse proxy in Go with a configurable YAML-based middleware pipeline.

## Features

- **YAML config** — define routes, upstreams, and middleware per path
- **Middleware pipeline** — chain middlewares in order per route
- **Built-in middlewares:**
  - **Logging** — structured JSON request/response logs
  - **Rate Limiting** — token bucket per remote address
  - **Security Headers** — HSTS, X-Frame-Options, X-Content-Type-Options, etc.
  - **Cache** — in-memory GET response cache with configurable TTL
- **Standard library** — built on `net/http` and `httputil.ReverseProxy`

## Configuration

```yaml
server:
  port: "9090"

routes:
  - path: /api
    upstream: http://localhost:8080
    middlewares: [logging, ratelimit, security]

  - path: /static
    upstream: http://localhost:8081
    middlewares: [logging, cache, security]
```

## Usage

```bash
# Build and run
go build -o goproxy .
./goproxy

# Or run directly
go run .

# Health check
curl http://localhost:9090/health
```

## How it works

1. Reads `config.yaml` to discover routes and their middleware chains
2. For each route, creates an `httputil.ReverseProxy` pointing to the upstream
3. Wraps the proxy handler with the configured middlewares in order
4. Incoming requests are matched by path prefix and routed through the pipeline

## Development

```bash
# Run tests
go test ./...

# Verbose
go test ./... -v
```

## Tech stack

- Go (standard library `net/http`, `net/http/httputil`)
- `gopkg.in/yaml.v3` (config parsing)
