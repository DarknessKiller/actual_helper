package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"actual_helper/internal/config"
	"actual_helper/internal/models"
	"actual_helper/internal/pdfutil"
	"actual_helper/internal/providers"
)

type ConvertService struct {
	registry *providers.Registry
	loader   *config.Loader
}

func NewConvertService(registry *providers.Registry, loader *config.Loader) *ConvertService {
	return &ConvertService{registry: registry, loader: loader}
}

func (service *ConvertService) ConvertFile(ctx context.Context, providerName string, file io.Reader, filename, contentType, password string) ([]byte, error) {
	logger := slog.With("provider", providerName, "filename", filename)

	provider, ok := service.registry.Get(providerName)
	if !ok {
		return nil, fmt.Errorf("provider %q not found", providerName)
	}

	service.reloadProvider(providerName, provider)

	var reports []models.ActualBudgetReport
	var err error

	if strings.Contains(contentType, "pdf") {
		var text string
		text, err = pdfutil.ExtractText(file, password)
		if err == nil {
			reports, err = provider.ParsePDFText(ctx, text)
		}
	} else {
		var data []byte
		data, err = io.ReadAll(file)
		if err == nil {
			logger.InfoContext(ctx, "file parsing started", "size_bytes", len(data))
			reports, err = provider.ParseCSV(ctx, bytes.NewReader(data))
		}
	}

	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	logger.InfoContext(ctx, "parsing complete", "records", len(reports))

	csvData, err := ToActualCSV(reports)
	if err != nil {
		return nil, fmt.Errorf("csv conversion: %w", err)
	}

	logger.InfoContext(ctx, "csv conversion complete", "bytes", len(csvData))
	return csvData, nil
}

func (service *ConvertService) reloadProvider(name string, provider providers.Provider) {
	if service.loader == nil {
		return
	}
	pc := service.loader.ProviderConfig(name)
	if cp, ok := provider.(providers.ConfigurableProvider); ok {
		cp.Reload(pc.ExcludeKeywords, pc.IncludeKeywords, pc.Categories, pc.AccountMappings)
	}
}
