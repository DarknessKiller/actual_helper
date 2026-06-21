package bootstrap

import (
	"actual_helper/internal/config"
	"actual_helper/internal/models"
	"actual_helper/internal/providers"
)

type ProviderFactory func(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider

func Init(factories map[string]ProviderFactory) (*providers.Registry, *config.Loader, config.Env) {

	env := config.LoadEnv()
	loader := config.NewLoader(env.ProviderConfigPath)
	registry := providers.NewRegistry()

	for name, factory := range factories {
		pc := loader.ProviderConfig(name)
		provider := factory(pc.ExcludeKeywords, pc.IncludeKeywords, pc.Categories, pc.AccountMappings)
		registry.Register(provider)
	}

	return registry, loader, env
}
