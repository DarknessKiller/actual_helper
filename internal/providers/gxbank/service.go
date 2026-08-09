package gxbank

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"actual_helper/internal/models"
	"actual_helper/internal/pdfutil"
	"actual_helper/internal/providers"
	"actual_helper/internal/providers/cardutil"
	"actual_helper/internal/rule"
)

type GXBankProvider struct {
	engine         *rule.Engine
	mu             sync.RWMutex
	accountMapping map[string]string
}

func New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider {
	return &GXBankProvider{
		engine:         rule.NewEngine(excludeKeywords, includeKeywords, categories),
		accountMapping: accountMappings,
	}
}

func (p *GXBankProvider) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) {
	p.engine.Reload(excludeKeywords, includeKeywords, categories)
	p.mu.Lock()
	p.accountMapping = accountMappings
	p.mu.Unlock()
}

func (p *GXBankProvider) Name() string {
	return "gxbank"
}

func (p *GXBankProvider) ParseCSV(ctx context.Context, r io.Reader) ([]models.ActualBudgetReport, error) {
	records, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, errors.New("no CSV records found")
	}
	header := make(map[string]int)
	for i, name := range records[0] {
		header[strings.TrimSpace(name)] = i
	}
	var reports []GXReport
	for _, row := range records[1:] {
		get := func(name string) string {
			if i, ok := header[name]; ok && i < len(row) {
				return row[i]
			}
			return ""
		}
		amount := get("Amount")
		reports = append(reports, GXReport{Date: get("Date"), Description: get("Description"), Amount: amount, IsCredit: !strings.HasPrefix(strings.TrimSpace(amount), "-")})
	}
	result := p.toActualReports(ctx, slog.With("provider", "gxbank", "format", "csv"), reports, "")
	if len(result) == 0 {
		return nil, errors.New("no transactions found after filtering")
	}
	return result, nil
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
	var result []models.ActualBudgetReport

	p.mu.RLock()
	if p.accountMapping != nil {
		if mapped, ok := p.accountMapping[accountName]; ok {
			accountName = mapped
		}
	}
	p.mu.RUnlock()

	for _, report := range reports {
		if p.shouldSkip(report.Description) {
			logger.DebugContext(ctx, "row skipped: filtered description", "description", report.Description)
			continue
		}

		parsedDate, err := parseDate(report.Date)
		if err != nil {
			logger.DebugContext(ctx, "row skipped: invalid date", "raw", report.Date)
			continue
		}

		description := strings.TrimSpace(cardutil.WhitespaceRe.ReplaceAllString(report.Description, " "))

		amountStr := strings.TrimPrefix(strings.TrimPrefix(report.Amount, "+"), "-")
		amountStr = strings.ReplaceAll(amountStr, ",", "")
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

func (p *GXBankProvider) shouldSkip(description string) bool {
	return p.engine.ShouldSkip(description)
}

func (p *GXBankProvider) matchCategory(description string) (string, string) {
	return p.engine.MatchCategory(description)
}

func (p *GXBankProvider) ExtractionMethod() pdfutil.ExtractionMethod {
	return pdfutil.ExtractionMethodDigital
}

func parseDate(raw string) (time.Time, error) {
	formats := []string{"2 January 2006", "2 Jan 2006"}
	for _, fmt := range formats {
		t, err := time.Parse(fmt, raw)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid date format")
}
