package uobcredit

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"actual_helper/internal/models"
	"actual_helper/internal/pdfutil"
	"actual_helper/internal/providerbase"
	"actual_helper/internal/providers"
	"actual_helper/internal/rule"
)

type UOBProvider struct {
	engine *rule.Engine
}

func New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider {
	return &UOBProvider{engine: rule.NewEngine(excludeKeywords, includeKeywords, categories, accountMappings)}
}

func (p *UOBProvider) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) {
	p.engine.Reload(excludeKeywords, includeKeywords, categories, accountMappings)
}

func (p *UOBProvider) Name() string { return "uobcredit" }

func (p *UOBProvider) ParseCSV(_ context.Context, _ io.Reader) ([]models.ActualBudgetReport, error) {
	return nil, errors.New("not supported for uobcredit provider")
}

func (p *UOBProvider) ParsePDFText(ctx context.Context, text string) ([]models.ActualBudgetReport, error) {
	logger := slog.With("provider", "uobcredit", "format", "pdf")

	accountName := extractAccountName(text)
	reports, err := parseTransactions(text)
	if err != nil {
		return nil, err
	}

	logger.InfoContext(ctx, "pdf parsing started", "transactions", len(reports), "account", accountName)

	mapped := toBaseReports(reports)
	result := providerbase.MapReports(ctx, logger, p.engine, mapped, accountName)
	logger.InfoContext(ctx, "pdf parsing complete", "parsed_count", len(result))
	if len(result) == 0 {
		return nil, providerbase.ErrNoTransactions
	}
	return result, nil
}

func toBaseReports(in []UOBReport) []providerbase.PDFReport {
	out := make([]providerbase.PDFReport, len(in))
	for i, r := range in {
		out[i] = providerbase.PDFReport{
			TransDate:   r.TransDate,
			Description: r.Description,
			Amount:      r.Amount,
			IsCredit:    r.IsCredit,
		}
	}
	return out
}

func (p *UOBProvider) ExtractionMethod() pdfutil.ExtractionMethod {
	return pdfutil.ExtractionMethodPdftotext
}
