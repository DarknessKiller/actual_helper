# Privacy-First Client-Side Conversion

**Date:** 2026-07-21
**Status:** Proposed
**Author:** Atlas

## Problem

Current architecture sends all sensitive data to the Go server:
- Original PDF/CSV files
- PDF passwords
- Extracted text
- Transaction data
- Generated CSV

The server writes plaintext temp files during PDF/OCR processing. No E2E encryption exists.

## Goal

Zero sensitive data leaves the browser. All processing local. Encrypted storage.

## Target Architecture

```
Browser (Svelte + WASM)
├── pdf.js          → PDF text extraction (digital PDFs)
├── Tesseract.js    → OCR (scanned PDFs, HSBC)
├── Provider parsers → 6 providers ported to TypeScript
├── Rule engine     → Filter + categorize locally
├── Web Crypto      → Encrypt IndexedDB at rest
└── Blob download   → CSV never touches server
```

Go server: **optional static hosting only** (or fully removed).

## Scope

### In scope
- PDF.js integration (Web Worker)
- Tesseract.js/WASM integration (Web Worker, eng + msa)
- Port all 6 provider parsers to TypeScript
- Port rule engine (filter, categorize, account mapping)
- Port CSV serializer
- Encrypted IndexedDB for history/config
- Offline asset bundling (WASM, language models, PDF.js)
- Strict CSP, no network requests during conversion
- Parity fixtures: synthetic data, expected outputs matching current behavior
- PWA/offline support

### Out of scope (for now)
- Removing the Go server entirely (keep as optional static host)
- Mobile app conversion
- Real-time collaboration

## Provider Migration Map

| Provider | Current Method | Browser Replacement |
|----------|---------------|-------------------|
| tng | pdf.js (digital) | pdf.js text layer |
| ryt | pdf.js (digital) | pdf.js text layer |
| hlb | pdftotext (layout) | pdf.js text layer + layout heuristics |
| hsbccredit | Tesseract OCR | Tesseract.js/WASM |
| uobcredit | pdftotext (layout) | pdf.js text layer + layout heuristics |
| gxbank | pdf.js (digital) | pdf.js text layer |

## E2E Encryption

- User-provided passphrase → PBKDF2 → AES-256-GCM key
- Encrypt: provider config, conversion history, account mappings
- Stored in IndexedDB (encrypted blobs)
- Passphrase never stored; re-prompt on session start
- Optional: encrypt in-memory processed data during session

## Acceptance Criteria

1. All 6 providers produce identical CSV output from parity fixtures
2. No network requests during conversion (verified via Service Worker interception)
3. Offline operation with bundled assets
4. Encrypted IndexedDB: data unreadable without passphrase
5. Tesseract.js OCR accuracy within 5% of server-side Tesseract (HSBC fixtures)
6. Build produces self-contained static bundle (< 50MB with language models)

## Risks

1. **OCR accuracy**: Tesseract.js may differ from native Tesseract. Mitigate with parity fixtures.
2. **PDF layout**: pdf.js text layer differs from pdftotext -layout. Mitigate with heuristics and fixtures.
3. **Browser performance**: Large PDFs/slow OCR. Mitigate with progress UI and cancellation.
4. **Language models**: eng + msa WASM models are ~15MB each. Bundle or lazy-load.

## Migration Strategy

Strangler pattern: incremental, provider-by-provider, with Go server as fallback until parity proven.
