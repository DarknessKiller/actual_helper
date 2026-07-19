# GX Bank Provider Design

## Overview

New provider for GX Bank PDF statements. Two account types:
- **GX Savings Account** — has account number, mixed transactions
- **Pocket** — no account number, typically interest only

## Scope

- PDF only (no CSV)
- Extraction: `ledongthuc/pdf` (digital)
- Credit/debit via `+`/`-` prefix
- Account name from PDF header, config mapping override

## File Structure

```
internal/providers/gxbank/
├── report.go              # GXReport struct
├── pdf.go                 # PDF text parsing
├── pdf_test.go            # PDF parsing tests
├── service.go             # Provider implementation
├── service_test.go        # Service tests
└── gxbank_suite_test.go   # Ginkgo suite
```

## Data Model

### GXReport (provider-internal)

```go
type GXReport struct {
    Date        string // raw date from PDF, e.g. "1 Jun 2026"
    Description string // transaction description (may be multi-line, joined)
    Amount      string // raw amount with sign, e.g. "+0.55" or "-10,000.00"
    IsCredit    bool   // true if + prefix, false if - prefix
}
```

### Output

Maps to `models.ActualBudgetReport` via shared `toActualReports()` pattern.

## PDF Parsing Strategy

### 1. Locate Transaction Table

Use `strings.Index` to find marker `"Closing balance (RM)"`. Skip header before marker.

### 2. Extract Account Name

- Find `"Statements of Accounts"` line
- Line after = account name (e.g. "GX Savings Account", "Secret stash Bonus Pocket")
- If `"Account number"` line exists, append masked number
- Fallback: `"GX Bank"`

### 3. Parse Transaction Blocks

State machine driven by date line regex: `(?i)^\d{1,2}\s+(?:Jan(?:uary)?|...|Dec(?:ember)?)\b(?:\s+\d{4})?$`

State transitions: idle → after date → after time → in description → after amount → idle.

For each transaction:
1. **Date:** first line matching date regex; two-field dates (no year) get year appended from statement header
2. **Description:** non-amount, non-date, non-time lines joined, trimmed
3. **Amount:** first line matching `[+-][\d,]+\.\d{2}` — prefix = credit/debit
4. **Skip** "Opening balance" (case-insensitive) blocks
5. **Skip** blocks with no valid amount

### 4. Credit/Debit Logic

- `+` → credit (positive)
- `-` → debit (negative)
- Strip prefix before numeric parse

## Service Layer

Follows existing provider pattern:

```go
func New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider
func (p *GXBankProvider) Reload(...)
func (p *GXBankProvider) Name() string                    // "gxbank"
func (p *GXBankProvider) ParseCSV(...) error              // "not supported"
func (p *GXBankProvider) ParsePDFText(ctx, text) ([]ActualBudgetReport, error)
func (p *GXBankProvider) ExtractionMethod() ExtractionMethod  // pdfutil.ExtractionMethodDigital
```

### toActualReports Flow

1. Apply account mapping (config override)
2. For each GXReport:
   - Skip "Opening balance"
   - `shouldSkip()` via rule.Engine
   - Parse date: `time.Parse("2 January 2006", raw)`, fallback `time.Parse("2 Jan 2006", raw)`
   - Clean description: collapse whitespace
   - Parse amount: strip prefix, strip commas, `strconv.ParseFloat`
   - Sign: positive if `IsCredit`, negative otherwise
   - `matchCategory()` via rule.Engine
   - Build `ActualBudgetReport`

## Registration

Add to `cmd/app/main.go`:

```go
gxbankprov "actual_helper/internal/providers/gxbank"

// In bootstrap.Init map:
"gxbank": gxbankprov.New,
```

## Provider Config

```json
{
  "providers": {
    "gxbank": {
      "exclude_keywords": [],
      "include_keywords": [],
      "categories": [],
      "account_mappings": {}
    }
  }
}
```

## Testing

### Data Privacy

All test data fake/anonymized:
- Account numbers masked (e.g. `8888-XX-XX-5`)
- No real names, reference IDs, or PII
- Synthetic transaction amounts

### Test Cases

#### PDF Parsing (`pdf_test.go`)

1. Interest-only (Pocket) — daily interest
2. Savings — mixed transactions
3. Credit — `+` prefix
4. Debit — `-` prefix
5. Multi-line description — joined correctly
6. Opening balance — skipped
7. Account name — from header
8. Account number — from "Account number" line
9. No transactions — error
10. Commas in amount — "+10,097.90"

#### Service (`service_test.go`)

1. Happy path — full flow
2. Account mapping — config override
3. Filtering — exclude_keywords
4. Categorization — category rules

## Edge Cases

- Multi-page: ledongthuc/pdf digital extraction, state machine parser handles
- Empty transactions: returns error
- Bilingual headers: English marker detection
- Page boundary summaries: filtered

## Dependencies

No new deps. Uses existing:
- `actual_helper/internal/rule`
- `actual_helper/internal/pdfutil`
- `actual_helper/internal/models`
- Standard library only
