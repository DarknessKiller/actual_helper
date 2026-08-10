package tng

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
	"actual_helper/internal/providers"
	"actual_helper/internal/providers/cardutil"
	"actual_helper/internal/rule"
)

type TNGTransactionType string

const (
	Reload             TNGTransactionType = "Reload"
	ReceiveFromWallet  TNGTransactionType = "Receive from Wallet"
	DuitNowReceiveFrom TNGTransactionType = "DUITNOW_RECEIVEFROM"
	Refund             TNGTransactionType = "Refund"
	GODailyEarnings    TNGTransactionType = "GO+ Daily Earnings"
	GOPlusCashIn       TNGTransactionType = "GO+ Cash In"
)

var creditTransactionTypes = map[TNGTransactionType]struct{}{
	Reload:             {},
	ReceiveFromWallet:  {},
	DuitNowReceiveFrom: {},
	Refund:             {},
	GODailyEarnings:    {},
	GOPlusCashIn:       {},
}

type TNGProvider struct {
	engine         *rule.Engine
	mu             sync.RWMutex
	accountMapping map[string]string
}

func New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider {
	return &TNGProvider{
		engine:         rule.NewEngine(excludeKeywords, includeKeywords, categories),
		accountMapping: accountMappings,
	}
}


func (p *TNGProvider) shouldSkip(description string) bool {
	return p.engine.ShouldSkip(description)
}

func (p *TNGProvider) matchCategory(description string) (string, string) {
	return p.engine.MatchCategory(description)
}

func (p *TNGProvider) Name() string {
	return "tng"
}

func (p *TNGProvider) ParseCSV(ctx context.Context, r io.Reader) ([]models.ActualBudgetReport, error) {
	records, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, errors.New("no CSV records found")
	}
	var reports []TNGReport
	header := make(map[string]int)
	for i, name := range records[0] {
		header[strings.TrimSpace(name)] = i
	}
	for _, row := range records[1:] {
		get := func(name string) string {
			if i, ok := header[name]; ok && i < len(row) {
				return row[i]
			}
			return ""
		}
		reports = append(reports, TNGReport{Date: get("F"), Status: get("Status"), TransactionType: get("Transaction Type"), Description: get("Description"), Amount: get("Amount(RM)")})
	}
	result := p.toActualReports(ctx, slog.With("provider", "tng", "format", "csv"), reports, "")
	if len(result) == 0 {
		return nil, errors.New("no transactions found after filtering")
	}
	return result, nil
}

func (p *TNGProvider) toActualReports(ctx context.Context, logger *slog.Logger, reports []TNGReport, accountName string) []models.ActualBudgetReport {
	var result []models.ActualBudgetReport

	// Apply account mapping once before the loop
	p.mu.RLock()
	if p.accountMapping != nil {
		if mapped, ok := p.accountMapping[accountName]; ok {
			accountName = mapped
		}
	}
	p.mu.RUnlock()

	for _, report := range reports {
		if report.Status != "Success" {
			logger.DebugContext(ctx, "row skipped: non-success status", "status", report.Status)
			continue
		}

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

		amount, err := parseAmount(report.Amount)
		if err != nil || amount == 0 {
			logger.DebugContext(ctx, "row skipped: invalid amount", "raw", report.Amount)
			continue
		}

		if !isCredit(report.TransactionType) {
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

func (p *TNGProvider) ParsePDFText(ctx context.Context, text string) ([]models.ActualBudgetReport, error) {
	logger := slog.With("provider", "tng", "format", "pdf")

	reports, err := parsePDFBlocks(text)
	if err != nil {
		return nil, err
	}

	logger.InfoContext(ctx, "pdf parsing started", "blocks", len(reports))

	result := p.toActualReports(ctx, logger, reports, "")
	logger.InfoContext(ctx, "pdf parsing complete", "parsed_count", len(result))
	if len(result) == 0 {
		return nil, errors.New("no transactions found after filtering")
	}
	return result, nil
}

func isCredit(transactionType string) bool {
	_, ok := creditTransactionTypes[TNGTransactionType(transactionType)]
	return ok
}

func parseDate(raw string) (time.Time, error) {
	t, err := time.Parse("2/1/2006", raw)
	if err != nil {
		t, err = time.Parse("02/01/2006", raw)
	}
	return t, err
}

func parseAmount(amountStr string) (float64, error) {
	amountStr = strings.ReplaceAll(amountStr, "RM", "")
	amountStr = strings.ReplaceAll(amountStr, ",", "")
	return strconv.ParseFloat(amountStr, 64)
}

