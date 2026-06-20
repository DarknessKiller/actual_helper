# Per-IP Rate Limiting

## Problem

The `/convert/{provider}` endpoint has no request limits, making it vulnerable to API spam and abuse.

## Design

### Approach

Per-IP token bucket rate limiter using `golang.org/x/time/rate`, applied as a Fuego global middleware.

### Configuration (Environment Variables)

| Variable | Default | Description |
|----------|---------|-------------|
| `RATE_LIMIT_RATE` | `10/60` | Token replenishment rate (tokens per second). Default 10 req/min |
| `RATE_LIMIT_BURST` | `10` | Maximum burst size (max accumulated tokens) |

### Behavior

- Each unique client IP gets its own token bucket
- On each request, the bucket is checked:
  - Tokens available → request passes through
  - No tokens → returns `429 Too Many Requests` with `Retry-After` header (time until next token)
- Stale IP entries are cleaned up periodically (every 5 minutes) to prevent memory leaks
- Rate limit headers added to responses: `X-RateLimit-Limit`, `X-RateLimit-Remaining`

### Files

| File | Action | Purpose |
|------|--------|---------|
| `internal/ratelimit/middleware.go` | Create | Rate limiter middleware |
| `internal/ratelimit/middleware_test.go` | Create | Tests for rate limiter |
| `cmd/app/main.go` | Modify | Register middleware with server |
| `go.mod` / `go.sum` | Modify | Add `golang.org/x/time` dependency |

### Tests

| Test | Description |
|------|-------------|
| Allows requests under the limit | 10 requests in quick succession → all pass (200) |
| Blocks requests over the limit | 11th request → returns 429 |
| Resets after window expires | Wait for window → request passes again |
| Different IPs have independent limits | IP A can exhaust its limit, IP B still passes |
| Missing env vars use defaults | Unset `RATE_LIMIT_REQUESTS` + `RATE_LIMIT_WINDOW` → default limits apply |

## No other changes

This is a self-contained rate limiter package added to the project, plus one line in `main.go` to register it. No handlers, services, or providers are modified.
