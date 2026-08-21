package handlers_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"actual_helper/internal/config"
	"actual_helper/internal/handlers"
	"actual_helper/internal/models"
	"actual_helper/internal/pdfutil"
	"actual_helper/internal/providers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// configurableMock is a fake Provider that records ConfigurableProvider.Reload
// calls so config lifecycle tests can assert applied tuning. All data is fake.
type configurableMock struct {
	name        string
	lastReload  reloadCall
	reloadCount int
}

type reloadCall struct {
	excludeKeywords []string
	includeKeywords []string
	categories      []models.CategoryRule
	accountMappings map[string]string
}

func (m *configurableMock) Name() string { return m.name }
func (m *configurableMock) ParseCSV(context.Context, io.Reader) ([]models.ActualBudgetReport, error) {
	return nil, nil
}
func (m *configurableMock) ParsePDFText(context.Context, string) ([]models.ActualBudgetReport, error) {
	return nil, nil
}
func (m *configurableMock) ExtractionMethod() pdfutil.ExtractionMethod {
	return pdfutil.ExtractionMethodDigital
}
func (m *configurableMock) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) {
	m.lastReload = reloadCall{
		excludeKeywords: excludeKeywords,
		includeKeywords: includeKeywords,
		categories:      categories,
		accountMappings: accountMappings,
	}
	m.reloadCount++
}

func newConfigServer(env config.Env, registry *providers.Registry, loader *config.Loader) (http.Handler, *handlers.ConfigHandler) {
	mux := http.NewServeMux()
	handler := handlers.NewConfigHandler(loader, registry, env)
	handlers.RegisterConfigRoutes(mux, handler)
	return mux, handler
}

var _ = Describe("ConfigHandler", func() {
	Describe("GET /config", func() {
		It("serves the sample config file with application/json", func() {
			path := filepath.Join(GinkgoT().TempDir(), "provider_config.example.json")
			Expect(os.WriteFile(path, []byte(`{"global":{"exclude_keywords":["Sample"]}}`), 0644)).To(Succeed())

			env := config.Env{ProviderConfigPath: path}
			loader := config.NewLoader()
			mux, _ := newConfigServer(env, providers.NewRegistry(), loader)

			req := httptest.NewRequest(http.MethodGet, "/config", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/json"))
			Expect(w.Body.Bytes()).To(MatchJSON(`{"global":{"exclude_keywords":["Sample"]}}`))
		})

		It("returns 404 when the sample path is empty", func() {
			env := config.Env{}
			mux, _ := newConfigServer(env, providers.NewRegistry(), config.NewLoader())

			req := httptest.NewRequest(http.MethodGet, "/config", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("returns 404 when the sample file is unreadable", func() {
			env := config.Env{ProviderConfigPath: filepath.Join(GinkgoT().TempDir(), "missing.json")}
			mux, _ := newConfigServer(env, providers.NewRegistry(), config.NewLoader())

			req := httptest.NewRequest(http.MethodGet, "/config", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})

	Describe("POST /config", func() {
		It("applies merged tuning to every configurable provider", func() {
			tng := &configurableMock{name: "tng"}
			registry := providers.NewRegistry()
			registry.Register(tng)

			loader := config.NewLoader()
			mux, _ := newConfigServer(config.Env{}, registry, loader)

			body := `{
				"global": {"exclude_keywords": ["Global Noise"]},
				"providers": {"tng": {"exclude_keywords": ["TNG Fee"], "account_mappings": {"": "TNG"}}}
			}`
			req := httptest.NewRequest(http.MethodPost, "/config", stringReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			var resp map[string][]string
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp["applied"]).To(ConsistOf("tng"))

			Expect(tng.reloadCount).To(Equal(1))
			Expect(tng.lastReload.excludeKeywords).To(ConsistOf("Global Noise", "TNG Fee"))
			Expect(tng.lastReload.accountMappings).To(Equal(map[string]string{"": "TNG"}))

			pc := loader.ProviderConfig("tng")
			Expect(pc.ExcludeKeywords).To(ConsistOf("Global Noise", "TNG Fee"))
		})

		It("returns 400 on invalid JSON without mutating providers", func() {
			tng := &configurableMock{name: "tng"}
			registry := providers.NewRegistry()
			registry.Register(tng)

			mux, _ := newConfigServer(config.Env{}, registry, config.NewLoader())

			req := httptest.NewRequest(http.MethodPost, "/config", stringReader("{invalid"))
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(tng.reloadCount).To(BeZero())
		})

		It("returns 400 when the body is unreadable", func() {
			mux, _ := newConfigServer(config.Env{}, providers.NewRegistry(), config.NewLoader())

			req := httptest.NewRequest(http.MethodPost, "/config", errReader{})
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("DELETE /config", func() {
		It("clears tuning on every configurable provider", func() {
			tng := &configurableMock{name: "tng"}
			registry := providers.NewRegistry()
			registry.Register(tng)

			loader := config.NewLoader()
			Expect(loader.ApplyConfig([]byte(`{"global":{"exclude_keywords":["Noise"]}}`))).To(Succeed())
			tng.lastReload.excludeKeywords = []string{"Noise"}

			mux, _ := newConfigServer(config.Env{}, registry, loader)

			req := httptest.NewRequest(http.MethodDelete, "/config", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			var resp map[string]bool
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp["cleared"]).To(BeTrue())

			Expect(tng.reloadCount).To(Equal(1))
			Expect(tng.lastReload.excludeKeywords).To(BeNil())
			Expect(tng.lastReload.includeKeywords).To(BeNil())
			Expect(tng.lastReload.categories).To(BeNil())
			Expect(tng.lastReload.accountMappings).To(BeNil())

			pc := loader.ProviderConfig("tng")
			Expect(pc.ExcludeKeywords).To(BeEmpty())
		})
	})

	Describe("method handling", func() {
		It("returns 405 for unsupported methods", func() {
			mux, _ := newConfigServer(config.Env{}, providers.NewRegistry(), config.NewLoader())

			req := httptest.NewRequest(http.MethodPut, "/config", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
		})
	})
})
