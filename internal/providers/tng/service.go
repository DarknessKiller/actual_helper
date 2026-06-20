package tng

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"actual-helper/internal/models"
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
	excludeKeywords []string
	includeKeywords []string
	categories      []models.CategoryRule
	mu              sync.RWMutex
}

func New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule) *TNGProvider {
	eks := make([]string, len(excludeKeywords))
	copy(eks, excludeKeywords)
	iks := make([]string, len(includeKeywords))
	copy(iks, includeKeywords)
	cats := make([]models.CategoryRule, len(categories))
	copy(cats, categories)

	return &TNGProvider{
		excludeKeywords: eks,
		includeKeywords: iks,
		categories:      cats,
	}
}

func (p *TNGProvider) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.excludeKeywords = make([]string, len(excludeKeywords))
	copy(p.excludeKeywords, excludeKeywords)
	p.includeKeywords = make([]string, len(includeKeywords))
	copy(p.includeKeywords, includeKeywords)
	p.categories = make([]models.CategoryRule, len(categories))
	copy(p.categories, categories)
}

func (p *TNGProvider) shouldSkip(description string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	lower := strings.ToLower(description)

	for _, kw := range p.includeKeywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return false
		}
	}

	for _, kw := range p.excludeKeywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}

	return false
}

func (p *TNGProvider) matchCategory(description string) (string, string) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	lower := strings.ToLower(description)

	for _, r := range p.categories {
		if strings.Contains(lower, strings.ToLower(r.Keyword)) {
			return r.Group, r.Category
		}
	}

	return "", ""
}

func (p *TNGProvider) Name() string {
	return "tng"
}

func (p *TNGProvider) ParseCSV(ctx context.Context, fileReader io.Reader) ([]models.ActualBudgetReport, error) {
	logger := slog.With("provider", "tng", "format", "csv")

	csvReader := csv.NewReader(fileReader)
	csvReader.LazyQuotes = true
	csvReader.FieldsPerRecord = -1

	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}

	logger.InfoContext(ctx, "csv parsing started", "total_rows", len(records))

	if len(records) < 2 {
		logger.InfoContext(ctx, "csv parsing complete", "parsed_count", 0)
		return nil, nil
	}

	columnIndex := buildIndex(records[0])
	var reports []TNGReport

	for i, row := range records[1:] {
		report, err := parseRow(columnIndex, row)
		if err != nil {
			logger.DebugContext(ctx, "row skipped", "row", i+1, "reason", err.Error())
			continue
		}
		reports = append(reports, report)
	}

	result := p.toActualReports(ctx, logger, reports)
	logger.InfoContext(ctx, "csv parsing complete", "parsed_count", len(result))
	return result, nil
}

func (p *TNGProvider) toActualReports(ctx context.Context, logger *slog.Logger, reports []TNGReport) []models.ActualBudgetReport {
	whitespacePattern := regexp.MustCompile(`\s+`)
	var result []models.ActualBudgetReport

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

		description := strings.TrimSpace(whitespacePattern.ReplaceAllString(report.Description, " "))

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
			Account:       "Current",
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

	result := p.toActualReports(ctx, logger, reports)
	logger.InfoContext(ctx, "pdf parsing complete", "parsed_count", len(result))
	return result, nil
}

func buildIndex(header []string) map[string]int {
	columnIndex := make(map[string]int, len(header))
	for i, name := range header {
		columnIndex[strings.TrimSpace(name)] = i
	}
	return columnIndex
}

func lookup(columnIndex map[string]int, row []string, name string) string {
	index, ok := columnIndex[name]
	if !ok || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func parseRow(columnIndex map[string]int, row []string) (TNGReport, error) {
	report := TNGReport{
		Date:            lookup(columnIndex, row, "F"),
		Status:          lookup(columnIndex, row, "Status"),
		TransactionType: lookup(columnIndex, row, "Transaction Type"),
		Reference:       lookup(columnIndex, row, "Reference"),
		Description:     lookup(columnIndex, row, "Description"),
		Details:         lookup(columnIndex, row, "Details"),
		Amount:          lookup(columnIndex, row, "Amount(RM)"),
	}
	if report.Date == "" || report.Status == "" || report.Description == "" || report.Amount == "" {
		return report, errors.New("missing required column")
	}
	return report, nil
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
