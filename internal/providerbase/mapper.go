// Package providerbase holds helpers shared by credit-card-style providers.
package providerbase

import (
	"context"
	"errors"
	"log/slog"
	"strconv"

	"actual_helper/internal/models"
	"actual_helper/internal/rule"
	"actual_helper/internal/textutil"
)

// PDFReport is the parsed row shape every credit-card provider feeds
// into the shared mapper.
type PDFReport struct {
	TransDate   string
	Description string
	Amount      string
	IsCredit    bool
}

// MapReports filters by engine rules and emits ActualBudgetReport rows.
func MapReports(ctx context.Context, logger *slog.Logger, engine *rule.Engine, reports []PDFReport, accountName string) []models.ActualBudgetReport {
	accountName = engine.MapAccount(accountName)
	var result []models.ActualBudgetReport
	for _, report := range reports {
		if engine.ShouldSkip(report.Description) {
			logger.DebugContext(ctx, "row skipped: filtered description", "description", report.Description)
			continue
		}
		description := textutil.Collapse(report.Description)
		amount, err := textutil.ParseAmount(report.Amount)
		if err != nil || amount == 0 {
			logger.DebugContext(ctx, "row skipped: invalid amount", "raw", report.Amount)
			continue
		}
		if !report.IsCredit {
			amount = -amount
		}
		grp, cat := engine.MatchCategory(description)
		result = append(result, models.ActualBudgetReport{
			Account:       accountName,
			Date:          report.TransDate,
			Payee:         "",
			Notes:         description,
			CategoryGroup: grp,
			Category:      cat,
			Amount:        strconv.FormatFloat(amount, 'f', 2, 64),
		})
	}
	return result
}

// ErrNoTransactions is the canonical "nothing left after filtering" error.
var ErrNoTransactions = errors.New("no transactions found after filtering")
