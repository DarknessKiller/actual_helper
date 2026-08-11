package services_test

import (
	"testing"

	"actual_helper/internal/models"
	"actual_helper/internal/services"
)

var testReport = models.ActualBudgetReport{
	Account:       "TNG",
	Date:          "2025-01-15",
	Payee:         "",
	Notes:         "GrabFood Order",
	CategoryGroup: "Food",
	Category:      "Delivery",
	Amount:        "-25.50",
	SplitAmount:   "",
	Cleared:       "Cleared",
}

func BenchmarkToActualCSV(b *testing.B) {
	reports := make([]models.ActualBudgetReport, 100)
	for i := range reports {
		reports[i] = testReport
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = services.ToActualCSV(reports)
	}
}
