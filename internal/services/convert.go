package services

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"

	"actual_helper/internal/bootstrap"
	"actual_helper/internal/config"
	"actual_helper/internal/models"
	"actual_helper/internal/pdfutil"
	"actual_helper/internal/providers"
)

type ConvertService struct {
	registry  *providers.Registry
	loader    *config.Loader
	factories map[string]bootstrap.ProviderFactory
}

func NewConvertService(registry *providers.Registry, loader *config.Loader,
	factories map[string]bootstrap.ProviderFactory) *ConvertService {
	return &ConvertService{registry: registry, loader: loader, factories: factories}
}

// providerFor resolves the provider for one conversion. With session rules
// configured it builds a fresh provider instance carrying those rules, so the
// shared singleton stays pristine for everyone else.
func (service *ConvertService) providerFor(ctx context.Context, providerName string) (providers.Provider, error) {
	provider, ok := service.registry.Get(providerName)
	if !ok {
		return nil, fmt.Errorf("provider %q not found", providerName)
	}
	if provider == nil || provider.ExtractionMethod() == "" {
		return nil, fmt.Errorf("provider %q not configured", providerName)
	}
	if service.loader == nil {
		return provider, nil
	}

	pc := service.loader.SessionProviderConfig(sessionIDFromContext(ctx), providerName)
	if pc.ExcludeKeywords == nil && pc.IncludeKeywords == nil && pc.Categories == nil && pc.AccountMappings == nil {
		return provider, nil
	}
	factory, ok := service.factories[providerName]
	if !ok {
		return nil, fmt.Errorf("provider %q does not support session configuration", providerName)
	}
	return factory(pc.ExcludeKeywords, pc.IncludeKeywords, pc.Categories, pc.AccountMappings), nil
}

// ConvertFile converts file bytes, which stay in memory for the whole request.
func (service *ConvertService) ConvertFile(ctx context.Context, providerName string, file []byte, contentType, password string) ([]byte, error) {
	logger := slog.With("provider", providerName)

	provider, err := service.providerFor(ctx, providerName)
	if err != nil {
		return nil, err
	}

	var reports []models.ActualBudgetReport

	switch {
	case strings.Contains(contentType, "pdf"):
		var text string
		text, err = pdfutil.ExtractText(ctx, file, password, provider.ExtractionMethod())
		if err != nil {
			return nil, fmt.Errorf("pdf extraction: %w", err)
		}
		reports, err = provider.ParsePDFText(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("pdf parsing: %w", err)
		}
	case strings.Contains(contentType, "csv"):
		logger.InfoContext(ctx, "file parsing started", "size_bytes", len(file))
		reports, err = provider.ParseCSV(ctx, bytes.NewReader(file))
		if err != nil {
			return nil, fmt.Errorf("csv parsing: %w", err)
		}
	default:
		err := fmt.Errorf("unsupported content type")
		return nil, err
	}

	logger.InfoContext(ctx, "parsing complete", "records", len(reports))

	csvData, err := ToActualCSV(reports)
	if err != nil {
		return nil, fmt.Errorf("csv conversion: %w", err)
	}

	logger.InfoContext(ctx, "csv conversion complete", "bytes", len(csvData))
	return csvData, nil
}
