package hlb

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
	accountName := cardutil.ExtractAfterMarker(text, "Credit Card Number", "HLB Credit Card")
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
	accountName := extractAccountFromMarkers(text, []string{"A/C No", "No Akaun"}, "HLB Debit Account")

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

func extractAccountFromMarkers(text string, markers []string, fallback string) string {
	for _, marker := range markers {
		if idx := strings.Index(text, marker); idx != -1 {
			return extractAccountFromMarker(text, idx, fallback)
		}
	}
	return fallback
}

func extractAccountFromMarker(text string, idx int, fallback string) string {
	line := text[idx:]
	newlineIdx := strings.Index(line, "\n")
	if newlineIdx == -1 {
		newlineIdx = len(line)
	}
	sameLine := line[:newlineIdx]
	if colonIdx := strings.Index(sameLine, ":"); colonIdx != -1 {
		value := strings.TrimSpace(sameLine[colonIdx+1:])
		if spaceIdx := strings.Index(value, " "); spaceIdx != -1 {
			value = value[:spaceIdx]
		}
		value = strings.ReplaceAll(value, " ", "")
		value = strings.ReplaceAll(value, "/", "")
		value = strings.ReplaceAll(value, "-", "")
		if len(value) > 0 {
			return value
		}
	} else if newlineIdx < len(line) {
		remaining := text[idx+newlineIdx+1:]
		lines := strings.SplitN(remaining, "\n", 3)
		for _, l := range lines {
			trimmed := strings.TrimSpace(l)
			if strings.HasPrefix(trimmed, ":") {
				valueLine := strings.TrimSpace(trimmed[1:])
				valueLine = strings.ReplaceAll(valueLine, " ", "")
				valueLine = strings.ReplaceAll(valueLine, "/", "")
				valueLine = strings.ReplaceAll(valueLine, "-", "")
				if len(valueLine) > 0 {
					return valueLine
				}
				break
			}
		}
	}
	return fallback
}

func (p *HLBProvider) ExtractionMethod() pdfutil.ExtractionMethod {
	return pdfutil.ExtractionMethodPdftotext
}
