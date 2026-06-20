package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"actual-helper/internal/models"
	"actual-helper/internal/pdfutil"
	"actual-helper/internal/providers"
)

type ConvertService struct {
	registry *providers.Registry
}

func NewConvertService(registry *providers.Registry) *ConvertService {
	return &ConvertService{registry: registry}
}

func (service *ConvertService) ConvertFile(ctx context.Context, providerName string, file io.Reader, filename, contentType, password string) ([]byte, error) {
	logger := slog.With("provider", providerName, "filename", filename)

	provider, ok := service.registry.Get(providerName)
	if !ok {
		return nil, fmt.Errorf("provider %q not found", providerName)
	}

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
