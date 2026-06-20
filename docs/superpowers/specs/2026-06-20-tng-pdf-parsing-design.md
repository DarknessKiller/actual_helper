# TNG PDF Parsing Design

**Created:** 2026-06-20T21:00:00+08:00
**Updated:** 2026-06-20T21:00:00+08:00

## Overview

Add PDF parsing support to the TNG provider so it handles both CSV (from bentopdf) and raw encrypted PDF (direct from TNG email attachment). The approach extracts transaction blocks from extracted PDF text, reconstructs `TNGReport` structs, then feeds them through a shared mapping layer.

## Architecture

Introduce a shared `toActualReports()` method on `TNGProvider` to eliminate duplication between `ParseCSV` and `ParsePDFText`:

```
ParseCSV:     csv.Reader → []TNGReport ─┐
                                         ├── → toActualReports() → []ActualBudgetReport
ParsePDFText: parsePDFBlocks(text) → ────┘
```

## PDF Text Parsing Algorithm

### Block Splitting

1. Find `TNG WALLET TRANSACTION` marker in extracted text.
2. Regex `(?m)^(\d{1,2}/\d{1,2}/\d{4})\s+(Success|Failed)\s+` finds all transaction start positions.
3. Each block spans from one start to the next (or end of text).

### First Line Parsing

Extract `date` and `status` from regex capture groups. Everything after the status word is the remainder.

### Type / Reference Splitting

Scan remainder tokens left-to-right. A token starts the reference if it **starts with a digit** or is a **known ref prefix** (`TNGD`, `TNGQR`, `TNGOW`). All tokens before it form the type.

### Multi-line Type Detection

If no reference token is found on line 1 and the block has more lines:
- If next line is non-blank, doesn't start with a digit, and doesn't match the date pattern → it's a type continuation.
- Repeat until a reference delimiter or blank line is found.

Examples:
- `DUITNOW_RECEI` + `VEFROM` → type `DUITNOW_RECEIVEFROM`
- `Reload` → type `Reload` (no continuation needed)

### Reference Accumulation

All lines from the first reference token (or first ref-containing line) through subsequent non-blank lines. Concatenated without spaces.

### Description Extraction

All lines after the reference block has ended (first blank line after reference start), up to (but not including) the RM amount line. Whitespace collapsed and trimmed.

### Amount Extraction

Last line matching `RM(\d+[.,]\d{2})\s+RM(\d+[.,]\d{2})`. First group = amount, second group = wallet balance (unused).

### Credit/Debit Determination

The existing `isCredit()` function checks `TNGReport.TransactionType`. The extracted type from PDF parsing follows the same semantics: if type contains `Reload`, `Receive`, or `Refund` → credit (positive amount in output).

## Refactoring: Shared Mapping Layer

Extract the filtering/categorization/mapping loop (currently `ParseCSV` lines 57–111) into:

```go
func (provider *TNGProvider) toActualReports(
    ctx context.Context,
    logger *slog.Logger,
    reports []TNGReport,
) []models.ActualBudgetReport
```

Responsibilities:
- Call `ensureRulesLoaded()`
- Filter non-Success status rows (skip with debug log)
- Filter excluded descriptions (`Quick Reload Payment`, `Daily Interest`, `Via eWallet to GO+`)
- Parse date via existing `parseDate()`
- Normalize payee (whitespace collapse + trim)
- Parse amount via existing `parseAmount()`
- Determine sign via existing `isCredit()`
- Apply categories via existing `match()`
- Map to `ActualBudgetReport`

`ParseCSV` changes: read CSV → `[]TNGReport` → call `toActualReports()`. No behavior change.

`ParsePDFText` signature changes from returning an error to:

```go
func (provider *TNGProvider) ParsePDFText(
    ctx context.Context,
    text string,
) ([]models.ActualBudgetReport, error)
```

## Error Handling

| Scenario | Behavior |
|---|---|
| No `TNG WALLET TRANSACTION` marker found | Return error |
| Block with missing required fields | Log debug, skip block |
| Unparseable amount line | Log debug, skip block |
| Invalid date | Log debug, skip block |
| Malformed rules file | Log warning, continue without categories |

Same skip-and-continue pattern as the CSV path.

## Files Modified

- `internal/providers/tng/service.go` — add `parsePDFBlocks()`, add `toActualReports()`, simplify `ParseCSV`, implement `ParsePDFText`
- `internal/providers/tng/service_test.go` — replace existing `ParsePDFText` "not supported" test with real PDF parsing tests
- Optional: `internal/providers/tng/report.go` — no changes expected (TNGReport unchanged)

## Test Plan

All tests through the public `ParsePDFText` API:

### Success Paths

- Single Reload transaction (credit, positive amount in output)
- Single Payment transaction (debit, negative amount in output)
- `DUITNOW_RECEIVEFROM` split across lines (credit)
- Mixed multi-transaction block (correct sign per transaction)
- Transaction with filtered description (skipped)

### Failure Paths

- Empty text / no transaction marker → error
- Malformed first line → skip block
- Missing amount line → skip block

### Categories

- Rules file applies categorization to PDF-parsed transactions (same pattern as CSV tests)
- Missing env var continues without categories

### Edge Cases

- Date with single-digit day/month (`1/5/2026`)
- Date with double-digit day/month (`01/12/2026`)
- Description with extra whitespace (collapsed)
- RM amount with commas (`RM1,234.56`)

## Risks

- PDF text extraction quality varies by library. The `ledongthuc/pdf` extractor may produce different line breaks than bentopdf. If upstream PDF layout changes, parsing may break.
- Type/reference splitting heuristic could fail for an unseen transaction type. Mitigation: if no type can be extracted, the block is skipped (logged), and user falls back to CSV.
