package ratelimit_test

import (
	"net/http"
	"net/http/httptest"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"actual-helper/internal/ratelimit"
)

var _ = Describe("RateLimiter", func() {
	var handler http.Handler

	BeforeEach(func() {
		os.Unsetenv("RATE_LIMIT_RATE")
		os.Unsetenv("RATE_LIMIT_BURST")
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

	It("returns rate limit headers on rate limit", func() {
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
		Expect(w.Code).To(Equal(http.StatusTooManyRequests))
		Expect(w.Header().Get("Retry-After")).ToNot(BeEmpty())
		Expect(w.Header().Get("X-RateLimit-Limit")).To(Equal("10"))
		Expect(w.Header().Get("X-RateLimit-Remaining")).To(Equal("0"))
	})

	It("returns X-RateLimit-Limit and X-RateLimit-Remaining on success", func() {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.6:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		Expect(w.Header().Get("X-RateLimit-Limit")).To(Equal("10"))
		Expect(w.Header().Get("X-RateLimit-Remaining")).To(Equal("9"))
	})
})
