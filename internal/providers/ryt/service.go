package ryt

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
	"actual_helper/internal/providers"
	"actual_helper/internal/rule"
)

type RytProvider struct {
	engine         *rule.Engine
	accountMapping map[string]string
}

func New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider {
	return &RytProvider{
		engine:         rule.NewEngine(excludeKeywords, includeKeywords, categories),
		accountMapping: accountMappings,
	}
}

func (p *RytProvider) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) {
	p.engine.Reload(excludeKeywords, includeKeywords, categories)
	p.accountMapping = accountMappings
}

func (p *RytProvider) shouldSkip(description string) bool {
	return p.engine.ShouldSkip(description)
}

func (p *RytProvider) matchCategory(description string) (string, string) {
	return p.engine.MatchCategory(description)
}

func (p *RytProvider) Name() string {
	return "ryt"
}

func (p *RytProvider) ParseCSV(ctx context.Context, r io.Reader) ([]models.ActualBudgetReport, error) {
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
	return result, nil
}

func (p *RytProvider) toActualReports(ctx context.Context, logger *slog.Logger, reports []RytReport, accountName string) []models.ActualBudgetReport {
	whitespacePattern := regexp.MustCompile(`\s+`)
	var result []models.ActualBudgetReport

	for _, report := range reports {
		if strings.Contains(strings.ToLower(report.Description), "opening balance") {
			logger.DebugContext(ctx, "row skipped: opening balance", "description", report.Description)
			continue
		}

		if p.shouldSkip(report.Description) {
			logger.DebugContext(ctx, "row skipped: filtered description", "description", report.Description)
			continue
		}

		parsedDate, err := time.Parse("2 January 2006", report.Date)
		if err != nil {
			logger.DebugContext(ctx, "row skipped: invalid date", "raw", report.Date)
			continue
		}

		description := strings.TrimSpace(whitespacePattern.ReplaceAllString(report.Description, " "))

		amountStr := strings.ReplaceAll(report.Amount, ",", "")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || amount == 0 {
			logger.DebugContext(ctx, "row skipped: invalid amount", "raw", report.Amount)
			continue
		}

		categoryGroup, category := p.matchCategory(description)

		if p.accountMapping != nil {
			if mapped, ok := p.accountMapping[accountName]; ok {
				accountName = mapped
			}
		}

		result = append(result, models.ActualBudgetReport{
			Account:       accountName,
			Date:          parsedDate.Format("2006-01-02"),
			Payee:         "",
			Notes:         description,
			CategoryGroup: categoryGroup,
			Category:      category,
			Amount:        strconv.FormatFloat(amount, 'f', 2, 64),
		})
	}

	return result
}
