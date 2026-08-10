# Actual Helper

A Go web server that serves a browser-based converter for bank and fintech transaction files into [Actual Budget](https://actualbudget.org)-compatible CSV format. All conversion — CSV parsing, PDF extraction, OCR, and provider logic — runs in the browser via WebAssembly. Raw documents never reach the server.

---

## Features

- **Browser-local conversion** — CSV parsing, digital PDF extraction (PDF.js), scanned-PDF OCR (Tesseract.js), and provider conversion run entirely in the browser; documents never leave the device
- **Session-only results** — generated CSV is shown in the current session and is not offered as a download
- **Browser-side configuration** — load a JSON config file in the browser to set filters, categories, and account mappings; no server restart needed
- **Smart filtering** — `exclude_keywords` removes noise; `include_keywords` overrides exclusions to keep important rows
- **Auto-categorization** — case-insensitive keyword matching with global and per-provider category rules; first match wins
- **Account name mapping** — maps provider-specific account names to Actual Budget account names
- **Single output format** — clean CSV with standard Actual Budget columns

---

## Architecture

```
Browser UI → PDF.js / Tesseract.js → Go WASM providers → CSV
Server     → static frontend + /version endpoint only
```

The Go providers compile to WebAssembly (`actual-helper.wasm`) and run in the browser. The server is a pure static file server with a version endpoint — it never sees user documents.

### How conversion works

1. User selects a provider and drops a CSV or PDF file
2. For PDFs: PDF.js extracts text digitally; if that fails, Tesseract.js performs OCR
3. The extracted text (or raw CSV) is passed to the Go WASM provider
4. The provider parses transactions, applies filtering and categorization rules
5. `services.ToActualCSV` serializes the result to Actual Budget CSV format

---

## Supported Providers

### Supported input formats

Only TNG accepts CSV input. All providers accept PDF input; PDF extraction and OCR run in the browser. Password-protected PDFs can be opened with the password entered in the browser.

### TNG (Touch 'n Go eWallet)

| | |
|---|---|
| **Provider name** | `tng` |
| **File formats** | CSV, PDF |
| **Credit detection** | Transaction type-based: Reload, Receive from Wallet, DUITNOW_RECEIVEFROM, Refund, GO+ Daily Earnings, GO+ Cash In |
| **Debit detection** | All other transaction types |
| **Filtering** | Reference token detection skips lines with long reference IDs or known prefixes (TNGD, TNGQR, TNGOW) |

### Ryt Bank

| | |
|---|---|
| **Provider name** | `ryt` |
| **File formats** | PDF only |
| **Amount sign** | Explicit `+`/`-` prefix in the PDF text |
| **Date format** | `d Month YYYY` (e.g., `1 May 2026`) |
| **Special handling** | Opening balance rows are automatically skipped |

### HSBC Credit Card (Malaysia)

| | |
|---|---|
| **Provider name** | `hsbccredit` |
| **File formats** | PDF only |
| **Credit detection** | Amount suffixed with `CR` (e.g., `259.72CR` = payment received) |
| **Debit detection** | Plain positive amount (e.g., `8.50` = purchase) |
| **Date format** | `DD MMM` (year inferred from statement header; cross-year boundary handled) |
| **Special handling** | Summary rows (previous balance, credit limit, charges) are automatically skipped; OCR fallback for scanned/image-based statements |

### HLB (Hong Leong Bank)

| | |
|---|---|
| **Provider name** | `hlb` |
| **File formats** | PDF only (digital extraction via PDF.js) |
| **Statement types** | Credit card and debit account — auto-detected from PDF content |
| **Credit card detection** | Amount suffixed with `CR` (e.g., `45.90 CR`) |
| **Credit card debit detection** | Plain positive amount (e.g., `19.05`) |
| **Debit account detection** | Column position (layout format) or description match (`Deposit` = credit) |
| **Date format** | `DD MMM` (year inferred from statement date; cross-year boundary handled) |
| **Statement date format** | `DD MMM YYYY` (e.g., `14 JUL 2026`) |
| **Special handling** | Summary rows (previous balance, charges, subtotal) are automatically skipped; format auto-detected per statement |

### UOB Credit Card

| | |
|---|---|
| **Provider name** | `uobcredit` |
| **File formats** | PDF only (digital extraction via PDF.js) |
| **Credit detection** | Amount suffixed with `CR` (e.g., `326.76 CR`) |
| **Debit detection** | Plain positive amount (e.g., `89.00`) |
| **Date format** | `DD MMM` (year inferred from statement date; cross-year boundary handled) |
| **Statement date format** | `DD MMM YY` or `DD MMM YYYY` (e.g., `16 JUL 26`) |
| **Card detection** | Extracts card number from WORLD MASTERCARD / MASTERCARD / VISA areas |
| **Special handling** | Summary rows (sub-total, minimum payment due, credit limit, previous balance) are automatically skipped |

### GX Bank

| | |
|---|---|
| **Provider name** | `gxbank` |
| **File formats** | PDF only (digital extraction via PDF.js) |
| **Credit detection** | Amount prefixed with `+` (e.g., `+100.00`) |
| **Debit detection** | Amount prefixed with `-` (e.g., `-50.00`) |
| **Date format** | `d Month YYYY` (e.g., `1 May 2026`) |
| **Account types** | GX Savings Account, Pocket |
| **Account name mapping** | Configurable via `account_mappings` in provider config |
| **Special handling** | Opening balance rows are automatically skipped; account name extracted from PDF header |

### Adding a New Provider

1. Create a new package under `internal/providers/<name>/`
2. Implement the `Provider` interface:
   ```go
   type Provider interface {
       Name() string
       ParseCSV(ctx context.Context, r io.Reader) ([]models.ActualBudgetReport, error)
       ParsePDFText(ctx context.Context, text string) ([]models.ActualBudgetReport, error)
       ExtractionMethod() pdfutil.ExtractionMethod
   }
   ```
3. Add a factory function `New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule, accountMappings map[string]string) providers.Provider`
4. For credit card providers, use `internal/providers/cardutil` for card number extraction and account mapping:
   - `cardutil.ExtractAfterMarker(text, marker, fallback)` — extract card number after a text marker
   - `cardutil.ExtractNearCardType(text, cardTypes, fallback)` — extract card number near card type indicators
   - `cardutil.ApplyMapping(mapping, name)` — apply account name mapping from config
5. Register the provider factory in `cmd/wasm/main.go`

---

## Server

The server is a static file server. It serves the frontend SPA and exposes `GET /version`. There is no document-upload conversion endpoint — all conversion happens in the browser.

---

## Configuration

Provider rules are loaded in the browser. Use **Load JSON** in the UI to replace them for the current session, or **Download Sample** to get a template. The JSON file is read locally and never uploaded or persisted.

The config schema:

```json
{
  "global": {
    "exclude_keywords": ["Global Noise"],
    "include_keywords": [],
    "categories": [
      { "keyword": "shopee", "group": "Shopping", "category": "Online" }
    ]
  },
  "providers": {
    "tng": {
      "account_mappings": { "": "TNG" },
      "exclude_keywords": ["Quick Reload Payment", "Via eWallet to GO+"],
      "include_keywords": ["Daily Interest"],
      "categories": [
        { "keyword": "grab", "group": "Food & Dining", "category": "Delivery" }
      ]
    },
    "ryt": {
      "account_mappings": { "": "RYT" }
    },
    "hsbccredit": {
      "account_mappings": { "1234 5678 9012 3456": "HSBC Credit Card" },
      "exclude_keywords": ["Grab"],
      "include_keywords": [],
      "categories": [
        { "keyword": "shopee", "group": "Shopping", "category": "Online" }
      ]
    },
    "hlb": {
      "account_mappings": {
        "1234 5678 9012 3456": "HLB Credit Card",
        "HLB Debit Account": "HLB Savings"
      },
      "exclude_keywords": [],
      "include_keywords": [],
      "categories": [
        { "keyword": "grab", "group": "Food & Dining", "category": "Delivery" }
      ]
    },
    "uobcredit": {
      "account_mappings": { "1234 5678 9012 3456": "UOB Credit Card" },
      "exclude_keywords": [],
      "include_keywords": [],
      "categories": []
    },
    "gxbank": {
      "account_mappings": {
        "GX Savings Account": "GX Savings",
        "Secret stash Bonus Pocket": "GX Pocket"
      },
      "exclude_keywords": [],
      "include_keywords": [],
      "categories": [
        { "keyword": "interest earned", "group": "Income", "category": "Interest" },
        { "keyword": "duitnow", "group": "Transfer", "category": "DuitNow" },
        { "keyword": "pocket", "group": "Transfer", "category": "Pocket" }
      ]
    }
  }
}
```

### Merge Rules

| Field | Strategy |
|---|---|
| `exclude_keywords` | Union of global + provider-specific keywords |
| `include_keywords` | Union — if any match, the row is kept even if an exclude keyword also matches |
| `categories` | Global rules first, then provider-specific rules appended; first match wins |
| `account_mappings` | Provider-specific only |

If no config is loaded, conversion runs without filtering or categorization.

---

## Testing

All packages use [Ginkgo](https://onsi.github.io/ginkgo/) and [Gomega](https://onsi.github.io/gomega/). Test data uses fake/anonymized data.

```bash
ginkgo run ./...
```

Each package has its own test suite covering success paths, failure paths, and edge cases (empty inputs, missing fields, boundary conditions).

---

## Tech Stack

| | |
|---|---|
| **Language** | Go 1.26 |
| **Web framework** | [Fuego](https://github.com/go-fuego/fuego) |
| **Frontend** | Svelte 5, Vite, Tailwind CSS, DaisyUI |
| **PDF extraction** | PDF.js (digital), Tesseract.js (OCR fallback) |
| **Provider runtime** | Go compiled to WebAssembly |
| **Testing** | Ginkgo v2, Gomega, httptest |
