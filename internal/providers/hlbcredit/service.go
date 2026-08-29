package hlbcredit

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

type HLBProvider struct {
	engine *rule.Engine
}

func New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider {
	return &HLBProvider{engine: rule.NewEngine(excludeKeywords, includeKeywords, categories, accountMappings)}
}

func (p *HLBProvider) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) {
	p.engine.Reload(excludeKeywords, includeKeywords, categories, accountMappings)
}

func (p *HLBProvider) Name() string { return "hlbcredit" }

func (p *HLBProvider) ParseCSV(_ context.Context, _ io.Reader) ([]models.ActualBudgetReport, error) {
	return nil, errors.New("not supported for hlbcredit provider")
}

func (p *HLBProvider) ParsePDFText(ctx context.Context, text string) ([]models.ActualBudgetReport, error) {
	logger := slog.With("provider", "hlbcredit", "format", "pdf")

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

func toBaseReports(in []HLBReport) []providerbase.PDFReport {
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

func (p *HLBProvider) ExtractionMethod() pdfutil.ExtractionMethod {
	return pdfutil.ExtractionMethodPdftotext
}
