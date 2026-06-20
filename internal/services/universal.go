package services

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"reflect"

	"actual-helper/internal/models"
)

func ToActualCSV(reports []models.ActualBudgetReport) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	header, err := csvHeader()
	if err != nil {
		return nil, err
	}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("write header: %w", err)
	}

	for _, report := range reports {
		row := csvRow(report)
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

func csvHeader() ([]string, error) {
	reportType := reflect.TypeFor[models.ActualBudgetReport]()
	columns := make([]string, reportType.NumField())
	for i := range reportType.NumField() {
		tag := reportType.Field(i).Tag.Get("csv")
		if tag == "" {
			return nil, fmt.Errorf("field %q missing csv tag", reportType.Field(i).Name)
		}
		columns[i] = tag
	}
	return columns, nil
}

func csvRow(report models.ActualBudgetReport) []string {
	reportValue := reflect.ValueOf(report)
	columns := make([]string, reportValue.NumField())
	for i := range reportValue.NumField() {
		columns[i] = reportValue.Field(i).String()
	}
	return columns
}
