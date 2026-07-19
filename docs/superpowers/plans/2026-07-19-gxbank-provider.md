# GX Bank Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans task-by-task.

**Goal:** Add GX Bank PDF statement provider (savings + pocket accounts).

**Architecture:** New `internal/providers/gxbank/` following TNG/HLB pattern. ledongthuc/pdf digital extraction, state machine parser, +/- prefix credit/debit.

**Tech Stack:** Go, Ginkgo/Gomega, ledongthuc/pdf

## Global Constraints

- PDF only, no CSV
- Extraction: `ledongthuc/pdf` (digital)
- Credit/debit via `+`/`-` prefix
- Account name from PDF header, config mapping override
- All test data fake/anonymized
- Follow existing provider pattern exactly

---

## File Structure

```
internal/providers/gxbank/
├── report.go              # GXReport struct
├── pdf.go                 # PDF text parsing
├── pdf_test.go            # PDF parsing tests
├── service.go             # Provider impl
├── service_test.go        # Service tests
└── gxbank_suite_test.go   # Ginkgo suite

cmd/app/main.go            # Register gxbank
```

---

### Task 1: Scaffold provider package + report struct

**Files:** Create `gxbank_suite_test.go`, `report.go`

- [ ] Create Ginkgo suite (`package gxbank_test`, `TestGxbank`)
- [ ] Create `report.go` with `GXReport{Date, Description, Amount string; IsCredit bool}`
- [ ] `go test ./internal/providers/gxbank/...` — compiles
- [ ] Commit: `feat(gxbank): scaffold provider package and report struct`

---

### Task 2: PDF parsing — marker + account extraction

**Files:** Create `pdf.go`, `pdf_test.go`

**Produces:** `ParsePDFBlocks(text) ([]GXReport, error)`, `ExtractAccountName(text) string`, `ExtractStatementYear(text) string`

- [ ] Test: error when no `"Closing balance (RM)"` marker
- [ ] Implement: `strings.Index` marker detection, return error if missing
- [ ] Test: extract account name from `"Statements of Accounts"` line after header
- [ ] Implement: scan lines for marker, return next line as name, check `"Account number"` line, fallback `"GX Bank"`
- [ ] Test: extract year from month header (e.g. "May 2026" → "2026")
- [ ] Implement: regex `^(?:January|...|December)\s+(\d{4})$`
- [ ] `go test ./internal/providers/gxbank/... -v` — PASS
- [ ] Commit: `feat(gxbank): add PDF marker detection and account extraction`

---

### Task 3: PDF parsing — transaction block splitting

**Files:** Modify `pdf.go`, `pdf_test.go`

**Consumes:** `ParsePDFBlocks`, `ExtractStatementYear` from Task 2

- [ ] Test: parse single interest earned (`+0.55`, credit)
- [ ] Implement: state machine parsing — date lines trigger new transaction, description from non-amount/non-date/non-time lines joined, amount from first `[+-][\d,]+\.\d{2}` match
- [ ] Test: multi-line description joined (`"Pocket\nWithdraw from Pocket"`)
- [ ] Test: amount with commas (`"+10,097.90"`)
- [ ] Test: multiple transactions, opening balance skipped
- [ ] Skip: time lines (`12:00 AM`), empty lines, `"Opening balance"` descriptions
- [ ] `go test ./internal/providers/gxbank/... -v` — PASS
- [ ] Commit: `feat(gxbank): implement PDF transaction block parsing`

---

### Task 4: Provider service implementation

**Files:** Create `service.go`, `service_test.go`

**Consumes:** All functions from Tasks 1-3

- [ ] Create `GXBankProvider` struct with `engine *rule.Engine`, `mu sync.RWMutex`, `accountMapping map[string]string`
- [ ] Implement `New()`, `Reload()`, `Name()` → `"gxbank"`, `ParseCSV()` → error, `ParsePDFText()`, `ExtractionMethod()` → `pdfutil.ExtractionMethodDigital`
- [ ] Implement `toActualReports()`: account mapping, skip opening balance, `shouldSkip()`, parse date (`"2 January 2006"` then `"2 Jan 2006"`), clean description, parse amount (strip prefix/commas), sign from `IsCredit`, `matchCategory()`
- [ ] Test: full flow PDF text → `ActualBudgetReport`
- [ ] Test: account mapping override
- [ ] Test: exclude_keywords filtering
- [ ] Test: CSV returns error
- [ ] Test: extraction method is `pdfutil.ExtractionMethodDigital`
- [ ] `go test ./internal/providers/gxbank/... -v` — PASS
- [ ] Commit: `feat(gxbank): implement provider service with filtering and mapping`

---

### Task 5: Register provider in main.go

**Files:** Modify `cmd/app/main.go`

- [ ] Add import `gxbankprov "actual_helper/internal/providers/gxbank"`
- [ ] Add `"gxbank": gxbankprov.New` to bootstrap.Init map
- [ ] `go build ./cmd/app/` — compiles
- [ ] `go test ./...` — all pass
- [ ] Commit: `feat(gxbank): register provider in main.go`

---

### Task 6: Final verification

- [ ] `go test ./... -v` — all pass, 0 failures
- [ ] `go test ./internal/providers/... -v` — no regressions
