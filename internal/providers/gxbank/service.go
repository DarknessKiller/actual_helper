package gxbank

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"actual_helper/internal/models"
	"actual_helper/internal/pdfutil"
	"actual_helper/internal/providers"
	"actual_helper/internal/rule"
	"actual_helper/internal/textutil"
)

type GXBankProvider struct {
	engine *rule.Engine
}

func New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider {
	return &GXBankProvider{engine: rule.NewEngine(excludeKeywords, includeKeywords, categories, accountMappings)}
}

func (p *GXBankProvider) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) {
	p.engine.Reload(excludeKeywords, includeKeywords, categories, accountMappings)
}

func (p *GXBankProvider) Name() string { return "gxbank" }

func (p *GXBankProvider) ParseCSV(_ context.Context, _ io.Reader) ([]models.ActualBudgetReport, error) {
	return nil, errors.New("not supported for gxbank provider")
}

func (p *GXBankProvider) ParsePDFText(ctx context.Context, text string) ([]models.ActualBudgetReport, error) {
	logger := slog.With("provider", "gxbank", "format", "pdf")

	accountName := ExtractAccountName(text)
	reports, err := ParsePDFBlocks(text)
	if err != nil {
		return nil, err
	}

	logger.InfoContext(ctx, "pdf parsing started", "transactions", len(reports), "account", accountName)

	result := p.toActualReports(ctx, logger, reports, accountName)
	logger.InfoContext(ctx, "pdf parsing complete", "parsed_count", len(result))
	if len(result) == 0 {
		return nil, errors.New("no transactions found after filtering")
	}
	return result, nil
}

func (p *GXBankProvider) toActualReports(ctx context.Context, logger *slog.Logger, reports []GXReport, accountName string) []models.ActualBudgetReport {
	accountName = p.engine.MapAccount(accountName)
	var result []models.ActualBudgetReport

	for _, report := range reports {
		if p.engine.ShouldSkip(report.Description) {
			logger.DebugContext(ctx, "row skipped: filtered description", "description", report.Description)
			continue
		}

		parsedDate, err := textutil.ParseDateMulti(report.Date, "2 January 2006", "2 Jan 2006")
		if err != nil {
			logger.DebugContext(ctx, "row skipped: invalid date", "raw", report.Date)
			continue
		}

		description := textutil.Collapse(report.Description)

		amountStr := strings.TrimPrefix(strings.TrimPrefix(report.Amount, "+"), "-")
		amount, err := textutil.ParseAmount(amountStr)
		if err != nil || amount == 0 {
			logger.DebugContext(ctx, "row skipped: invalid amount", "raw", report.Amount)
			continue
		}

		if !report.IsCredit {
			amount = -amount
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

func (p *GXBankProvider) ExtractionMethod() pdfutil.ExtractionMethod {
	return pdfutil.ExtractionMethodDigital
}
