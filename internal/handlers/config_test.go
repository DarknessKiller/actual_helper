package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"actual_helper/internal/config"
	"actual_helper/internal/handlers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func newConfigServer(env config.Env, loader *config.Loader) http.Handler {
	mux := http.NewServeMux()
	handler := handlers.NewConfigHandler(loader, env)
	handlers.RegisterConfigRoutes(mux, handler)
	return mux
}

// sessionConfigBody is fake tuning; names never reference real data.
var sessionConfigBody = `{
	"global": {"exclude_keywords": ["Session Noise"]},
	"providers": {"tng": {"exclude_keywords": ["Session TNG Noise"]}}
}`

var _ = Describe("ConfigHandler", func() {
	var loader *config.Loader
	var mux http.Handler

	BeforeEach(func() {
		loader = config.NewLoader()
		mux = newConfigServer(config.Env{}, loader)
	})

	Describe("GET /config", func() {
		It("serves the sample config file with application/json", func() {
			path := filepath.Join(GinkgoT().TempDir(), "provider_config.example.json")
			Expect(os.WriteFile(path, []byte(`{"global":{"exclude_keywords":["Sample"]}}`), 0644)).To(Succeed())

			env := config.Env{ProviderConfigPath: path}
			mux = newConfigServer(env, loader)

			req := httptest.NewRequest(http.MethodGet, "/config", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/json"))
			Expect(w.Body.Bytes()).To(MatchJSON(`{"global":{"exclude_keywords":["Sample"]}}`))
		})

		It("returns 404 when the sample path is empty", func() {
			mux = newConfigServer(config.Env{}, loader)

			req := httptest.NewRequest(http.MethodGet, "/config", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("returns 404 when the sample file is unreadable", func() {
			env := config.Env{ProviderConfigPath: filepath.Join(GinkgoT().TempDir(), "missing.json")}
			mux = newConfigServer(env, loader)

			req := httptest.NewRequest(http.MethodGet, "/config", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("POST /config", func() {
		It("stores the rules under a fresh session cookie", func() {
			req := httptest.NewRequest(http.MethodPost, "/config", stringReader(sessionConfigBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			var resp map[string]bool
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp["stored"]).To(BeTrue())

			cookies := w.Result().Cookies()
			Expect(cookies).NotTo(BeEmpty())
			sid := cookies[0].Value
			Expect(sid).To(HaveLen(64))

			pc := loader.SessionProviderConfig(sid, "tng")
			Expect(pc.ExcludeKeywords).To(ConsistOf("Session Noise", "Session TNG Noise"))
		})

		It("returns 400 on invalid JSON without storing the rules", func() {
			req := httptest.NewRequest(http.MethodPost, "/config", stringReader("{invalid"))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(loader.SessionProviderConfig(w.Result().Cookies()[0].Value, "tng").ExcludeKeywords).To(BeEmpty())
		})

		It("requires a valid session id: garbage cookies are replaced", func() {
			req := httptest.NewRequest(http.MethodPost, "/config", stringReader(sessionConfigBody))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(&http.Cookie{Name: "ah_sid", Value: "not-a-real-sid-wisely-rejected"})
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Result().Cookies()[0].Value).To(HaveLen(64))
		})

		It("rejects new sessions with 429 when the store is full", func() {
			loader.SessionLimit(0)
			loader.SetSessionTTL(-1) // ensure even existing entries expire

			req := httptest.NewRequest(http.MethodPost, "/config", stringReader(sessionConfigBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusTooManyRequests))
		})
	})

	Describe("DELETE /config", func() {
		It("clears the caller's own session rules", func() {
			req := httptest.NewRequest(http.MethodPost, "/config", stringReader(sessionConfigBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			sid := w.Result().Cookies()[0].Value

			del := httptest.NewRequest(http.MethodDelete, "/config", nil)
			del.AddCookie(&http.Cookie{Name: "ah_sid", Value: sid})
			w = httptest.NewRecorder()
			mux.ServeHTTP(w, del)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(loader.SessionProviderConfig(sid, "tng").ExcludeKeywords).To(BeEmpty())
		})

		It("is a no-op without a session cookie", func() {
			req := httptest.NewRequest(http.MethodDelete, "/config", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("method handling", func() {
		It("returns 405 for unsupported methods", func() {
			req := httptest.NewRequest(http.MethodPut, "/config", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
		})
	})
})
