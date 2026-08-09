# Actual Helper

A privacy-first, client-side tool that converts bank and fintech transaction files into [Actual Budget](https://actualbudget.org)-compatible CSV format. All processing runs entirely in your browser — no data ever leaves your device.

---

## Features

- **100% client-side** — PDF parsing, OCR, and conversion all run in the browser via WebAssembly
- **Privacy-first** — no server uploads, no network requests during conversion
- **Offline-capable** — works without internet after first load (PWA-ready)
- **Multi-provider** — supports 6 Malaysian banks/e-wallets
- **Smart filtering** — exclude noise, whitelist important transactions
- **Auto-categorization** — keyword-based category assignment
- **Password-protected PDFs** — decryption happens client-side

---

## Supported Providers

| Provider | Name | Format | Notes |
|---|---|---|---|
| TNG (Touch 'n Go) | `tng` | PDF | Digital extraction |
| RYT Bank | `ryt` | PDF | Digital extraction |
| HSBC Credit Card | `hsbccredit` | PDF | OCR (scanned statements) |
| HLB (Hong Leong Bank) | `hlb` | PDF | Credit + debit auto-detected |
| UOB Credit Card | `uobcredit` | PDF | Digital extraction |
| GX Bank | `gxbank` | PDF | Digital extraction |

---

## Quick Start

### Development

```bash
cd frontend
npm install
npm run dev
```

Open `http://localhost:5173` in your browser.

### Production (Docker)

```bash
docker compose up -d
```

Open `http://localhost:8080` in your browser.

### Production (Static)

```bash
cd frontend
npm run build
# Serve the dist/ directory with any static file server
npx serve dist
```

---

## Architecture

```
┌─────────────────────────────────────────────┐
│  Browser (Client-Side)                      │
│                                             │
│  PDF.js ──→ Text Extraction                 │
│  Tesseract.js ──→ OCR (scanned PDFs)        │
│                                             │
│  Provider Parsers ──→ ActualBudgetReport[]  │
│  Rule Engine ──→ Filter + Categorize        │
│  CSV Serializer ──→ Download                │
└─────────────────────────────────────────────┘
```

All processing happens client-side:

1. **PDF Extraction** — pdfjs-dist extracts text from digital PDFs
2. **OCR** — Tesseract.js handles scanned/image-based PDFs (HSBC Credit)
3. **Provider Parsing** — Each provider has a TypeScript parser that extracts transactions
4. **Rule Engine** — Filters noise and auto-categorizes transactions
5. **CSV Output** — Generates Actual Budget-compatible CSV for download

No data leaves your browser. No server-side processing.

---

## Tech Stack

| | |
|---|---|
| **Frontend** | Svelte 5, Vite 6, Tailwind 4, DaisyUI 5 |
| **PDF Parsing** | pdfjs-dist (WebAssembly) |
| **OCR** | Tesseract.js (WebAssembly) |
| **Testing** | Vitest |
| **Deployment** | Docker (nginx) or static hosting |

---

## Configuration

The app works without configuration. To customize filtering and categorization, create a `provider_config.json` and load it via the UI.

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
      "exclude_keywords": ["Quick Reload Payment"],
      "include_keywords": [],
      "categories": [
        { "keyword": "grab", "group": "Food & Dining", "category": "Delivery" }
      ],
      "account_mappings": { "": "TNG" }
    }
  }
}
```

### Merge Rules

| Field | Strategy |
|---|---|
| `exclude_keywords` | Union of global + provider-specific |
| `include_keywords` | Whitelist — if any match, row is kept |
| `categories` | First match wins |
| `account_mappings` | Provider-specific only |

---

## Testing

```bash
cd frontend
npm test
```

Tests use Vitest with inline fixtures matching the Go test data for parity verification.

---

## Project Structure

```
frontend/
├── src/
│   ├── lib/
│   │   ├── types.ts              # Shared TypeScript interfaces
│   │   ├── rules.ts              # Rule engine (filter + categorize)
│   │   ├── csv.ts                # CSV serializer
│   │   ├── dateutil.ts           # Date parsing utilities
│   │   ├── cardutil.ts           # Card number extraction
│   │   ├── pdf-worker.ts         # PDF.js wrapper
│   │   ├── ocr-worker.ts         # Tesseract.js wrapper
│   │   ├── api.ts                # Local conversion pipeline
│   │   ├── providers/
│   │   │   ├── tng.ts            # TNG parser
│   │   │   ├── ryt.ts            # RYT Bank parser
│   │   │   ├── hlb.ts            # HLB parser (credit + debit)
│   │   │   ├── hsbccredit.ts     # HSBC Credit parser
│   │   │   ├── uobcredit.ts      # UOB Credit parser
│   │   │   └── gxbank.ts         # GX Bank parser
│   │   ├── components/
│   │   │   ├── UploadForm.svelte # File upload + conversion
│   │   │   ├── ResultPanel.svelte
│   │   │   └── HistoryDashboard.svelte
│   │   └── stores/
│   │       └── history.ts        # Conversion history
│   └── main.ts
├── package.json
└── vite.config.js
```

---

## Adding a New Provider

1. Create `frontend/src/lib/providers/<name>.ts`
2. Export a parser function: `(text: string, config: ProviderConfig) => ActualBudgetReport[]`
3. Register in `api.ts`:
   ```typescript
   import { parseNewProvider } from './providers/<name>'
   const parsers = { ..., newprovider: parseNewProvider }
   ```
4. Add to `UploadForm.svelte` providers list
5. Add parity tests in `__tests__/<name>.test.ts`

---

## Legacy (Go Server)

The original Go server implementation is preserved in the `internal/` directory for reference. It is no longer required for the application to function.

To run the legacy server:

```bash
# Requires Go 1.26+
go run ./cmd/app
```

---

## License

MIT
