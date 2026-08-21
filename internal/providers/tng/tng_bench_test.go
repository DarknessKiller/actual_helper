package tng_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"actual_helper/internal/models"
	"actual_helper/internal/providers/tng"
)

// tng is a PDF-only provider: ParseCSV returns "not supported". The real TNG
// conversion path is ParsePDFText, so this benchmark exercises that on a small
// fixed columnar PDF text input (the format ledongthuc/pdf yields). All data
// is fake. v0.3.1 had no provider benchmarks; this establishes one.

// benchTNGText is a small fixed input with three Success transactions in the
// columnar cell-per-line shape the PDF extractor produces.
const benchTNGText = `TNG WALLET TRANSACTION
Date
Status
Transaction Type
Reference
Description
Details
Amount (RM)
Wallet Balance
1/5/2026
Success
Payment
111111
Merchant Alpha
222222
RM34.00
RM5.10
2/5/2026
Success
Reload
333333
Top Up from Bank

RM100.00
RM150.00
3/5/2026
Success
Payment
444444
Merchant Beta
555555
RM12.50
RM0.00`

// BenchmarkTNGParsePDFText measures the TNG provider parse path (the step
// ConvertFile runs after PDF extraction). It is benchmarked both with empty
// tuning (the default state until a config is loaded) and with a loaded
// config so the filtering/categorization branches are covered.
func BenchmarkTNGParsePDFText(b *testing.B) {
	// Silence the provider's per-call slog.Info logging so the bench measures
	// parse cost, not log I/O. Restored at the end to avoid leaking global
	// state into other tests in the package.
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer slog.SetDefault(prev)

	ctx := context.Background()

	b.Run("empty", func(b *testing.B) {
		provider := tng.New(nil, nil, nil, nil)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := provider.ParsePDFText(ctx, benchTNGText); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("loaded", func(b *testing.B) {
		provider := tng.New(
			nil,
			nil,
			[]models.CategoryRule{
				{Keyword: "merchant", Group: "Shopping", Category: "Online"},
			},
			nil,
		)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := provider.ParsePDFText(ctx, benchTNGText); err != nil {
				b.Fatal(err)
			}
		}
	})
}
