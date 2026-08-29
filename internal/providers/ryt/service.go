package ryt

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"actual_helper/internal/models"
	"actual_helper/internal/pdfutil"
	"actual_helper/internal/providers"
	"actual_helper/internal/rule"
	"actual_helper/internal/textutil"
)

type RytProvider struct {
	engine *rule.Engine
}

func New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider {
	return &RytProvider{engine: rule.NewEngine(excludeKeywords, includeKeywords, categories, accountMappings)}
}

func (p *RytProvider) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) {
	p.engine.Reload(excludeKeywords, includeKeywords, categories, accountMappings)
}

func (p *RytProvider) Name() string { return "ryt" }

func (p *RytProvider) ParseCSV(_ context.Context, _ io.Reader) ([]models.ActualBudgetReport, error) {
	return nil, errors.New("not supported for ryt provider")
}

func (p *RytProvider) ParsePDFText(ctx context.Context, text string) ([]models.ActualBudgetReport, error) {
	logger := slog.With("provider", "ryt", "format", "pdf")

	accountName := extractAccountName(text)
	reports, err := parseBlocks(text)
	if err != nil {
		return nil, err
	}

	logger.InfoContext(ctx, "pdf parsing started", "blocks", len(reports), "account", accountName)

	result := p.toActualReports(ctx, logger, reports, accountName)
	logger.InfoContext(ctx, "pdf parsing complete", "parsed_count", len(result))
	if len(result) == 0 {
		return nil, errors.New("no transactions found after filtering")
	}
	return result, nil
}

func (p *RytProvider) toActualReports(ctx context.Context, logger *slog.Logger, reports []RytReport, accountName string) []models.ActualBudgetReport {
	accountName = p.engine.MapAccount(accountName)
	var result []models.ActualBudgetReport

	for _, report := range reports {
		if strings.Contains(strings.ToLower(report.Description), "opening balance") {
			logger.DebugContext(ctx, "row skipped: opening balance", "description", report.Description)
			continue
		}

		if p.engine.ShouldSkip(report.Description) {
			logger.DebugContext(ctx, "row skipped: filtered description", "description", report.Description)
			continue
		}

		parsedDate, err := time.Parse("2 January 2006", report.Date)
		if err != nil {
			logger.DebugContext(ctx, "row skipped: invalid date", "raw", report.Date)
			continue
		}

		description := textutil.Collapse(report.Description)

		amount, err := textutil.ParseAmount(report.Amount)
		if err != nil || amount == 0 {
			logger.DebugContext(ctx, "row skipped: invalid amount", "raw", report.Amount)
			continue
		}

		grp, cat := p.engine.MatchCategory(description)

		result = append(result, models.ActualBudgetReport{
			Account:       accountName,
			Date:          parsedDate.Format("2006-01-02"),
			Payee:         "",
			Notes:         description,
			CategoryGroup: grp,
			Category:      cat,
			Amount:        strconv.FormatFloat(amount, 'f', 2, 64),
		})
	}

	return result
}

func (p *RytProvider) ExtractionMethod() pdfutil.ExtractionMethod {
	return pdfutil.ExtractionMethodDigital
}
