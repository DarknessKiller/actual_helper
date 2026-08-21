# Actual Helper

A Go web server that converts bank and fintech transaction files into [Actual Budget](https://actualbudget.org)-compatible CSV format. Supports multiple financial providers and file formats (CSV, PDF) through a single REST API.

---

## Features

- **Multi-provider architecture** — each financial institution gets its own provider package; easy to extend
- **CSV & PDF support** — including password-protected PDFs via decryption
- **Hot-reload configuration** — upload filters, categories, and account mappings via `POST /config` without restarting the server; takes effect on the next request
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

### Config Lifecycle

Providers are statically compiled into the binary and always active; configuration is tuning only. Config is empty by default and lives in memory — nothing is read from disk at startup or per request.

- `GET /config` — download the sample config file (served from `PROVIDER_CONFIG_PATH`; `404` if unset/unreadable)
- `POST /config` — upload a config `{global, providers}`; validated and hot-applied to every provider on the next request
- `DELETE /config` — clear all tuning back to empty

Invalid uploaded config is rejected with `400` and the previously loaded config stays active.

---

## Supported Providers

### TNG (Touch 'n Go eWallet)

| | |
|---|---|
| **Provider name** | `tng` |
| **File formats** | PDF only |
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
| **File formats** | PDF only (image-based, OCR via tesseract + gosseract) |
| **Credit detection** | Amount suffixed with `CR` (e.g., `259.72CR` = payment received) |
| **Debit detection** | Plain positive amount (e.g., `8.50` = purchase) |
| **Date format** | `DD MMM` (year inferred from statement header; cross-year boundary handled) |
| **Special handling** | Summary rows (previous balance, credit limit, charges) are automatically skipped; OCR fallback for scanned/image-based statements |

### HLB (Hong Leong Bank)

| | |
|---|---|
| **Provider name** | `hlb` |
| **File formats** | PDF only (digital extraction via pdftotext) |
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
| **File formats** | PDF only (digital extraction via pdftotext) |
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
| **File formats** | PDF only (digital extraction via ledongthuc/pdf) |
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
3. Implement `ConfigurableProvider` if the provider supports config-driven filtering/categorization
4. For credit card providers, use `internal/providers/cardutil` for card number extraction and account mapping:
   - `cardutil.ExtractAfterMarker(text, marker, fallback)` — extract card number after a text marker
   - `cardutil.ExtractNearCardType(text, cardTypes, fallback)` — extract card number near card type indicators
   - `cardutil.ApplyMapping(mapping, name)` — apply account name mapping from config
5. Register the provider in `cmd/app/main.go`

A source-only reference implementation lives at `internal/providers/sample/` (implements the full contract, not registered by default); see `docs/providers.md` for build/load instructions.

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

### GET /config

Downloads the sample config file (served from `PROVIDER_CONFIG_PATH`). `404` when the path is unset or unreadable.

### POST /config

Uploads a config JSON `{global, providers}`. Validated and hot-applied to every provider on the next request. Invalid JSON or missing body → `400` (previous config stays active).

### DELETE /config

Unloads the config: clears all tuning back to empty.

---

## Configuration

Providers run with empty tuning until you load a config via `POST /config` (or `DELETE /config` to clear it). `PROVIDER_CONFIG_PATH` only tells `GET /config` where the sample file lives — it is never auto-applied.

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

Config is optional and empty by default. Invalid uploads are rejected with `400` and the previously loaded config stays active.

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
| **PDF extraction** | ledongthuc/pdf (digital), pdftotext (text extraction), tesseract + gosseract (OCR fallback) |
| **PDF decryption** | pdfcpu |
| **PDF rendering** | poppler-utils (pdftoppm) |
