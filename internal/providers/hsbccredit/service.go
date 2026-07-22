package hsbccredit

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"actual_helper/internal/models"
	"actual_helper/internal/pdfutil"
	"actual_helper/internal/providers"
	"actual_helper/internal/providers/cardutil"
	"actual_helper/internal/rule"
)

type HSBCProvider struct {
	engine         *rule.Engine
	mu             sync.RWMutex
	accountMapping map[string]string
}

func New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider {
	return &HSBCProvider{
		engine:         rule.NewEngine(excludeKeywords, includeKeywords, categories),
		accountMapping: accountMappings,
	}
}

func (p *HSBCProvider) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) {
	p.engine.Reload(excludeKeywords, includeKeywords, categories)
	p.mu.Lock()
	p.accountMapping = accountMappings
	p.mu.Unlock()
}

func (p *HSBCProvider) shouldSkip(description string) bool {
	return p.engine.ShouldSkip(description)
}

func (p *HSBCProvider) matchCategory(description string) (string, string) {
	return p.engine.MatchCategory(description)
}

func (p *HSBCProvider) Name() string {
	return "hsbccredit"
}

func (p *HSBCProvider) ParseCSV(ctx context.Context, r io.Reader) ([]models.ActualBudgetReport, error) {
	return nil, errors.New("not supported for hsbc provider")
}

func (p *HSBCProvider) ParsePDFText(ctx context.Context, text string) ([]models.ActualBudgetReport, error) {
	logger := slog.With("provider", "hsbccredit", "format", "pdf")

	accountName := extractAccountName(text)
	reports, err := parseTransactions(text)
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

func (p *HSBCProvider) toActualReports(ctx context.Context, logger *slog.Logger, reports []HSBCReport, accountName string) []models.ActualBudgetReport {
	var result []models.ActualBudgetReport

	// Apply account mapping once before the loop
	p.mu.RLock()
	accountName = cardutil.ApplyMapping(p.accountMapping, accountName)
	p.mu.RUnlock()

	for _, report := range reports {
		if p.shouldSkip(report.Description) {
			logger.DebugContext(ctx, "row skipped: filtered description", "description", report.Description)
			continue
		}

		description := strings.TrimSpace(cardutil.WhitespaceRe.ReplaceAllString(report.Description, " "))

		amountStr := strings.ReplaceAll(report.Amount, ",", "")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || amount == 0 {
			logger.DebugContext(ctx, "row skipped: invalid amount", "raw", report.Amount)
			continue
		}

		if !report.IsCredit {
			amount = -amount
		}

		categoryGroup, category := p.matchCategory(description)

		result = append(result, models.ActualBudgetReport{
			Account:       accountName,
			Date:          report.TransDate,
			Payee:         "",
			Notes:         description,
			CategoryGroup: categoryGroup,
			Category:      category,
			Amount:        strconv.FormatFloat(amount, 'f', 2, 64),
		})
	}

	return result
}

func (p *HSBCProvider) ExtractionMethod() pdfutil.ExtractionMethod {
	return pdfutil.ExtractionMethodOCR
}
