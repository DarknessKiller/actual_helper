package providers

import (
	"context"
	"io"

	"actual-helper/internal/models"
)

type Provider interface {
	Name() string
	ParseCSV(ctx context.Context, r io.Reader) ([]models.ActualBudgetReport, error)
	ParsePDFText(ctx context.Context, text string) ([]models.ActualBudgetReport, error)
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
