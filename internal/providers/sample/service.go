// Package sample is a source-only demonstration of the provider contracts.
//
// It implements providers.Provider, providers.ConfigurableProvider and
// io.Closer against the same seam the real providers (tng, ryt, ...) use, so a
// custom build can drop it in without touching the service or handler layers.
//
// It is intentionally NOT registered in cmd/app/main.go: add it to the bootstrap
// factory map to activate it (see docs/providers.md).
//
// Parsing is deterministic and synthetic. There is no network access, no
// credentials and no real personal data — it echoes a couple of fixed
// ActualBudgetReport rows so the end-to-end server-side conversion flow can be
// exercised against a known shape.
package sample

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"sync"

	"actual_helper/internal/models"
	"actual_helper/internal/pdfutil"
	"actual_helper/internal/providers"
	"actual_helper/internal/rule"
)

const providerName = "sample"

const defaultDescription = "Sample Merchant"

// SampleProvider demonstrates the providers.Provider +
// providers.ConfigurableProvider + io.Closer contracts.
//
// It mirrors the real providers' shared-mapper seam (toActualReports): both
// ParseCSV and ParsePDFText funnel through sampleRows, which applies
// exclude/include keyword filtering and category matching via the rule engine
// so the hot-reload (Reload) path is exercised end to end.
type SampleProvider struct {
	engine  *rule.Engine
	mu      sync.RWMutex
	account map[string]string
}

// New is the bootstrap.ProviderFactory for the sample provider.
func New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider {
	return &SampleProvider{
		engine:  rule.NewEngine(excludeKeywords, includeKeywords, categories),
		account: accountMappings,
	}
}

func (p *SampleProvider) Name() string { return providerName }

// Reload satisfies providers.ConfigurableProvider. It swaps the tuning rules
// the engine uses for filtering/categorization; the synthetic parse is
// unchanged (there is nothing provider-specific to reconfigure).
func (p *SampleProvider) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) {
	p.engine.Reload(excludeKeywords, includeKeywords, categories)
	p.mu.Lock()
	p.account = accountMappings
	p.mu.Unlock()
}

// Close satisfies io.Closer. The provider is stateless, so there is nothing to
// release.
func (p *SampleProvider) Close() error { return nil }

// sampleRows is the shared mapper for ParseCSV and ParsePDFText. It builds a
// small deterministic pair of reports (one debit, one credit) and applies the
// configured filtering and categorization, exactly like the real providers'
// toActualReports.
func (p *SampleProvider) sampleRows(description string) []models.ActualBudgetReport {
	if description == "" {
		description = defaultDescription
	}
	if p.engine.ShouldSkip(description) {
		return nil
	}

	categoryGroup, category := p.engine.MatchCategory(description)

	account := "Sample Account"
	p.mu.RLock()
	if p.account != nil {
		if mapped, ok := p.account[account]; ok {
			account = mapped
		}
	}
	p.mu.RUnlock()

	return []models.ActualBudgetReport{
		{
			Account:       account,
			Date:          "2026-01-01",
			Payee:         "",
			Notes:         description,
			CategoryGroup: categoryGroup,
			Category:      category,
			Amount:        "-10.00",
		},
		{
			Account:       account,
			Date:          "2026-01-02",
			Payee:         "",
			Notes:         description + " (credit)",
			CategoryGroup: categoryGroup,
			Category:      category,
			Amount:        "5.50",
		},
	}
}

// ParseCSV echoes the CSV header's first column as the synthetic description.
// An empty input still yields the default sample rows.
func (p *SampleProvider) ParseCSV(_ context.Context, r io.Reader) ([]models.ActualBudgetReport, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return p.sampleRows(""), nil
		}
		return nil, err
	}
	description := strings.TrimSpace(strings.Join(header, " "))
	return p.sampleRows(description), nil
}

// ParsePDFText uses the first non-empty line of the extracted text as the
// synthetic description. An empty input still yields the default sample rows.
func (p *SampleProvider) ParsePDFText(_ context.Context, text string) ([]models.ActualBudgetReport, error) {
	description := ""
	for _, line := range strings.Split(text, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			description = trimmed
			break
		}
	}
	return p.sampleRows(description), nil
}

// ExtractionMethod returns a non-OCR method so the service never routes the
// sample provider through the gosseract/CGO OCR path.
func (p *SampleProvider) ExtractionMethod() pdfutil.ExtractionMethod {
	return pdfutil.ExtractionMethodDigital
}
