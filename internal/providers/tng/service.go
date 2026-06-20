package tng

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"actual-helper/internal/models"
)

type TNGProvider struct {
	categoriesPath string
	rules          []rule
	lastRuleLoad   time.Time
}

func New() *TNGProvider {
	provider := &TNGProvider{}
	if path := os.Getenv("TNG_CATEGORIES_PATH"); path != "" {
		provider.categoriesPath = path
		slog.Info("using categories config", "path", path)
	}
	return provider
}

func (provider *TNGProvider) Name() string {
	return "tng"
}

func (provider *TNGProvider) ParseCSV(ctx context.Context, fileReader io.Reader) ([]models.ActualBudgetReport, error) {
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

	result := provider.toActualReports(ctx, logger, reports)
	logger.InfoContext(ctx, "csv parsing complete", "parsed_count", len(result))
	return result, nil
}

func (provider *TNGProvider) toActualReports(ctx context.Context, logger *slog.Logger, reports []TNGReport) []models.ActualBudgetReport {
	provider.ensureRulesLoaded()
	whitespacePattern := regexp.MustCompile(`\s+`)
	var result []models.ActualBudgetReport

	for _, report := range reports {
		if report.Status != "Success" {
			logger.DebugContext(ctx, "row skipped: non-success status", "status", report.Status)
			continue
		}

		if strings.Contains(report.Description, "Quick Reload Payment") ||
			strings.Contains(report.Description, "Daily Interest") ||
			strings.Contains(report.Description, "Via eWallet to GO+") {
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
		if err != nil {
			logger.DebugContext(ctx, "row skipped: invalid amount", "raw", report.Amount)
			continue
		}

		if !isCredit(report.TransactionType) {
			amount = -amount
		}

		categoryGroup, category := match(provider.rules, description)

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

func (provider *TNGProvider) ParsePDFText(ctx context.Context, text string) ([]models.ActualBudgetReport, error) {
	logger := slog.With("provider", "tng", "format", "pdf")

	reports, err := parsePDFBlocks(text)
	if err != nil {
		return nil, err
	}

	logger.InfoContext(ctx, "pdf parsing started", "blocks", len(reports))

	result := provider.toActualReports(ctx, logger, reports)
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
	return transactionType == "Reload" ||
		strings.Contains(transactionType, "Receive from Wallet") ||
		strings.Contains(transactionType, "DUITNOW_RECEIVEFROM") ||
		strings.Contains(transactionType, "Refund")
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
