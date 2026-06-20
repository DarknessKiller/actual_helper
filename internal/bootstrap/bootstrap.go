package bootstrap

import (
	"log/slog"
	"os"

	"actual-helper/internal/config"
	"actual-helper/internal/models"
	"actual-helper/internal/providers"
)

type ProviderFactory func(excludeKeywords, includeKeywords []string, categories []models.CategoryRule) providers.Provider

func Init(factories map[string]ProviderFactory) (*providers.Registry, *config.Loader) {
	configPath := os.Getenv("PROVIDER_CONFIG_PATH")
	loader := config.NewLoader(configPath)
	registry := providers.NewRegistry()

	if configPath == "" {
		slog.Warn("PROVIDER_CONFIG_PATH not set, running without filters or categories")
	}

	for name, factory := range factories {
		pc := loader.ProviderConfig(name)
		provider := factory(pc.ExcludeKeywords, pc.IncludeKeywords, pc.Categories)
		registry.Register(provider)
	}

	return registry, loader
}
