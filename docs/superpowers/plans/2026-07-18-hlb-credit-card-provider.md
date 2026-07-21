# HLB Credit Card Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a new provider for Hong Leong Bank (HLB) Credit Card statements that parses PDF statements using `pdftotext` extraction and outputs Actual Budget-compatible CSV.

**Architecture:** Follow the existing provider pattern: Handler → Service → Provider. The HLB provider implements the `Provider` and `ConfigurableProvider` interfaces, uses `pdftotext` for PDF extraction, and registers via the bootstrap factory pattern.

**Tech Stack:** Go, Ginkgo/Gomega, `pdftotext` (external binary), `rule.Engine` for filtering/categorization

## Global Constraints

- Provider name: `hlb`
- Extraction method: `pdftotext` (not OCR)
- Statement date format: `DD MMM YYYY` (e.g., `14 JUL 2026`)
- Transaction date format: `DD MMM` (year inferred from statement date)
- Credit marker: `CR` suffix
- All test data must use fake/anonymized data
- Follow existing HSBC Credit provider patterns

---

## File Structure

| File | Responsibility |
|------|----------------|
| `internal/providers/hlb/report.go` | `HLBReport` struct definition |
| `internal/providers/hlb/pdf.go` | Pure parsing functions (regex, transaction parsing) |
| `internal/providers/hlb/service.go` | Provider struct, implements `Provider` + `ConfigurableProvider` |
| `internal/providers/hlb/hlb_suite_test.go` | Ginkgo test suite bootstrap |
| `internal/providers/hlb/pdf_test.go` | PDF parsing unit tests |
| `internal/providers/hlb/service_test.go` | Provider behavior tests |
| `cmd/app/main.go` | Register provider in bootstrap |

---

### Task 1: Create report.go with HLBReport struct

**Files:**
- Create: `internal/providers/hlb/report.go`

**Interfaces:**
- Produces: `HLBReport` struct used by `pdf.go` and `service.go`

- [ ] **Step 1: Create report.go**

```go
package hlb

type HLBReport struct {
	TransDate   string
	PostDate    string
	Description string
	Amount      string
	IsCredit    bool
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/providers/hlb/`
Expected: No output (success)

- [ ] **Step 3: Commit**

```bash
git add internal/providers/hlb/report.go
git commit -m "feat(hlb): add HLBReport struct"
```

---

### Task 2: Create pdf.go with parsing functions

**Files:**
- Create: `internal/providers/hlb/pdf.go`

**Interfaces:**
- Consumes: `HLBReport` from Task 1
- Produces: `parseTransactions()`, `extractStatementDate()`, `extractAccountName()`, `parseTransactionLine()`

- [ ] **Step 1: Create pdf.go with all parsing functions**

```go
package hlb

import (
	"errors"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	statementDateRe    = regexp.MustCompile(`(?:Tarikh Penyata|Statement Date)\s+(\d{2} \w{3} \d{4})`)
	cardNumberRe       = regexp.MustCompile(`(\d{4}\s\d{4}\s\d{4}\s\d{4})`)
	transactionLineRe  = regexp.MustCompile(`^\s*(\d{2} \w{3})\s+(\d{2} \w{3})\s+(.+?)\s{2,}([\d,.]+)\s*(CR)?$`)
	monthNames         = map[string]time.Month{
		"JAN": time.January, "FEB": time.February, "MAR": time.March,
		"APR": time.April, "MAY": time.May, "JUN": time.June,
		"JUL": time.July, "AUG": time.August, "SEP": time.September,
		"OCT": time.October, "NOV": time.November, "DEC": time.December,
	}
	skipPatterns = []string{
		"PREVIOUS BALANCE FROM LAST STATEMENT",
		"NEW TRANSACTION / CHARGES",
		"SUB TOTAL",
		"TOTAL BALANCE",
		"PAYMENT RECEIVED - THANK YOU",
	}
)

func parseTransactions(text string) ([]HLBReport, error) {
	lines := strings.Split(text, "\n")

	stmtDateStr := extractStatementDate(lines)
	if stmtDateStr == "" {
		slog.Warn("statement date not found in HLB text",
			"text_preview", truncate(text, 400),
		)
		return nil, errors.New("statement date not found")
	}

	stmtDate, err := time.Parse("02 Jan 2006", stmtDateStr)
	if err != nil {
		slog.Warn("invalid statement date format", "raw", stmtDateStr)
		return nil, errors.New("invalid statement date")
	}

	var reports []HLBReport
	for _, line := range lines {
		if shouldSkipLine(line) {
			continue
		}

		report, err := parseTransactionLine(line, stmtDate)
		if err != nil {
			continue
		}
		reports = append(reports, report)
	}

	return reports, nil
}

func extractStatementDate(lines []string) string {
	for _, line := range lines {
		matches := statementDateRe.FindStringSubmatch(line)
		if matches != nil {
			return matches[1]
		}
	}
	return ""
}

func extractAccountName(text string) string {
	matches := cardNumberRe.FindString(text)
	if matches != "" {
		return matches
	}
	slog.Debug("card number not found in HLB text", "preview", truncate(text, 600))
	return "HLB Credit Card"
}

func shouldSkipLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "empty"
	}
	lower := strings.ToLower(trimmed)
	for _, pattern := range skipPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return pattern
		}
	}
	return ""
}

func parseTransactionLine(line string, stmtDate time.Time) (HLBReport, error) {
	matches := transactionLineRe.FindStringSubmatch(line)
	if matches == nil {
		return HLBReport{}, errors.New("no match")
	}

	transDateStr := matches[1]
	postDateStr := matches[2]
	description := strings.TrimSpace(matches[3])
	amountStr := matches[4]
	isCredit := matches[5] == "CR"

	transDate := formatDate(transDateStr, stmtDate)
	postDate := formatDate(postDateStr, stmtDate)

	return HLBReport{
		TransDate:   transDate,
		PostDate:    postDate,
		Description: description,
		Amount:      amountStr,
		IsCredit:    isCredit,
	}, nil
}

func formatDate(ddmmm string, stmtDate time.Time) string {
	parts := strings.SplitN(ddmmm, " ", 2)
	if len(parts) != 2 {
		return ddmmm
	}

	day, err := strconv.Atoi(parts[0])
	if err != nil {
		return ddmmm
	}

	monthNum, ok := monthNames[strings.ToUpper(parts[1])]
	if !ok {
		return ddmmm
	}

	stmtMonth := stmtDate.Month()
	year := stmtDate.Year()

	if monthNum > stmtMonth {
		year--
	}

	t := time.Date(year, monthNum, day, 0, 0, 0, 0, time.UTC)
	return t.Format("2006-01-02")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/providers/hlb/`
Expected: No output (success)

- [ ] **Step 3: Commit**

```bash
git add internal/providers/hlb/pdf.go
git commit -m "feat(hlb): add PDF parsing functions"
```

---

### Task 3: Create pdf_test.go with parsing tests

**Files:**
- Create: `internal/providers/hlb/hlb_suite_test.go`
- Create: `internal/providers/hlb/pdf_test.go`

**Interfaces:**
- Consumes: `parseTransactions()`, `extractStatementDate()`, `extractAccountName()` from Task 2

- [ ] **Step 1: Create Ginkgo suite bootstrap**

```go
package hlb_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHLBCredit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HLB Credit Suite")
}
```

- [ ] **Step 2: Create pdf_test.go with all parsing tests**

```go
package hlb_test

import (
	"context"

	hlbprov "actual_helper/internal/providers/hlb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParsePDFText", func() {
	var (
		provider = hlbprov.New(nil, nil, nil, nil)
		ctx      = context.Background()
	)

	It("parses a debit transaction", func() {
		text := `Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      GRAB-EC            PETALING JAYA                                                                     19.05`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("-19.05"))
		Expect(reports[0].Date).To(Equal("2026-06-15"))
		Expect(reports[0].Notes).To(Equal("GRAB-EC PETALING JAYA"))
	})

	It("parses a credit transaction with CR suffix", func() {
		text := `Tarikh Penyata                    14 JUL 2026
  02 JUL          03 JUL      Shopee MY Marketplace                                                                   45.90    CR`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("45.90"))
		Expect(reports[0].Date).To(Equal("2026-07-02"))
		Expect(reports[0].Notes).To(Equal("Shopee MY Marketplace"))
	})

	It("parses multiple transactions", func() {
		text := `Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      GRAB-EC            PETALING JAYA                                                                     19.05
  20 JUN          22 JUN      BHPETROL           CHERAS                                                                             57.24`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(2))
		Expect(reports[0].Amount).To(Equal("-19.05"))
		Expect(reports[1].Amount).To(Equal("-57.24"))
	})

	It("skips summary lines", func() {
		text := `Tarikh Penyata                    14 JUL 2026
PREVIOUS BALANCE FROM LAST STATEMENT                                                                  181.42
NEW TRANSACTION / CHARGES
  15 JUN          16 JUN      GRAB-EC            PETALING JAYA                                                                     19.05
SUB TOTAL                                                                                             456.24
TOTAL BALANCE                                                                                         456.24`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Notes).To(Equal("GRAB-EC PETALING JAYA"))
	})

	It("returns error for text without statement date", func() {
		_, err := provider.ParsePDFText(ctx, "random text")
		Expect(err).To(HaveOccurred())
	})

	It("returns empty for text with header but no transactions", func() {
		text := `Tarikh Penyata                    14 JUL 2026
NEW TRANSACTION / CHARGES`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(BeEmpty())
	})

	It("handles year boundary: December transaction on January statement", func() {
		text := `Tarikh Penyata                    14 JAN 2027
  25 DEC          26 DEC      Online Shopping                                                                     50.00
  02 JAN          03 JAN      Ride Service                                                                        15.00CR`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(2))
		Expect(reports[0].Date).To(Equal("2026-12-25"))
		Expect(reports[0].Amount).To(Equal("-50.00"))
		Expect(reports[1].Date).To(Equal("2027-01-02"))
		Expect(reports[1].Amount).To(Equal("15.00"))
	})

	It("extracts card number from text for account name", func() {
		text := `Credit Card Number    1234 5678 9012 3456
Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      GRAB-EC            PETALING JAYA                                                                     19.05`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Account).To(Equal("1234 5678 9012 3456"))
	})

	It("falls back to HLB Credit Card when no card number found", func() {
		text := `Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      GRAB-EC            PETALING JAYA                                                                     19.05`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Account).To(Equal("HLB Credit Card"))
	})

	It("filters description using exclude keywords", func() {
		provider := hlbprov.New([]string{"GRAB"}, nil, nil, nil)
		text := `Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      GRAB-EC            PETALING JAYA                                                                     19.05
  20 JUN          22 JUN      BHPETROL           CHERAS                                                                             57.24`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Notes).To(Equal("BHPETROL CHERAS"))
	})

	It("matches category from description", func() {
		categories := []models.CategoryRule{
			{Keyword: "GRAB", Group: "Food & Dining", Category: "Delivery"},
		}
		provider := hlbprov.New(nil, nil, categories, nil)
		text := `Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      GRAB-EC            PETALING JAYA                                                                     19.05`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].CategoryGroup).To(Equal("Food & Dining"))
		Expect(reports[0].Category).To(Equal("Delivery"))
	})

	It("skips payment received lines", func() {
		text := `Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      GRAB-EC            PETALING JAYA                                                                     19.05
PAYMENT RECEIVED - THANK YOU
  04 JUL          04 JUL      PAYMENT THANK YOU CR                                                                181.42CR`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(2))
		Expect(reports[0].Notes).To(Equal("GRAB-EC PETALING JAYA"))
		Expect(reports[1].Notes).To(Equal("PAYMENT THANK YOU CR"))
	})
})
```

- [ ] **Step 3: Run tests to verify they compile and run**

Run: `go test ./internal/providers/hlb/ -v`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add internal/providers/hlb/hlb_suite_test.go internal/providers/hlb/pdf_test.go
git commit -m "test(hlb): add PDF parsing unit tests"
```

---

### Task 4: Create service.go with provider implementation

**Files:**
- Create: `internal/providers/hlb/service.go`

**Interfaces:**
- Consumes: `parseTransactions()`, `extractAccountName()` from Task 2, `HLBReport` from Task 1
- Produces: `New()` factory, implements `providers.Provider` + `providers.ConfigurableProvider`

- [ ] **Step 1: Create service.go**

```go
package hlb

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"actual_helper/internal/models"
	"actual_helper/internal/pdfutil"
	"actual_helper/internal/providers"
	"actual_helper/internal/rule"
)

type HLBProvider struct {
	engine         *rule.Engine
	accountMapping map[string]string
}

func New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider {
	return &HLBProvider{
		engine:         rule.NewEngine(excludeKeywords, includeKeywords, categories),
		accountMapping: accountMappings,
	}
}

func (p *HLBProvider) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) {
	p.engine.Reload(excludeKeywords, includeKeywords, categories)
	p.accountMapping = accountMappings
}

func (p *HLBProvider) shouldSkip(description string) bool {
	return p.engine.ShouldSkip(description)
}

func (p *HLBProvider) matchCategory(description string) (string, string) {
	return p.engine.MatchCategory(description)
}

func (p *HLBProvider) Name() string {
	return "hlb"
}

func (p *HLBProvider) ParseCSV(ctx context.Context, r io.Reader) ([]models.ActualBudgetReport, error) {
	return nil, errors.New("not supported for hlb provider")
}

func (p *HLBProvider) ParsePDFText(ctx context.Context, text string) ([]models.ActualBudgetReport, error) {
	logger := slog.With("provider", "hlb", "format", "pdf")

	accountName := extractAccountName(text)
	reports, err := parseTransactions(text)
	if err != nil {
		return nil, err
	}

	logger.InfoContext(ctx, "pdf parsing started", "blocks", len(reports), "account", accountName)

	result := p.toActualReports(ctx, logger, reports, accountName)
	logger.InfoContext(ctx, "pdf parsing complete", "parsed_count", len(result))
	return result, nil
}

var whitespacePattern = strings.NewReplacer("  ", " ")

func (p *HLBProvider) toActualReports(ctx context.Context, logger *slog.Logger, reports []HLBReport, accountName string) []models.ActualBudgetReport {
	var result []models.ActualBudgetReport

	for _, report := range reports {
		if p.shouldSkip(report.Description) {
			logger.DebugContext(ctx, "row skipped: filtered description", "description", report.Description)
			continue
		}

		description := strings.TrimSpace(whitespacePattern.Replace(report.Description, " "))

		amountStr := strings.ReplaceAll(report.Amount, ",", "")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || amount == 0 {
			logger.DebugContext(ctx, "row skipped: invalid amount", "raw", report.Amount)
			continue
		}

		if !report.IsCredit {
			amount = -amount
		}

		categoryGroup, category := p.matchCategory(description)

		if p.accountMapping != nil {
			if mapped, ok := p.accountMapping[accountName]; ok {
				accountName = mapped
			}
		}

		result = append(result, models.ActualBudgetReport{
			Account:       accountName,
			Date:          report.PostDate,
			Payee:         "",
			Notes:         description,
			CategoryGroup: categoryGroup,
			Category:      category,
			Amount:        strconv.FormatFloat(amount, 'f', 2, 64),
		})
	}

	return result
}

func (p *HLBProvider) ExtractionMethod() pdfutil.ExtractionMethod {
	return pdfutil.ExtractionMethodPdftotext
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/providers/hlb/`
Expected: No output (success)

- [ ] **Step 3: Commit**

```bash
git add internal/providers/hlb/service.go
git commit -m "feat(hlb): add provider implementation"
```

---

### Task 5: Create service_test.go with provider tests

**Files:**
- Create: `internal/providers/hlb/service_test.go`

**Interfaces:**
- Consumes: `New()` from Task 4

- [ ] **Step 1: Create service_test.go**

```go
package hlb_test

import (
	"context"
	"strings"

	hlbprov "actual_helper/internal/providers/hlb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HLBProvider", func() {
	Describe("Name", func() {
		It("returns hlb", func() {
			provider := hlbprov.New(nil, nil, nil, nil)
			Expect(provider.Name()).To(Equal("hlb"))
		})
	})

	Describe("ParseCSV", func() {
		It("returns error because hlb only supports PDF", func() {
			provider := hlbprov.New(nil, nil, nil, nil)
			_, err := provider.ParseCSV(context.Background(), strings.NewReader("a,b,c\n1,2,3"))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ParsePDFText with account mapping", func() {
		It("maps account name from config using card number", func() {
			accountMappings := map[string]string{
				"1234 5678 9012 3456": "HLB Credit",
			}
			provider := hlbprov.New(nil, nil, nil, accountMappings)
			text := `Credit Card Number    1234 5678 9012 3456
Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      GRAB-EC            PETALING JAYA                                                                     19.05`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Account).To(Equal("HLB Credit"))
		})

		It("falls back to HLB Credit Card when no card number in PDF", func() {
			provider := hlbprov.New(nil, nil, nil, nil)
			text := `Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      GRAB-EC            PETALING JAYA                                                                     19.05`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Account).To(Equal("HLB Credit Card"))
		})
	})

	Describe("ParsePDFText with filtering", func() {
		It("skips rows matching exclude keywords", func() {
			provider := hlbprov.New([]string{"GRAB"}, nil, nil, nil)
			text := `Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      GRAB-EC            PETALING JAYA                                                                     19.05
  20 JUN          22 JUN      BHPETROL           CHERAS                                                                             57.24`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Notes).To(Equal("BHPETROL CHERAS"))
		})
	})
})
```

- [ ] **Step 2: Run all tests to verify everything passes**

Run: `go test ./internal/providers/hlb/ -v`
Expected: All tests PASS

- [ ] **Step 3: Commit**

```bash
git add internal/providers/hlb/service_test.go
git commit -m "test(hlb): add provider unit tests"
```

---

### Task 6: Register provider in main.go

**Files:**
- Modify: `cmd/app/main.go`

**Interfaces:**
- Consumes: `New()` factory from Task 4

- [ ] **Step 1: Add import and registration to main.go**

In `cmd/app/main.go`, add the import:

```go
import (
	// ... existing imports ...
	hlbprov "actual_helper/internal/providers/hlb"
)
```

In `bootstrap.Init` map, add:

```go
registry, loader, env := bootstrap.Init(map[string]bootstrap.ProviderFactory{
	"tng":        tngprov.New,
	"ryt":        rytprov.New,
	"hsbccredit": hsbccreditprov.New,
	"hlb":  hlbprov.New,
})
```

- [ ] **Step 2: Verify full build compiles**

Run: `go build ./...`
Expected: No output (success)

- [ ] **Step 3: Run all tests**

Run: `go test ./...`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/app/main.go
git commit -m "feat(hlb): register provider in main"
```

---

### Task 7: Final verification

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: All tests PASS

- [ ] **Step 2: Verify no compilation errors**

Run: `go vet ./...`
Expected: No output (success)

- [ ] **Step 3: Final commit if needed**

```bash
git status
# Only commit if there are unstaged changes
```
