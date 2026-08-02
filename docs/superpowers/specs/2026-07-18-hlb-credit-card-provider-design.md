# HLB Provider Design

## Overview

Add a unified provider for Hong Leong Bank (HLB) that parses both credit card and debit account PDF statements using `pdftotext` extraction. Format is auto-detected from PDF content.

## Provider Name

`hlb`

## Extraction Method

`pdftotext` (digital extraction, not OCR)

## Statement Formats

### Credit Card Statements

- **Statement date** in `DD MMM YYYY` format (e.g., `14 JUL 2026`)
- **Card number** in `XXXX XXXX XXXX XXXX` format
- **Transactions** with transaction date, posting date, description, amount, and optional `CR` credit marker
- **Section markers**: `NEW TRANSACTION / CHARGES`, `PAYMENT RECEIVED - THANK YOU`
- **Summary lines**: `PREVIOUS BALANCE FROM LAST STATEMENT`, `SUB TOTAL`, `TOTAL BALANCE`

Credit card transaction line format:
```
  15 JUN          16 JUN      GRAB-EC            PETALING JAYAMYS                                                                     19.05
  02 JUL          03 JUL      Shopee MY Marketplace KualaLumpur MYS                                                                   45.90    CR
```

Pattern: `^\s*(\d{2} \w{3})\s+(\d{2} \w{3})\s+(.+?)\s{2,}([\d,.]+)\s*(CR)?$`

- `CR` suffix → positive amount (credit)
- No suffix → negative amount (debit)

### Debit Account Statements

- **Account number** in `A/C No / No Akaun` or `No Akaun` header
- **Transactions** with date (`DD-MM-YYYY`), description, and amount
- Supports two PDF layouts: column-based and layout-based (with deposit/withdrawal/balance columns)
- **Summary lines**: `Total Withdrawals`, `Total Deposits`, `Closing Balance`, `Baki Akhir`
- **Opening balance**: `Balance from previous statement` is skipped

Debit account detection: explicit `Deposit`/`Withdrawal` column headers in the PDF.

## Format Auto-Detection

`DetectFormat(text)` examines PDF text content to route to the correct parser:
- `"credit"`: text contains `Credit Card Number`, `HLB Credit Card`, or `Tarikh Penyata`
- `"debit"`: text contains `A/C No`, `No Akaun`, or both `Deposit` and `Withdrawal`
- `"unknown"`: no recognized markers

## Parsing Logic

### Credit Card Parser

- Statement date regex: `Tarikh Penyata\s+(\d{2} \w{3} \d{4})`
- Card number via `cardutil.ExtractAfterMarker(text, "Credit Card Number", "HLB Credit Card")`
- Date formatting: `DD MMM` → `YYYY-MM-DD` (year inferred from statement date)

### Debit Account Parser

- Account number extracted inline from `A/C No` or `No Akaun` headers
- Two parsing strategies: column-based and layout-based (deposit/withdrawal/balance columns)
- Date format: `DD-MM-YYYY`

## Files

| File | Responsibility |
|------|----------------|
| `internal/providers/hlb/report.go` | `HLBReport` struct definition |
| `internal/providers/hlb/credit.go` | Credit card PDF parsing |
| `internal/providers/hlb/debit.go` | Debit account PDF parsing |
| `internal/providers/hlb/detect.go` | Format auto-detection (`DetectFormat`) |
| `internal/providers/hlb/service.go` | Provider struct, implements `Provider` + `ConfigurableProvider` |
| `internal/providers/hlb/hlbcredit_suite_test.go` | Ginkgo test suite bootstrap |
| `internal/providers/hlb/pdf_test.go` | Credit card PDF parsing tests |
| `internal/providers/hlb/debit_test.go` | Debit account PDF parsing tests |
| `internal/providers/hlb/detect_test.go` | Format detection tests |
| `internal/providers/hlb/service_test.go` | Provider behavior tests |

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

## Config

Provider config supports:
- `exclude_keywords` — skip rows matching these descriptions
- `include_keywords` — whitelist mode
- `categories` — auto-categorization rules
- `account_mappings` — map card/account numbers to account names

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
        "1234 5678 9012 3456": "HLB Credit Card",
        "12345678901": "HLB Savings"
      }
    }
  }
}
```

## Testing

All tests use fake data — no real names, card numbers, or amounts.

Test cases:
- Credit: parse debit/credit transactions, multiple transactions, skip summary lines, year boundary, filtering, categorization, card number extraction, account mapping
- Debit: parse transactions, layout format, amounts with commas, opening balance skip, multiple transactions, account extraction
- Detection: credit/debit/unknown format detection
- Provider: name, CSV error, account mapping for credit and debit, auto-detect routing, filtering
