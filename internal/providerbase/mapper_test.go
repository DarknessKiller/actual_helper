package providerbase

import (
	"context"
	"log/slog"
	"testing"

	"actual_helper/internal/models"
	"actual_helper/internal/rule"

	"github.com/stretchr/testify/require"
)

func TestMapReports_Empty(t *testing.T) {
	engine := rule.NewEngine(nil, nil, nil, nil)
	got := MapReports(context.Background(), slog.Default(), engine, nil, "acc")
	require.Empty(t, got)
}

func TestMapReports_AppliesMapping(t *testing.T) {
	engine := rule.NewEngine(nil, nil, nil, map[string]string{"raw": "MAPPED"})
	reports := []PDFReport{
		{TransDate: "2026-01-01", Description: "Item", Amount: "10.00", IsCredit: true},
	}
	got := MapReports(context.Background(), slog.Default(), engine, reports, "raw")
	require.Len(t, got, 1)
	require.Equal(t, "MAPPED", got[0].Account)
	require.Equal(t, "10.00", got[0].Amount)
}

func TestMapReports_NegatesDebit(t *testing.T) {
	engine := rule.NewEngine(nil, nil, nil, nil)
	reports := []PDFReport{
		{TransDate: "2026-01-01", Description: "Purchase", Amount: "5.00", IsCredit: false},
		{TransDate: "2026-01-01", Description: "Refund", Amount: "5.00", IsCredit: true},
	}
	got := MapReports(context.Background(), slog.Default(), engine, reports, "acc")
	require.Len(t, got, 2)
	require.Equal(t, "-5.00", got[0].Amount)
	require.Equal(t, "5.00", got[1].Amount)
}

func TestMapReports_AppliesExcludeAndCategory(t *testing.T) {
	engine := rule.NewEngine(
		[]string{"skip"},
		nil,
		[]models.CategoryRule{{Keyword: "grab", Group: "Food", Category: "Delivery"}},
		nil,
	)
	reports := []PDFReport{
		{TransDate: "2026-01-01", Description: "skip this", Amount: "5.00"},
		{TransDate: "2026-01-01", Description: "GrabFood", Amount: "5.00", IsCredit: true},
	}
	got := MapReports(context.Background(), slog.Default(), engine, reports, "acc")
	require.Len(t, got, 1)
	require.Equal(t, "GrabFood", got[0].Notes)
	require.Equal(t, "Food", got[0].CategoryGroup)
	require.Equal(t, "Delivery", got[0].Category)
}
