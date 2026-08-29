package tng

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"time"

	"actual_helper/internal/models"
	"actual_helper/internal/pdfutil"
	"actual_helper/internal/providers"
	"actual_helper/internal/rule"
	"actual_helper/internal/textutil"
)

var creditTransactionTypes = map[string]struct{}{
	"Reload":              {},
	"Receive from Wallet": {},
	"DUITNOW_RECEIVEFROM": {},
	"Refund":              {},
	"GO+ Daily Earnings":  {},
	"GO+ Cash In":         {},
}

type TNGProvider struct {
	engine *rule.Engine
}

func New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider {
	return &TNGProvider{engine: rule.NewEngine(excludeKeywords, includeKeywords, categories, accountMappings)}
}

func (p *TNGProvider) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) {
	p.engine.Reload(excludeKeywords, includeKeywords, categories, accountMappings)
}

func (p *TNGProvider) Name() string { return "tng" }

func (p *TNGProvider) ParseCSV(_ context.Context, _ io.Reader) ([]models.ActualBudgetReport, error) {
	return nil, errors.New("not supported for tng provider")
}

func (p *TNGProvider) toActualReports(ctx context.Context, logger *slog.Logger, reports []TNGReport, accountName string) []models.ActualBudgetReport {
	accountName = p.engine.MapAccount(accountName)
	var result []models.ActualBudgetReport

	for _, report := range reports {
		if report.Status != "Success" {
			logger.DebugContext(ctx, "row skipped: non-success status", "status", report.Status)
			continue
		}

		if p.engine.ShouldSkip(report.Description) {
			logger.DebugContext(ctx, "row skipped: filtered description", "description", report.Description)
			continue
		}

		parsedDate, err := textutil.ParseDateMulti(report.Date, "2/1/2006", "02/01/2006")
		if err != nil {
			logger.DebugContext(ctx, "row skipped: invalid date", "raw", report.Date)
			continue
		}

		description := textutil.Collapse(report.Description)

		amount, err := textutil.ParseAmount(report.Amount, "RM")
		if err != nil || amount == 0 {
			logger.DebugContext(ctx, "row skipped: invalid amount", "raw", report.Amount)
			continue
		}

		if !isCredit(report.TransactionType) {
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
	_, ok := creditTransactionTypes[transactionType]
	return ok
}

func (p *TNGProvider) ExtractionMethod() pdfutil.ExtractionMethod {
	return pdfutil.ExtractionMethodDigital
}

// time import retained for refactor compatibility with future providers.
var _ = time.Time{}
