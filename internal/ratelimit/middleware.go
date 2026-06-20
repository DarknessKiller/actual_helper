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
		setRateLimitHeaders := func() {
			remaining := limiter.Tokens()
			if remaining < 0 {
				remaining = 0
			}
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.burst))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(int(remaining)))
		}

		if !limiter.Allow() {
			setRateLimitHeaders()

			reserve := limiter.Reserve()
			retryAfter := reserve.Delay()
			reserve.Cancel()

			slog.Warn("rate limit exceeded", "ip", host, "path", r.URL.Path)
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
			http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
			return
		}

		setRateLimitHeaders()
		next.ServeHTTP(w, r)
	})
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
		slog.Warn("invalid env var, using default", "key", key, "value", v, "default", defaultVal)
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
		slog.Warn("invalid env var, using default", "key", key, "value", v, "default", defaultVal)
	}
	return defaultVal
}
