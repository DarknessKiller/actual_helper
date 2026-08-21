package services_test

import (
	"testing"

	"actual_helper/internal/models"
	"actual_helper/internal/services"
)

// benchReports is a fixed, fake slice of ActualBudgetReport used to measure
// services.ToActualCSV (the per-request output step) in isolation. No real
// personal data.
func benchReports(n int) []models.ActualBudgetReport {
	reports := make([]models.ActualBudgetReport, n)
	for i := range reports {
		reports[i] = models.ActualBudgetReport{
			Account:       "Wallet",
			Date:          "2026-01-02",
			Payee:         "",
			Notes:         "Sample Merchant",
			CategoryGroup: "Food & Dining",
			Category:      "Delivery",
			Amount:        "-12.34",
			SplitAmount:   "",
			Cleared:       "",
		}
	}
	return reports
}

// BenchmarkToActualCSV measures the CSV serialisation step that runs on every
// ConvertFile response. v0.3.1 had no benchmarks; this establishes a baseline
// for the output path.
func BenchmarkToActualCSV(b *testing.B) {
	reports := benchReports(100)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := services.ToActualCSV(reports); err != nil {
			b.Fatal(err)
		}
	}
}
