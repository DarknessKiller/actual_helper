package config_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"actual_helper/internal/config"
	"actual_helper/internal/models"
)

// benchFileConfig mirrors the unexported config.config shape so the
// from-file sub-benchmark pays the same json.Unmarshal cost the v0.3.1
// per-request path paid on every ConvertFile call.
type benchFileConfig struct {
	Global    benchProviderConfig            `json:"global"`
	Providers map[string]benchProviderConfig `json:"providers"`
}

type benchProviderConfig struct {
	ExcludeKeywords []string              `json:"exclude_keywords"`
	IncludeKeywords []string              `json:"include_keywords"`
	Categories      []models.CategoryRule `json:"categories"`
	AccountMappings map[string]string     `json:"account_mappings"`
}

// representativeConfig is a realistic tuning payload: global + per-provider
// keywords, categories and account mappings. All values are fake.
const representativeConfig = `{
	"global": {
		"exclude_keywords": ["Global Noise", "Junk Row"],
		"include_keywords": [],
		"categories": [
			{"keyword": "shopee", "group": "Shopping", "category": "Online"}
		]
	},
	"providers": {
		"tng": {
			"exclude_keywords": ["Quick Reload Payment", "Via eWallet to GO+"],
			"include_keywords": ["Daily Interest"],
			"categories": [
				{"keyword": "grab", "group": "Food & Dining", "category": "Delivery"}
			],
			"account_mappings": {"": "TNG Wallet"}
		}
	}
}`

// BenchmarkLoaderProviderConfig measures the per-request hot path
// (services.reloadProvider -> loader.ProviderConfig). v0.3.1 had no
// benchmarks; these establish the baseline and prove the in-memory cache
// (loaded via POST /config) removes the per-request stat/read/unmarshal.
func BenchmarkLoaderProviderConfig(b *testing.B) {
	// ApplyConfig logs at INFO; silence the default logger so the captured
	// bench output stays clean. Restored at the end.
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer slog.SetDefault(prev)

	b.Run("empty", func(b *testing.B) {
		loader := config.NewLoader()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = loader.ProviderConfig("tng")
		}
	})

	b.Run("loaded", func(b *testing.B) {
		loader := config.NewLoader()
		if err := loader.ApplyConfig([]byte(representativeConfig)); err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = loader.ProviderConfig("tng")
		}
	})

	// from_file replicates the pre-cache v0.3.1 behaviour: every request did
	// os.Stat + os.ReadFile + json.Unmarshal. It is the synthetic
	// "always-reload-from-file" path used to quantify the win the in-memory
	// cache delivers.
	b.Run("from_file", func(b *testing.B) {
		dir := b.TempDir()
		path := filepath.Join(dir, "provider_config.json")
		if err := os.WriteFile(path, []byte(representativeConfig), 0644); err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := os.Stat(path); err != nil {
				b.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				b.Fatal(err)
			}
			var cfg benchFileConfig
			if err := json.Unmarshal(data, &cfg); err != nil {
				b.Fatal(err)
			}
		}
	})
}
