package bootstrap

import (
	"actual-helper/internal/config"
	"actual-helper/internal/models"
	"actual-helper/internal/providers"
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
