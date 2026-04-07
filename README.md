# GoProxy

[![CI](https://github.com/benjaminserrano23/goproxy/actions/workflows/ci.yml/badge.svg)](https://github.com/benjaminserrano23/goproxy/actions/workflows/ci.yml)

Lightweight reverse proxy in Go with a configurable YAML-based middleware pipeline. Integrates with [ratelimiter-go](https://github.com/benjaminserrano23/ratelimiter-go) for distributed rate limiting via Redis.

## Features

- **YAML config** — define routes, upstreams, and middleware per path
- **Middleware pipeline** — chain middlewares in order per route
- **Built-in middlewares:**
  - **Logging** — structured JSON request/response logs
  - **Rate Limiting** — delegates to external ratelimiter-go service (fail-open)
  - **Security Headers** — HSTS, X-Frame-Options, X-Content-Type-Options, etc.
  - **Cache** — bounded in-memory GET response cache with TTL and eviction
  - **CORS** — configurable cross-origin resource sharing
- **Docker Compose stack** — goproxy + ratelimiter + Redis in one command
- **Graceful shutdown** — handles SIGINT/SIGTERM with in-flight request draining
- **Config validation** — clear error messages for invalid routes

## Quick Start with Docker

```bash
# Requires both repos side by side:
# parent/
#   goproxy/
#   ratelimiter-go/

cd goproxy
docker compose up --build
```

This starts 3 services:
- **Redis** (7-alpine) — persistent rate limit state
- **ratelimiter-go** — rate limiting API with Redis backend
- **goproxy** — reverse proxy on port 9090

```bash
# Test
curl http://localhost:9090/health
```

## Configuration

```yaml
server:
  port: "9090"

ratelimiter:
  url: http://localhost:8080
  limit: 60
  window: "1m"

routes:
  - path: /api
    upstream: http://localhost:8080
    middlewares: [logging, ratelimit, security]

  - path: /static
    upstream: http://localhost:8081
    middlewares: [logging, cache, security]
```

Environment variable overrides (for Docker):

| Env var | Description |
|---------|-------------|
| `PORT` | Server port |
| `RATELIMITER_URL` | URL of the ratelimiter service |

## Usage (standalone)

```bash
go build -o goproxy .
./goproxy

curl http://localhost:9090/health
```

## How it works

1. Reads `config.yaml` to discover routes and their middleware chains
2. For each route, creates an `httputil.ReverseProxy` pointing to the upstream
3. Wraps the proxy handler with the configured middlewares in order
4. Rate limiting calls the external ratelimiter-go service via HTTP
5. If the ratelimiter is unreachable, requests are **allowed** (fail-open pattern)

## Architecture

```
Client → goproxy(:9090) → [logging → ratelimit → security] → upstream
                                ↓
                        ratelimiter-go(:8080)
                                ↓
                           Redis(:6379)
```

## Development

```bash
go test ./...        # Run tests
go test ./... -v     # Verbose
go test ./... -race  # Race detector
```

## Tech stack

- Go (standard library `net/http`, `net/http/httputil`)
- `gopkg.in/yaml.v3` (config parsing)
- Docker + Docker Compose
