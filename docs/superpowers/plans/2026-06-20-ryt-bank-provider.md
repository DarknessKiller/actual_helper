# Ryt Bank Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) for syntax tracking.

**Goal:** Add a new `ryt` provider that parses Ryt Bank PDF statements into Actual Budget-compatible CSV.

**Architecture:** Follow the existing TNG provider pattern — `RytProvider` struct with embedded `*rule.Engine`, PDF-only, block-based parsing with a shared `toActualReports()` mapper. Account name is extracted from the PDF header and mapped via config.

**Tech Stack:** Go, Ginkgo/Gomega, ledongthuc/pdf (existing), pdfcpu (existing)

## Global Constraints

- Ryt is PDF-only — `ParseCSV` returns an error
- All transactions in the PDF are valid (no status filtering)
- Opening balance rows must be skipped
- Amount sign is explicit in the raw text (`+`/`-` prefix)
- Date format: `d Month YYYY` (e.g., "1 May 2026")
- Account name extracted from line after `"Account Transactions / Transaksi Akaun"`
- Account mapping via `account_mappings` in config JSON, per-provider
- Description lines concatenated with ` / ` separator into Notes
- Payee always empty

---
### Task 1: Add AccountMappings to Config and Interfaces

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/providers/provider.go`
- Modify: `internal/bootstrap/bootstrap.go`
- Modify: `internal/providers/tng/service.go`
- Modify: `internal/services/convert.go`

**Interfaces:**
- Consumes: existing `ProviderConfig`, `ConfigurableProvider`, `ProviderFactory`
- Produces: `ProviderConfig` with `AccountMappings` field, updated `ProviderFactory` and `ConfigurableProvider.Reload` signatures with `accountMappings map[string]string`

- [ ] **Step 1: Add AccountMappings to ProviderConfig**

In `internal/config/config.go:17`, add `AccountMappings` field:
```go
type ProviderConfig struct {
	ExcludeKeywords []string              `json:"exclude_keywords"`
	IncludeKeywords []string              `json:"include_keywords"`
	Categories      []models.CategoryRule `json:"categories"`
	AccountMappings map[string]string     `json:"account_mappings"`
}
```

- [ ] **Step 2: Update ProviderFactory and ConfigurableProvider signatures**

In `internal/providers/provider.go`, change `ConfigurableProvider`:
```go
type ConfigurableProvider interface {
	Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string)
}
```

- [ ] **Step 3: Update ProviderFactory signature in bootstrap**

In `internal/bootstrap/bootstrap.go:12`, change:
```go
type ProviderFactory func(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider
```

In `internal/bootstrap/bootstrap.go:25`, update the factory call:
```go
provider := factory(pc.ExcludeKeywords, pc.IncludeKeywords, pc.Categories, pc.AccountMappings)
```

- [ ] **Step 4: Update TNG's New and Reload**

In `internal/providers/tng/service.go`, update `New` and `Reload`:

```go
func New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider {
	return &TNGProvider{
		engine: rule.NewEngine(excludeKeywords, includeKeywords, categories),
	}
}

func (p *TNGProvider) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) {
	p.engine.Reload(excludeKeywords, includeKeywords, categories)
}
```

- [ ] **Step 5: Update reloadProvider in service**

In `internal/services/convert.go:74`, add account mappings to the reload call:
```go
func (service *ConvertService) reloadProvider(name string, provider providers.Provider) {
	if service.loader == nil {
		return
	}
	pc := service.loader.ProviderConfig(name)
	if cp, ok := provider.(providers.ConfigurableProvider); ok {
		cp.Reload(pc.ExcludeKeywords, pc.IncludeKeywords, pc.Categories, pc.AccountMappings)
	}
}
```

- [ ] **Step 6: Build and run tests**

Run: `go build ./...`
```
Expected: Successful compilation
```

Run: `go test ./...`
```
Expected: All existing tests pass (config, handlers, tng, rule, services)
```

- [ ] **Step 7: Commit**

```bash
git add internal/config/config.go internal/providers/provider.go internal/bootstrap/bootstrap.go internal/providers/tng/service.go internal/services/convert.go
git commit -m "feat(config): add account_mappings to ProviderConfig and interfaces"
```

---
### Task 2: Create Ryt Provider Package

**Files:**
- Create: `internal/providers/ryt/report.go`
- Create: `internal/providers/ryt/pdf.go`
- Create: `internal/providers/ryt/service.go`
- Create: `internal/providers/ryt/ryt_suite_test.go`
- Create: `internal/providers/ryt/pdf_test.go`
- Create: `internal/providers/ryt/service_test.go`

**Interfaces:**
- Consumes: `config.ProviderConfig` with `AccountMappings`, `rule.Engine`, models
- Produces: `ryt.RytProvider` implementing `providers.Provider` + `providers.ConfigurableProvider`

- [ ] **Step 1: Write the failing suite runner test**

Create `internal/providers/ryt/ryt_suite_test.go`:
```go
package ryt_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRytProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ryt Provider Suite")
}
```

- [ ] **Step 2: Write failing PDF parsing tests**

Create `internal/providers/ryt/pdf_test.go`:
```go
package ryt_test

import (
	"context"

	rytprov "actual-helper/internal/providers/ryt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParsePDFText", func() {
	var (
		provider = rytprov.New(nil, nil, nil, nil)
		ctx      = context.Background()
	)

	It("parses a credit transaction", func() {
		text := `Account Transactions / Transaksi Akaun
Main Account / Akaun Utama
Date
Tarikh
Description
Butiran
(MYR)
Amount
Amaun
(MYR)
Balance
Baki
1 May 2026 From Alice Tan
Transfer
Sent from Online
Ref. ID: F20260501ABCDEF1
+783.88 784.14`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("783.88"))
		Expect(reports[0].Date).To(Equal("2026-05-01"))
		Expect(reports[0].Payee).To(BeEmpty())
		Expect(reports[0].Notes).To(ContainSubstring("From Alice Tan"))
		Expect(reports[0].Notes).To(ContainSubstring("Transfer"))
		Expect(reports[0].Notes).To(ContainSubstring("Ref. ID: F20260501ABCDEF1"))
	})

	It("parses a debit transaction", func() {
		text := `Account Transactions / Transaksi Akaun
Main Account / Akaun Utama
Date
Tarikh
Description
Butiran
(MYR)
Amount
Amaun
(MYR)
Balance
Baki
1 May 2026 To Savings Goal
Money movement
Ref. ID: F20260501GHIJKL2
-784.14 0.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("-784.14"))
		Expect(reports[0].Date).To(Equal("2026-05-01"))
	})

	It("skips opening balance row", func() {
		text := `Account Transactions / Transaksi Akaun
Main Account / Akaun Utama
Date
Tarikh
Description
Butiran
(MYR)
Amount
Amaun
(MYR)
Balance
Baki
1 May 2026 Opening balance 0.26
1 May 2026 From Alice Tan
Transfer
Sent from Online
Ref. ID: F20260501ABCDEF1
+783.88 784.14`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
	})

	It("parses multiple transactions", func() {
		text := `Account Transactions / Transaksi Akaun
Main Account / Akaun Utama
Date
Tarikh
Description
Butiran
(MYR)
Amount
Amaun
(MYR)
Balance
Baki
1 May 2026 From Alice Tan
Transfer
Sent from Online
Ref. ID: F20260501ABCDEF1
+783.88 784.14
2 May 2026 From Daily Wallet
Money movement
Ref. ID: F20260502MNOPQR3
+10.00 10.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(2))
		Expect(reports[0].Amount).To(Equal("783.88"))
		Expect(reports[1].Amount).To(Equal("10.00"))
	})

	It("returns error for text without account transactions section", func() {
		_, err := provider.ParsePDFText(ctx, "random text")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no account transactions section found"))
	})

	It("returns empty for text with header but no transactions", func() {
		text := `Account Transactions / Transaksi Akaun
Main Account / Akaun Utama
Date
Tarikh
Description
Butiran
(MYR)
Amount
Amaun
(MYR)
Balance
Baki`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(BeEmpty())
	})
})
```

- [ ] **Step 3: Write failing service tests**

Create `internal/providers/ryt/service_test.go`:
```go
package ryt_test

import (
	"context"
	"strings"

	rytprov "actual-helper/internal/providers/ryt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RytProvider", func() {
	Describe("Name", func() {
		It("returns ryt", func() {
			provider := rytprov.New(nil, nil, nil, nil)
			Expect(provider.Name()).To(Equal("ryt"))
		})
	})

	Describe("ParseCSV", func() {
		It("returns error because ryt only supports PDF", func() {
			provider := rytprov.New(nil, nil, nil, nil)
			_, err := provider.ParseCSV(context.Background(), strings.NewReader("a,b,c\n1,2,3"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("CSV not supported"))
		})
	})

	Describe("ParsePDFText with account mapping", func() {
		It("maps account name from config", func() {
			accountMappings := map[string]string{
				"Main Account": "Ryt Bank Checking",
			}
			provider := rytprov.New(nil, nil, nil, accountMappings)
			text := `Account Transactions / Transaksi Akaun
Main Account / Akaun Utama
Date
Tarikh
Description
Butiran
(MYR)
Amount
Amaun
(MYR)
Balance
Baki
1 May 2026 From Alice Tan
Transfer
Ref. ID: F20260501ABCDEF1
+783.88 784.14`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Account).To(Equal("Ryt Bank Checking"))
		})

		It("falls back to extracted account name when no mapping exists", func() {
			provider := rytprov.New(nil, nil, nil, nil)
			text := `Account Transactions / Transaksi Akaun
Main Account / Akaun Utama
Date
Tarikh
Description
Butiran
(MYR)
Amount
Amaun
(MYR)
Balance
Baki
1 May 2026 From Alice Tan
Transfer
Ref. ID: F20260501ABCDEF1
+783.88 784.14`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Account).To(Equal("Main Account"))
		})
	})
})
```

- [ ] **Step 4: Run tests to verify they fail**

Run: `go test ./internal/providers/ryt/... -v`
Expected: Compilation errors (types don't exist yet)

- [ ] **Step 5: Create RytReport model**

Create `internal/providers/ryt/report.go`:
```go
package ryt

type RytReport struct {
	Date        string
	Description string
	Amount      string
}
```

- [ ] **Step 6: Create PDF parsing implementation**

Create `internal/providers/ryt/pdf.go`:
```go
package ryt

import (
	"errors"
	"log/slog"
	"regexp"
	"strings"
)

func extractAccountName(text string) string {
	const marker = "Account Transactions / Transaksi Akaun"
	idx := strings.Index(text, marker)
	if idx == -1 {
		return ""
	}
	after := text[idx+len(marker):]
	lines := strings.Split(after, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if parts := strings.SplitN(line, " / ", 2); len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	return ""
}

func parseBlocks(text string) ([]RytReport, error) {
	const marker = "Account Transactions / Transaksi Akaun"
	idx := strings.LastIndex(text, marker)
	if idx == -1 {
		slog.Debug("marker not found in text", "text_preview", truncate(text, 200))
		return nil, errors.New("no account transactions section found")
	}

	body := text[idx+len(marker):]
	slog.Debug("pdf body preview", "body", truncate(body, 500))

	// Find column header section: find "Balance\nBaki\n" then start after that
	balanceHeaderIdx := strings.Index(body, "Balance\nBaki")
	if balanceHeaderIdx == -1 {
		balanceHeaderIdx = strings.Index(body, "Baki")
		if balanceHeaderIdx == -1 {
			slog.Info("no column headers found in pdf body")
			return nil, nil
		}
	}

	dataStart := balanceHeaderIdx + len("Balance\nBaki")
	for dataStart < len(body) && (body[dataStart] == '\n' || body[dataStart] == '\r') {
		dataStart++
	}
	data := strings.TrimSpace(body[dataStart:])
	if data == "" {
		return nil, nil
	}

	dateRe := regexp.MustCompile(`(?m)^(\d{1,2} \w+ \d{4})\b`)
	splits := dateRe.FindAllStringSubmatchIndex(data, -1)
	if len(splits) == 0 {
		slog.Info("no transaction blocks found in pdf data",
			"data", truncate(data, 600),
		)
		return nil, nil
	}

	var reports []RytReport
	for i, split := range splits {
		blockStart := split[0]
		var blockEnd int
		if i+1 < len(splits) {
			blockEnd = splits[i+1][0]
		} else {
			blockEnd = len(data)
		}

		block := strings.TrimSpace(data[blockStart:blockEnd])
		if block == "" {
			continue
		}

		report, err := parseBlock(block)
		if err != nil {
			slog.Debug("pdf block skipped", "reason", err.Error())
			continue
		}
		reports = append(reports, report)
	}

	return reports, nil
}

func parseBlock(block string) (RytReport, error) {
	lines := strings.Split(block, "\n")
	if len(lines) < 2 {
		return RytReport{}, errors.New("block too short")
	}

	// First line: date + first part of description
	firstLine := strings.TrimSpace(lines[0])
	dateRe := regexp.MustCompile(`^(\d{1,2} \w+ \d{4})\s*(.*)`)
	matches := dateRe.FindStringSubmatch(firstLine)
	if matches == nil {
		return RytReport{}, errors.New("no date found in block")
	}

	date := matches[1]
	descParts := []string{strings.TrimSpace(matches[2])}

	// Middle lines: more description, last line: amount + balance
	lastLine := strings.TrimSpace(lines[len(lines)-1])

	// Parse amount from last line: matches +Amount or -Amount at start
	amountRe := regexp.MustCompile(`^([+-]\d+[.,]?\d*\.?\d*)`)
	amountMatch := amountRe.FindStringSubmatch(lastLine)
	if amountMatch == nil {
		return RytReport{}, errors.New("no amount found in block")
	}
	amount := amountMatch[1]

	// Collect description lines between first and last
	for i := 1; i < len(lines)-1; i++ {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			descParts = append(descParts, line)
		}
	}

	description := strings.Join(descParts, " / ")

	return RytReport{
		Date:        date,
		Description: description,
		Amount:      amount,
	}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
```

- [ ] **Step 7: Create service (RytProvider struct)**

Create `internal/providers/ryt/service.go`:
```go
package ryt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"actual-helper/internal/models"
	"actual-helper/internal/providers"
	"actual-helper/internal/rule"
)

type RytProvider struct {
	engine         *rule.Engine
	accountMapping map[string]string
}

func New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider {
	return &RytProvider{
		engine:         rule.NewEngine(excludeKeywords, includeKeywords, categories),
		accountMapping: accountMappings,
	}
}

func (p *RytProvider) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) {
	p.engine.Reload(excludeKeywords, includeKeywords, categories)
	p.accountMapping = accountMappings
}

func (p *RytProvider) shouldSkip(description string) bool {
	return p.engine.ShouldSkip(description)
}

func (p *RytProvider) matchCategory(description string) (string, string) {
	return p.engine.MatchCategory(description)
}

func (p *RytProvider) Name() string {
	return "ryt"
}

func (p *RytProvider) ParseCSV(ctx context.Context, r io.Reader) ([]models.ActualBudgetReport, error) {
	return nil, errors.New("CSV not supported for ryt provider")
}

func (p *RytProvider) ParsePDFText(ctx context.Context, text string) ([]models.ActualBudgetReport, error) {
	logger := slog.With("provider", "ryt", "format", "pdf")

	accountName := extractAccountName(text)

	reports, err := parseBlocks(text)
	if err != nil {
		return nil, err
	}

	logger.InfoContext(ctx, "pdf parsing started", "blocks", len(reports), "account", accountName)

	result := p.toActualReports(ctx, logger, reports, accountName)
	logger.InfoContext(ctx, "pdf parsing complete", "parsed_count", len(result))
	return result, nil
}

func (p *RytProvider) toActualReports(ctx context.Context, logger *slog.Logger, reports []RytReport, accountName string) []models.ActualBudgetReport {
	whitespacePattern := regexp.MustCompile(`\s+`)
	var result []models.ActualBudgetReport

	for _, report := range reports {
		if strings.Contains(strings.ToLower(report.Description), "opening balance") {
			logger.DebugContext(ctx, "row skipped: opening balance", "description", report.Description)
			continue
		}

		if p.shouldSkip(report.Description) {
			logger.DebugContext(ctx, "row skipped: filtered description", "description", report.Description)
			continue
		}

		parsedDate, err := time.Parse("2 January 2006", report.Date)
		if err != nil {
			logger.DebugContext(ctx, "row skipped: invalid date", "raw", report.Date)
			continue
		}

		description := strings.TrimSpace(whitespacePattern.ReplaceAllString(report.Description, " "))

		amountStr := strings.ReplaceAll(report.Amount, ",", "")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || amount == 0 {
			logger.DebugContext(ctx, "row skipped: invalid amount", "raw", report.Amount)
			continue
		}

		categoryGroup, category := p.matchCategory(description)

		account := accountName
		if p.accountMapping != nil {
			if mapped, ok := p.accountMapping[accountName]; ok {
				account = mapped
			}
		}

		result = append(result, models.ActualBudgetReport{
			Account:       account,
			Date:          parsedDate.Format("2006-01-02"),
			Payee:         "",
			Notes:         description,
			CategoryGroup: categoryGroup,
			Category:      category,
			Amount:        strconv.FormatFloat(amount, 'f', 2, 64),
		})
	}

	return result
}
```

- [ ] **Step 8: Run tests**

Run: `go test ./internal/providers/ryt/... -v`
Expected: All tests pass

- [ ] **Step 9: Run full test suite**

Run: `go test ./...`
Expected: All existing tests still pass, ryt tests pass

- [ ] **Step 10: Commit**

```bash
git add internal/providers/ryt/
git commit -m "feat(ryt): add ryt bank provider with PDF parsing and account mapping"
```

---
### Task 3: Wire Ryt Provider into Main

**Files:**
- Modify: `cmd/app/main.go`

**Interfaces:** N/A — wiring only

- [ ] **Step 1: Add ryt import and factory entry**

In `cmd/app/main.go`, add the import:
```go
import (
	"log"

	"actual-helper/internal/bootstrap"
	"actual-helper/internal/handlers"
	rytprov "actual-helper/internal/providers/ryt"
	tngprov "actual-helper/internal/providers/tng"
	"actual-helper/internal/services"

	"github.com/go-fuego/fuego"
)
```

Add `rytprov.New` to the factory map:
```go
registry, loader := bootstrap.Init(map[string]bootstrap.ProviderFactory{
	"tng": tngprov.New,
	"ryt": rytprov.New,
})
```

- [ ] **Step 2: Build and run all tests**

Run: `go build ./...`
Expected: Successful compilation

Run: `go test ./...`
Expected: All tests pass

- [ ] **Step 3: Commit**

```bash
git add cmd/app/main.go
git commit -m "feat(main): register ryt provider"
```
