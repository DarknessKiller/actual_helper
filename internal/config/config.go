package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"

	"actual_helper/internal/models"
)

type ProviderConfig struct {
	ExcludeKeywords []string              `json:"exclude_keywords"`
	IncludeKeywords []string              `json:"include_keywords"`
	Categories      []models.CategoryRule `json:"categories"`
	AccountMappings map[string]string     `json:"account_mappings"`
}

type config struct {
	Global    ProviderConfig            `json:"global"`
	Providers map[string]ProviderConfig `json:"providers"`
}

// Loader holds the active provider config in memory. By default nothing is
// loaded: ProviderConfig returns empty tuning until ApplyConfig is called.
// The file at ProviderConfigPath is read only by the GET /config download
// endpoint (see SampleConfig); it is never auto-applied to providers.
type Loader struct {
	mu     sync.Mutex
	loaded bool
	config config
}

func NewLoader() *Loader {
	return &Loader{}
}

// ApplyConfig parses a user-supplied config JSON document and stores it in
// memory. Subsequent ProviderConfig calls return the merged values. Invalid
// JSON leaves the existing config untouched and returns the parse error.
func (l *Loader) ApplyConfig(data []byte) error {
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	l.mu.Lock()
	l.config = cfg
	l.loaded = true
	l.mu.Unlock()

	slog.Info("provider config applied")
	return nil
}

// ClearConfig drops the in-memory config so providers run with empty tuning.
func (l *Loader) ClearConfig() {
	l.mu.Lock()
	l.config = config{}
	l.loaded = false
	l.mu.Unlock()

	slog.Info("provider config cleared")
}

func (l *Loader) ProviderConfig(name string) ProviderConfig {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.loaded {
		return ProviderConfig{}
	}
	return l.config.providerConfig(name)
}

// SampleConfig reads the sample config file at path. It is used by the
// GET /config download endpoint and never mutates Loader state. Returns an
// error when path is empty or the file cannot be read.
func SampleConfig(path string) ([]byte, error) {
	if path == "" {
		return nil, os.ErrNotExist
	}
	return os.ReadFile(path)
}

func (cfg config) providerConfig(name string) ProviderConfig {
	var pc ProviderConfig
	pc.ExcludeKeywords = append(pc.ExcludeKeywords, cfg.Global.ExcludeKeywords...)
	pc.IncludeKeywords = append(pc.IncludeKeywords, cfg.Global.IncludeKeywords...)
	pc.Categories = append(pc.Categories, cfg.Global.Categories...)

	if p, ok := cfg.Providers[name]; ok {
		pc.ExcludeKeywords = append(pc.ExcludeKeywords, p.ExcludeKeywords...)
		pc.IncludeKeywords = append(pc.IncludeKeywords, p.IncludeKeywords...)
		pc.Categories = append(pc.Categories, p.Categories...)
		pc.AccountMappings = p.AccountMappings
	}

	return pc
}
