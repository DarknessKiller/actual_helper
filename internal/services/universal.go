package services

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"actual_helper/internal/models"
)

var actualBudgetReportHeader = []string{
	"Account", "Date", "Payee", "Notes",
	"Category_Group", "Category", "Amount", "Split_Amount", "Cleared",
}

func ToActualCSV(reports []models.ActualBudgetReport) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	if err := writer.Write(actualBudgetReportHeader); err != nil {
		return nil, fmt.Errorf("write header: %w", err)
	}

	for _, report := range reports {
		row := []string{
			report.Account, report.Date, report.Payee, report.Notes,
			report.CategoryGroup, report.Category, report.Amount,
			report.SplitAmount, report.Cleared,
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("write row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("csv flush: %w", err)
	}

	return buffer.Bytes(), nil
}
