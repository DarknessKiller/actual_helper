# Ryt Bank Provider

**Date:** 2026-06-20
**Status:** Draft

## Overview

Add a new `ryt` provider that parses Ryt Bank PDF statements into Actual Budget-compatible CSV. Supports account name extraction from PDF headers with user-configurable mapping to Actual Budget account names.

## Config Changes

- `config.ProviderConfig` gains `AccountMappings map[string]string`
- `providers.ProviderFactory` signature gains `accountMappings map[string]string`
- `providers.ConfigurableProvider.Reload` signature gains `accountMappings map[string]string`
- `bootstrap.Init` passes through the new field
- TNG's `New` ignores the new parameter

## Provider Package: `internal/providers/ryt/`

### `report.go`

```go
type RytReport struct {
    Date        string
    Description string
    Amount      string
}
```

### `pdf.go`

- `extractAccountName(text string) string` — finds `"Account Transactions"` line, reads next non-empty line, splits on `" / "`, returns first part
- `parseBlocks(text string) ([]RytReport, error)` — splits text at date boundaries (`\d{1,2} \w+ \d{4}`), skipping rows matching "opening balance"
- `parseBlock(block string) (RytReport, error)` — parses one transaction block

### `service.go`

```go
type RytProvider struct {
    engine         *rule.Engine
    accountMapping map[string]string
}
```

- `New(excludeKeywords, includeKeywords, categories, accountMappings)`
- `Name()` → `"ryt"`
- `ParseCSV()` → returns error (PDF-only)
- `ParsePDFText()` → parse blocks → `toActualReports()`
- `toActualReports()` — shared mapper:
  1. Skip if "opening balance"
  2. `shouldSkip()` via rule engine
  3. Parse date: `"2 January 2006"`
  4. Parse amount (explicit sign)
  5. `matchCategory()` via rule engine
  6. Resolve account via mapping
  7. Build `ActualBudgetReport` (Payee always empty, Notes = description)

## Registration

One new line in `main.go` factory map.

## Example Config

```json
{
  "providers": {
    "ryt": {
      "account_mappings": {
        "Main Account": "Ryt Bank Checking"
      }
    }
  }
}
```

## Tests

- `pdf_test.go` — credit, debit, multiple blocks, opening balance skip, empty text
- `service_test.go` — Name(), ParseCSV error, account mapping resolution
