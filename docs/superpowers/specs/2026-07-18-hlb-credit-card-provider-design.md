# HLB Credit Card Provider Design

## Overview

Add a new provider for Hong Leong Bank (HLB) Credit Card statements. Parses PDF statements using `pdftotext` extraction and outputs Actual Budget-compatible CSV.

## Provider Name

`hlb`

## Extraction Method

`pdftotext` (digital extraction, not OCR)

## Statement Format

HLB credit card statements have:

- **Statement date** in `DD MMM YYYY` format (e.g., `14 JUL 2026`)
- **Card number** in `XXXX XXXX XXXX XXXX` format
- **Transactions** with transaction date, posting date, description, amount, and optional `CR` credit marker
- **Section markers**: `NEW TRANSACTION / CHARGES`, `PAYMENT RECEIVED - THANK YOU`
- **Summary lines**: `PREVIOUS BALANCE FROM LAST STATEMENT`, `SUB TOTAL`, `TOTAL BALANCE`

### Transaction Line Format

```
  15 JUN          16 JUN      GRAB-EC            PETALING JAYAMYS                                                                     19.05
  02 JUL          03 JUL      Shopee MY Marketplace KualaLumpur MYS                                                                   45.90    CR
```

Pattern: `^\s*(\d{2} \w{3})\s+(\d{2} \w{3})\s+(.+?)\s{2,}([\d,.]+)\s*(CR)?$`

- Group 1: Transaction date (`DD MMM`)
- Group 2: Posting date (`DD MMM`)
- Group 3: Description
- Group 4: Amount
- Group 5: `CR` if credit, empty if debit

### Credit/Debit Detection

- `CR` suffix → positive amount (credit)
- No suffix → negative amount (debit)

## Parsing Logic

### Statement Date

Regex: `Tarikh Penyata\s+(\d{2} \w{3} \d{4})`

Falls back to `Statement Date\s+(\d{2} \w{3} \d{4})` for English headers.

### Card Number (Account Name)

Regex: `(\d{4}\s\d{4}\s\d{4}\s\d{4})`

Falls back to `HLB Credit Card` if not found.

### Skip Rules

Lines are skipped if they match:
- `PREVIOUS BALANCE FROM LAST STATEMENT`
- `NEW TRANSACTION / CHARGES`
- `SUB TOTAL`
- `TOTAL BALANCE`
- `PAYMENT RECEIVED - THANK YOU`
- Empty lines

### Date Formatting

Transaction dates use `DD MMM` format. Year is inferred from statement date:
- If transaction month > statement month → previous year
- Otherwise → statement year

Result: `YYYY-MM-DD`

## Files to Create

### `internal/providers/hlb/service.go`

Provider struct implementing `providers.Provider` and `providers.ConfigurableProvider`:

```go
type HLBProvider struct {
    engine         *rule.Engine
    accountMapping map[string]string
}
```

Methods:
- `Name() string` → `"hlb"`
- `ParseCSV(ctx, reader)` → returns error (PDF only)
- `ParsePDFText(ctx, text)` → parses transactions, returns `[]ActualBudgetReport`
- `ExtractionMethod()` → `pdfutil.ExtractionMethodPdftotext`
- `Reload(...)` → updates engine and account mapping

### `internal/providers/hlb/pdf.go`

Pure parsing functions:
- `parseTransactions(text string) ([]HLBReport, error)`
- `extractStatementDate(lines []string) string`
- `extractAccountName(text string) string`
- `parseTransactionLine(line string, stmtDate time.Time) (HLBReport, error)`

### `internal/providers/hlb/report.go`

```go
type HLBReport struct {
    TransDate   string
    PostDate    string
    Description string
    Amount      string
    IsCredit    bool
}
```

### Test Files

- `hlb_suite_test.go` — Ginkgo bootstrap
- `pdf_test.go` — parsing tests (statement date, card number, transactions, credits, debits, year boundary, filtering, categorization)
- `service_test.go` — provider tests (name, CSV error, account mapping, filtering)

## Files to Modify

### `cmd/app/main.go`

Add import and registration:

```go
import hlbprov "actual_helper/internal/providers/hlb"

// In bootstrap.Init:
"hlb": hlbprov.New,
```

## Config

Provider config supports:
- `exclude_keywords` — skip rows matching these descriptions
- `include_keywords` — whitelist mode
- `categories` — auto-categorization rules
- `account_mappings` — map card numbers to account names

Example `provider_config.json`:

```json
{
  "providers": {
    "hlb": {
      "exclude_keywords": [],
      "include_keywords": [],
      "categories": [
        {"keyword": "grab", "group": "Food & Dining", "category": "Delivery"},
        {"keyword": "shopee", "group": "Shopping", "category": "Online"}
      ],
      "account_mappings": {
        "1234 5678 9012 3456": "HLB Credit Card"
      }
    }
  }
}
```

## Testing

All tests use fake data — no real names, card numbers, or amounts.

Test cases:
- Parse debit transaction (no CR suffix)
- Parse credit transaction (CR suffix)
- Parse multiple transactions
- Skip summary lines
- Error on missing statement date
- Empty transactions with header present
- Year boundary (December on January statement)
- Filter by exclude keywords
- Match category from description
- Extract card number for account name
- Fallback to default account name
- Account mapping from config
- CSV returns error (PDF only)
