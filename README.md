# Actual Helper

A Go web server that converts bank and fintech transaction files into [Actual Budget](https://actualbudget.org)-compatible CSV format. Supports multiple financial providers and file formats (CSV, PDF) through a single REST API.

---

## Features

- **Multi-provider architecture** — each financial institution gets its own provider package; easy to extend
- **CSV & PDF support** — including password-protected PDFs via decryption
- **Hot-reload configuration** — update filters, categories, and account mappings without restarting the server; takes effect on the next request
- **Smart filtering** — `exclude_keywords` removes noise; `include_keywords` overrides exclusions to keep important rows
- **Auto-categorization** — case-insensitive keyword matching with global and per-provider category rules; first match wins
- **Account name mapping** — maps provider-specific account names to Actual Budget account names
- **Single output format** — clean CSV with standard Actual Budget columns

---

## Architecture

```
Handler → Service → Provider
```

The project follows strict three-layer separation:

- **Handler** — parses HTTP requests, validates input, calls the service, returns responses. No business logic.
- **Service** — orchestrates conversion: looks up providers, reloads config, routes to CSV or PDF parsing, serializes output.
- **Provider** — parses provider-specific file formats and maps fields to the shared output model.

### Hot-Reload Design

Configuration is checked on every request by comparing the config file's mtime. No background goroutines or server restarts needed — changes apply instantly to the next request. Missing or invalid config logs a warning and returns empty rules; the server never crashes due to config issues.

---

## Supported Providers

### TNG (Touch 'n Go eWallet)

| | |
|---|---|
| **Provider name** | `tng` |
| **File formats** | CSV, PDF |
| **Credit detection** | Transaction type-based: Reload, Receive from Wallet, DUITNOW_RECEIVEFROM, Refund, GO+ Daily Earnings, GO+ Cash In |
| **Debit detection** | All other transaction types |
| **Filtering** | Reference token detection skips lines with long reference IDs or known prefixes (TNGD, TNGQR, TNGOW) |

CSV columns expected: `Date`, `Status`, `Transaction Type`, `Reference ID`, `Description`, `Details`, `Amount`.

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
| **File formats** | PDF only (image-based, OCR via tesseract + gosseract) |
| **Credit detection** | Amount suffixed with `CR` (e.g., `259.72CR` = payment received) |
| **Debit detection** | Plain positive amount (e.g., `8.50` = purchase) |
| **Date format** | `DD MMM` (year inferred from statement header; cross-year boundary handled) |
| **Special handling** | Summary rows (previous balance, credit limit, charges) are automatically skipped; OCR fallback for scanned/image-based statements |

### Adding a New Provider

1. Create a new package under `internal/providers/<name>/`
2. Implement the `Provider` interface:
   ```go
   type Provider interface {
       Name() string
       ParseCSV(ctx context.Context, r io.Reader) ([]models.ActualBudgetReport, error)
       ParsePDFText(ctx context.Context, text string) ([]models.ActualBudgetReport, error)
   }
   ```
3. Implement `ConfigurableProvider` if the provider supports config-driven filtering/categorization
4. Register the provider in `internal/bootstrap/bootstrap.go`

---

## API Reference

### POST /convert/{provider}

Converts a transaction file from the specified provider into Actual Budget CSV format.

**Request:** Multipart form data

| Field | Type | Required | Description |
|---|---|---|---|
| `file` | file | yes | The transaction file (CSV or PDF) |
| `password` | string | no | Password for encrypted PDF files |

**Response:** `200 OK` with `Content-Type: text/csv` and `Content-Disposition: attachment`

**Errors:**

| Status | Body |
|---|---|
| `400` | Missing file in request |
| `500` | Unknown provider or processing error |

---

## Configuration

Set the `PROVIDER_CONFIG_PATH` environment variable to point to a JSON configuration file.

### Schema

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
      "account_mappings": { "": "Current" },
      "exclude_keywords": ["Quick Reload Payment", "Via eWallet to GO+"],
      "include_keywords": ["Daily Interest"],
      "categories": [
        { "keyword": "grab", "group": "Food & Dining", "category": "Delivery" }
      ]
    },
    "ryt": {
      "account_mappings": { "Savings Account": "Current" }
    },
    "hsbccredit": {
      "account_mappings": { "xxxx xxxx xxxx xxxx": "HSBC XXXX" },
      "exclude_keywords": ["Grab"],
      "include_keywords": [],
      "categories": [
        { "keyword": "shopee", "group": "Shopping", "category": "Online" }
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

Config file is optional. If missing, invalid, or unset, the server logs a warning and runs without filtering or categorization.

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
| **Testing** | Ginkgo v2, Gomega, httptest |
| **PDF extraction** | ledongthuc/pdf (digital), tesseract + gosseract (OCR fallback) |
| **PDF decryption** | pdfcpu |
| **PDF rendering** | poppler-utils (pdftoppm) |
