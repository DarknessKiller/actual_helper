package providers

import (
	"context"
	"io"

	"actual_helper/internal/models"
	"actual_helper/internal/pdfutil"
)

type Provider interface {
	Name() string
	ParseCSV(ctx context.Context, r io.Reader) ([]models.ActualBudgetReport, error)
	ParsePDFText(ctx context.Context, text string) ([]models.ActualBudgetReport, error)

	ExtractionMethod() pdfutil.ExtractionMethod
}

// ConfigurableProvider is an optional interface providers can implement
// to receive runtime config updates (exclude/include keywords, category rules, account mappings).
type ConfigurableProvider interface {
	Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string)
}

type Registry struct {
	providers map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

func (registry *Registry) Register(provider Provider) {
	registry.providers[provider.Name()] = provider
}

func (registry *Registry) Get(name string) (Provider, bool) {
	provider, ok := registry.providers[name]
	return provider, ok
}
