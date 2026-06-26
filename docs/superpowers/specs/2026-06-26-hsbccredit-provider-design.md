# HSBCCredit Provider (HSBC Credit Card Malaysia)

**Created:** 2026-06-26T00:00:00+08:00
**Updated:** 2026-06-26T21:28:52+08:00
**Status:** Implemented

## Overview

Add a new `hsbccredit` provider that parses HSBC Malaysia Credit Card PDF statements into Actual Budget-compatible CSV. Statements are image-based (scanned) PDFs requiring OCR; the existing digital-only PDF pipeline (`ledongthuc/pdf`) returns empty text for these files.

## Problem

HSBC Malaysia sends credit card statements as image-based PDFs — the text is rendered as pixels, not as embedded text objects. The existing `pdfutil.ExtractText` relies on `ledongthuc/pdf` which only extracts digital text and returns empty output for scanned documents.

## Solution

### 1. Text Extraction Pipeline in `internal/pdfutil/extract.go`

Three-tier fallback pipeline, transparent to providers:

1. **`ledongthuc/pdf`** — digital text extraction (fast, for native PDFs)
2. **`pdftotext -layout`** (poppler-utils) — CLI-based extraction, more robust than the Go library for digital PDFs
3. **OCR** — for scanned/image-based PDFs:
   - `pdftoppm` (poppler-utils) — render each PDF page to PNG at 200 DPI
   - Split tall pages (>4000px) into overlapping strips (200px overlap) via ImageMagick `convert -crop`
   - `tesseract` CLI — fresh process per strip (no shared CGo state), `-l eng`

**Why `tesseract` CLI over `gosseract` CGo bindings:**
- `gosseract` v2.4.1 CGo bindings have internal buffer/state limits causing mid-page truncation
- Each CLI invocation is a fresh process with zero shared state
- Strip splitting prevents image height limits from truncating dense statements

### 2. Provider Package: `internal/providers/hsbccredit/`

#### `report.go`

```go
type HSBCReport struct {
    PostDate    string
    TransDate   string
    Description string
    Amount      string
    IsCredit    bool
}
```

#### `pdf.go`

- `parseTransactions(text)` — parses OCR'd statement text:
  1. Extract `Statement Date DD MMM YYYY` from header for year inference
  2. Extract **card number** (`\d{4}[\s-]*\d{4}[\s-]*\d{4}[\s-]*\d{4}`) — used as account name; falls back to `"HSBC Credit Card"`
  3. Find transaction section via `Post date | Transaction date | Transaction details | Amount (RM)` marker
  4. Parse each line with regex: `^(\d{2} \w{3})\s+(\d{2} \w{3})\s+(.+)\s+([\d,]+\.?\d*)(CR)?\s*$`
  5. Clean OCR artifacts: strip `|`, `[`, `]`
  6. Skip summary rows: "Your Previous Statement Balance", "Credit limit used last statement", etc.

**Year inference:**
- Statement date `04 JUN 2026` → year = 2026
- Transaction month > statement month → year = statement year - 1 (e.g., December on January statement)
- Otherwise → year = statement year

**Amount parsing:**
- Suffix `CR` → credit (payment received by bank, positive in Actual Budget)
- No suffix → debit (purchase, negative in Actual Budget)

#### `service.go`

```go
type HSBCProvider struct {
    engine         *rule.Engine
    accountMapping map[string]string
}
```

- `New()` — constructor with keywords, categories, account mappings
- `Name()` → `"hsbccredit"`
- `ParseCSV()` → returns error (PDF-only)
- `ParsePDFText()` → parse transactions → `toActualReports()`
- `toActualReports()` — shared mapper:
  1. `shouldSkip()` via rule engine
  2. Clean whitespace in description
  3. Parse amount, negate if debit (not credit)
  4. `matchCategory()` via rule engine
  5. Resolve account via mapping
  6. Build `ActualBudgetReport` (Payee empty, Notes = description)

## Config Changes

- `provider_config.json` — add `"hsbccredit"` section with `account_mappings`
- `provider_config.example.json` — same with example categories

**Account mapping key** is the **card number** (space-separated, e.g., `"1234 5678 9012 3456"`), not `"HSBC Credit Card"`.
Config entries are user-specific — each user maps their own card number to their Actual Budget account name.

**Card number extraction:** `extractCardNumber` normalizes dashes → spaces then matches `[\s-]*` (handles OCR multi-space artifacts like `5400  4190  1362  4330`). If extraction fails, `debugCardNumberLines` logs the surrounding text lines for diagnosis (run with `LOG_LEVEL=debug`).

## Registration

One new line in `main.go` factory map:

```go
"hsbccredit": hsbccreditprov.New,
```

## Dependencies

| Dependency | Purpose |
|---|---|
| `tesseract-ocr` (system) | OCR engine via CLI (`tesseract page.png stdout -l eng`) |
| `poppler-utils` (system) | `pdftoppm` for PDF-to-image rendering, `pdftotext` for digital text |
| ImageMagick (system) | `identify` for image dimensions, `convert -crop` for strip splitting |

No Go CGo bindings — all CLI-based (no CGo build issues, no shared process state).

Dockerfile changes:
- Builder: removed `gcc musl-dev tesseract-ocr-dev`, `CGO_ENABLED=1` → `CGO_ENABLED=0` (no CGo = no C toolchain)
- Runtime: `alpine:3.21` with `tesseract-ocr` + `poppler-utils` + `imagemagick`

## Tests

### `pdf_test.go`
- Credit transaction (CR suffix) → positive amount
- Debit transaction (purchase) → negative amount
- Multiple transactions
- Summary lines skipped
- Missing statement date → error
- Header with no transactions → empty result
- Year boundary (Dec on Jan statement) → correct year assignment
- Exclude keyword filtering
- Category matching
- Zero amount rows skipped
- Card number extraction → account name set to card number
- Fallback to `"HSBC Credit Card"` when no card number in PDF
- Card number with dashes → normalized to spaces

### `service_test.go`
- `Name()` returns `"hsbccredit"`
- `ParseCSV()` returns error
- Account mapping by card number → `"Current"`
- Fallback to `"HSBC Credit Card"` when no card number in PDF
- Exclude keyword filtering
