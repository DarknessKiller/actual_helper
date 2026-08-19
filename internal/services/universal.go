package services

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"reflect"

	"actual_helper/internal/models"
)

var cachedHeader []string

func init() {
	reportType := reflect.TypeFor[models.ActualBudgetReport]()
	cachedHeader = make([]string, reportType.NumField())
	for i := range reportType.NumField() {
		tag := reportType.Field(i).Tag.Get("csv")
		if tag == "" {
			panic(fmt.Sprintf("field %q missing csv tag", reportType.Field(i).Name))
		}
		cachedHeader[i] = tag
	}
}

func ToActualCSV(reports []models.ActualBudgetReport) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	if err := writer.Write(cachedHeader); err != nil {
		return nil, fmt.Errorf("write header: %w", err)
	}

	row := make([]string, len(cachedHeader))
	for _, report := range reports {
		csvRow(report, row)
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

func csvRow(report models.ActualBudgetReport, row []string) {
	row[0] = report.Account
	row[1] = report.Date
	row[2] = report.Payee
	row[3] = report.Notes
	row[4] = report.CategoryGroup
	row[5] = report.Category
	row[6] = report.Amount
	row[7] = report.SplitAmount
	row[8] = report.Cleared
}
