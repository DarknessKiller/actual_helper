# Per-IP Rate Limiting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-IP rate limiting middleware using `golang.org/x/time/rate` to prevent API spam on the `/convert/{provider}` endpoint.

**Architecture:** A new `internal/ratelimit` package with a `Middleware` function that returns `net/http` middleware. It maintains a map of client IP → token bucket, applies rate limiting based on env-configured limits, and returns 429 when exceeded. Registered as a Fuego global middleware via `fuego.Use`.

**Tech Stack:** Go, `golang.org/x/time/rate`, `net/http`, Fuego middleware

---

### Task 1: Add golang.org/x/time dependency

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Install the dependency**

Run: `cd /Users/itsgoyoung/Projects/actual_helper && go get golang.org/x/time@latest`

Expected output: `go.mod` and `go.sum` are updated with the new dependency.

- [ ] **Step 2: Verify the dependency is available**

Run: `cd /Users/itsgoyoung/Projects/actual_helper && go list -m golang.org/x/time`
Expected: `golang.org/x/time v0.x.x`

---

### Task 2: Create the rate limiter middleware

**Files:**
- Create: `internal/ratelimit/middleware.go`

- [ ] **Step 1: Write the middleware**

Create `internal/ratelimit/middleware.go`:

```go
package ratelimit

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     rate.Limit
	burst    int
}

func New() *RateLimiter {
	r := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate.Limit(getEnvFloat("RATE_LIMIT_RATE", 10.0/60.0)),
		burst:    getEnvInt("RATE_LIMIT_BURST", 10),
	}

	go r.cleanup()
	return r
}

func (rl *RateLimiter) getVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 5*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func Middleware(next http.Handler) http.Handler {
	rl := New()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}

		limiter := rl.getVisitor(host)
		if !limiter.Allow() {
			slog.Warn("rate limit exceeded", "ip", host, "path", r.URL.Path)
			w.Header().Set("Retry-After", "60")
			http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/itsgoyoung/Projects/actual_helper && go build ./internal/ratelimit/...`
Expected: No errors.

---

### Task 3: Write tests for the rate limiter

**Files:**
- Create: `internal/ratelimit/middleware_test.go`
- Create: `internal/ratelimit/ratelimit_suite_test.go`

- [ ] **Step 1: Create ginkgo test suite file**

Create `internal/ratelimit/ratelimit_suite_test.go`:

```go
package ratelimit_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRateLimit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RateLimit Suite")
}
```

- [ ] **Step 2: Write test file**

Create `internal/ratelimit/middleware_test.go`:

```go
package ratelimit_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"actual-helper/internal/ratelimit"
)

var _ = Describe("RateLimiter", func() {
	var handler http.Handler

	BeforeEach(func() {
		handler = ratelimit.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	})

	It("allows requests under the limit", func() {
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))
		}
	})

	It("blocks requests over the burst limit", func() {
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "192.168.1.2:12345"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.2:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusTooManyRequests))
	})

	It("different IPs have independent limits", func() {
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "192.168.1.3:12345"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.4:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("returns Retry-After header on rate limit", func() {
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "192.168.1.5:12345"
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.5:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		Expect(w.Header().Get("Retry-After")).To(Equal("60"))
	})
})
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `cd /Users/itsgoyoung/Projects/actual_helper && go test ./internal/ratelimit/... -v -count=1`

Expected: All 4 tests pass.

---

### Task 4: Register middleware with the server

**Files:**
- Modify: `cmd/app/main.go`

- [ ] **Step 1: Add import and register middleware**

Add `"actual-helper/internal/ratelimit"` to imports, and add `fuego.Use(server, ratelimit.Middleware)` before `handlers.RegisterConvertRoutes`:

```go
import (
	"log"

	"actual-helper/internal/bootstrap"
	"actual-helper/internal/handlers"
	rytprov "actual-helper/internal/providers/ryt"
	tngprov "actual-helper/internal/providers/tng"
	"actual-helper/internal/ratelimit"
	"actual-helper/internal/services"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
)

func main() {
	server := fuego.NewServer(
		fuego.WithEngineOptions(
			fuego.WithOpenAPIConfig(fuego.OpenAPIConfig{
				Info: &openapi3.Info{
					Title:       "Actual Helper",
					Description: "Converts bank/fintech transaction files (CSV or PDF) into Actual Budget-compatible CSV format.",
					Version:     "1.0.0",
				},
			}),
		),
	)

	registry, loader := bootstrap.Init(map[string]bootstrap.ProviderFactory{
		"tng": tngprov.New,
		"ryt": rytprov.New,
	})

	convertService := services.NewConvertService(registry, loader)
	handler := handlers.NewConvertHandler(convertService)
	fuego.Use(server, ratelimit.Middleware)
	handlers.RegisterConvertRoutes(server, handler)

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/itsgoyoung/Projects/actual_helper && go build ./...`
Expected: No errors.

---

### Task 5: Full test suite and commit

- [ ] **Step 1: Run full test suite**

Run: `cd /Users/itsgoyoung/Projects/actual_helper && go test ./... -count=1 2>&1 | tail -20`

Expected: All tests pass, including the new ratelimit tests.

- [ ] **Step 2: Commit**

```bash
git add go.mod go.sum internal/ratelimit/ cmd/app/main.go docs/superpowers/specs/2026-06-21-rate-limiting.md
git commit -m "feat: add per-IP rate limiting middleware

Uses golang.org/x/time/rate token bucket with per-IP tracking.
Configurable via RATE_LIMIT_RATE and RATE_LIMIT_BURST env vars.
Returns 429 Too Many Requests with Retry-After header when exceeded.
Stale visitor entries cleaned up every 5 minutes."
```
