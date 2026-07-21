package hlb

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"actual_helper/internal/models"
	"actual_helper/internal/pdfutil"
	"actual_helper/internal/providers"
	"actual_helper/internal/providers/cardutil"
	"actual_helper/internal/rule"
)

type HLBProvider struct {
	engine         *rule.Engine
	mu             sync.RWMutex
	accountMapping map[string]string
}

func New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider {
	return &HLBProvider{
		engine:         rule.NewEngine(excludeKeywords, includeKeywords, categories),
		accountMapping: accountMappings,
	}
}

func (p *HLBProvider) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) {
	p.engine.Reload(excludeKeywords, includeKeywords, categories)
	p.mu.Lock()
	p.accountMapping = accountMappings
	p.mu.Unlock()
}

func (p *HLBProvider) shouldSkip(description string) bool {
	return p.engine.ShouldSkip(description)
}

func (p *HLBProvider) matchCategory(description string) (string, string) {
	return p.engine.MatchCategory(description)
}

func (p *HLBProvider) Name() string {
	return "hlb"
}

func (p *HLBProvider) ParseCSV(ctx context.Context, r io.Reader) ([]models.ActualBudgetReport, error) {
	return nil, errors.New("not supported for hlb provider")
}

func (p *HLBProvider) ParsePDFText(ctx context.Context, text string) ([]models.ActualBudgetReport, error) {
	logger := slog.With("provider", "hlb", "format", "pdf")

	format := DetectFormat(text)
	switch format {
	case "credit":
		return p.parseCreditPDF(ctx, logger, text)
	case "debit":
		return p.parseDebitPDF(ctx, logger, text)
	default:
		return nil, errors.New("unable to detect HLB statement format")
	}
}

func (p *HLBProvider) parseCreditPDF(ctx context.Context, logger *slog.Logger, text string) ([]models.ActualBudgetReport, error) {
	accountName := extractCreditAccountName(text)
	reports, err := parseCreditTransactions(text)
	if err != nil {
		return nil, err
	}

	logger.InfoContext(ctx, "pdf parsing started", "transactions", len(reports), "account", accountName, "type", "credit")

	result := p.toActualReports(ctx, logger, reports, accountName)
	logger.InfoContext(ctx, "pdf parsing complete", "parsed_count", len(result))
	if len(result) == 0 {
		return nil, errors.New("no transactions found after filtering")
	}
	return result, nil
}

func (p *HLBProvider) parseDebitPDF(ctx context.Context, logger *slog.Logger, text string) ([]models.ActualBudgetReport, error) {
	accountName := extractDebitAccountName(text)
	reports, err := parseDebitTransactions(text)
	if err != nil {
		return nil, err
	}

	logger.InfoContext(ctx, "pdf parsing started", "transactions", len(reports), "account", accountName, "type", "debit")

	result := p.toActualReports(ctx, logger, reports, accountName)
	logger.InfoContext(ctx, "pdf parsing complete", "parsed_count", len(result))
	if len(result) == 0 {
		return nil, errors.New("no transactions found after filtering")
	}
	return result, nil
}

var whitespacePattern = regexp.MustCompile(`\s+`)

func (p *HLBProvider) toActualReports(ctx context.Context, logger *slog.Logger, reports []HLBReport, accountName string) []models.ActualBudgetReport {
	var result []models.ActualBudgetReport

	p.mu.RLock()
	accountName = cardutil.ApplyMapping(p.accountMapping, accountName)
	p.mu.RUnlock()

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

func (p *HLBProvider) ExtractionMethod() pdfutil.ExtractionMethod {
	return pdfutil.ExtractionMethodPdftotext
}
