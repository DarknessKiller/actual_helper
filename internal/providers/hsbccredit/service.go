package hsbccredit

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"actual_helper/internal/models"
	"actual_helper/internal/pdfutil"
	"actual_helper/internal/providers"
	"actual_helper/internal/rule"
)

var monthNames = map[string]time.Month{
	"JAN": time.January, "FEB": time.February, "MAR": time.March,
	"APR": time.April, "MAY": time.May, "JUN": time.June,
	"JUL": time.July, "AUG": time.August, "SEP": time.September,
	"OCT": time.October, "NOV": time.November, "DEC": time.December,
}

type HSBCProvider struct {
	engine         *rule.Engine
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
	p.accountMapping = accountMappings
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
	return result, nil
}

var whitespacePattern = regexp.MustCompile(`\s+`)

func (p *HSBCProvider) toActualReports(ctx context.Context, logger *slog.Logger, reports []HSBCReport, accountName string) []models.ActualBudgetReport {
	var result []models.ActualBudgetReport

	for _, report := range reports {
		if p.shouldSkip(report.Description) {
			logger.DebugContext(ctx, "row skipped: filtered description", "description", report.Description)
			continue
		}

		description := strings.TrimSpace(whitespacePattern.ReplaceAllString(report.Description, " "))

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

		if p.accountMapping != nil {
			if mapped, ok := p.accountMapping[accountName]; ok {
				accountName = mapped
			}
		}

		result = append(result, models.ActualBudgetReport{
			Account:       accountName,
			Date:          report.PostDate,
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
