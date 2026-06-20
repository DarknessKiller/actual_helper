package bootstrap

import (
	"log/slog"
	"os"

	"actual-helper/internal/config"
	"actual-helper/internal/providers"
	tngprov "actual-helper/internal/providers/tng"
)

func Init() (*providers.Registry, *config.Loader) {
	configPath := os.Getenv("PROVIDER_CONFIG_PATH")

	registry := providers.NewRegistry()
	loader := config.NewLoader(configPath)

	if configPath == "" {
		slog.Warn("PROVIDER_CONFIG_PATH not set, running without filters or categories")
		registry.Register(tngprov.New(nil, nil, nil))
		return registry, loader
	}

	tngCfg := loader.ProviderConfig("tng")
	tngProvider := tngprov.New(tngCfg.ExcludeKeywords, tngCfg.IncludeKeywords, tngCfg.Categories)
	registry.Register(tngProvider)

	return registry, loader
}
