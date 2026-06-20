package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"

	"actual-helper/internal/models"
)

type ProviderConfig struct {
	ExcludeKeywords []string              `json:"exclude_keywords"`
	IncludeKeywords []string              `json:"include_keywords"`
	Categories      []models.CategoryRule `json:"categories"`
}

type config struct {
	Global    ProviderConfig            `json:"global"`
	Providers map[string]ProviderConfig `json:"providers"`
}

type Loader struct {
	path       string
	lastLoaded time.Time
	config     config
	mu         sync.Mutex
}

func NewLoader(path string) *Loader {
	return &Loader{path: path}
}

func (l *Loader) ProviderConfig(name string) ProviderConfig {
	l.ensureLoaded()
	return l.config.providerConfig(name)
}

func (l *Loader) ensureLoaded() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.path == "" {
		return
	}

	info, err := os.Stat(l.path)
	if err != nil {
		l.config = config{}
		slog.Warn("config file not accessible", "path", l.path, "error", err)
		return
	}

	if !info.ModTime().After(l.lastLoaded) && l.lastLoaded != (time.Time{}) {
		return
	}

	data, err := os.ReadFile(l.path)
	if err != nil {
		slog.Warn("config file unreadable", "path", l.path, "error", err)
		return
	}

	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		slog.Warn("config file invalid JSON", "path", l.path, "error", err)
		return
	}

	l.config = cfg
	l.lastLoaded = info.ModTime()
	slog.Info("config loaded", "path", l.path)
}

func (cfg config) providerConfig(name string) ProviderConfig {
	var pc ProviderConfig
	pc.ExcludeKeywords = append(pc.ExcludeKeywords, cfg.Global.ExcludeKeywords...)
	pc.IncludeKeywords = append(pc.IncludeKeywords, cfg.Global.IncludeKeywords...)
	pc.Categories = append(pc.Categories, cfg.Global.Categories...)

	if p, ok := cfg.Providers[name]; ok {
		pc.ExcludeKeywords = append(pc.ExcludeKeywords, p.ExcludeKeywords...)
		pc.IncludeKeywords = append(pc.IncludeKeywords, p.IncludeKeywords...)
		pc.Categories = append(p.Categories, pc.Categories...)
	}

	return pc
}
