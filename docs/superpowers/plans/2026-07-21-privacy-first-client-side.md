# Privacy-First Client-Side Conversion — Implementation Plan

**Date:** 2026-07-21
**Spec:** `docs/superpowers/specs/2026-07-21-privacy-first-client-side.md`
**Strategy:** Strangler pattern — incremental, provider-by-provider, Go server as fallback until parity.

---

## Phase 1: Foundation (Browser Infrastructure)

### 1.1 Install & Configure WASM Dependencies
- `pdf.js` (pdfjs-dist)
- `tesseract.js`
- Web Worker infrastructure

### 1.2 Shared Types
- `ActualBudgetReport` TypeScript interface (mirror `internal/models/actual_report.go`)
- `ProviderConfig` interface (mirror `internal/config/config.go`)
- `ParseResult` type

### 1.3 PDF.js Worker
- Text extraction from digital PDFs
- Password-protected PDF handling
- Page-by-page processing with progress
- Web Worker wrapper with cancel support

### 1.4 Tesseract.js Worker
- OCR initialization with bundled eng + msa models
- Page rendering via pdf.js canvas
- Progress callbacks
- Cancel/abort support

### 1.5 Rule Engine Port
- `shouldSkip()` — keyword filter (include/exclude whitelist)
- `matchCategory()` — category matcher
- `accountMapping` — card number → friendly name
- Config loading from encrypted IndexedDB or bundled defaults

### 1.6 CSV Serializer
- `toActualCSV()` — reflection-free, direct field mapping
- Blob creation for download

**Exit criteria:** Can parse a simple CSV and generate output locally.

---

## Phase 2: Provider Parsers (TypeScript Port)

Port each provider's parsing logic from Go to TypeScript. Use parity fixtures.

### 2.1 TNG (Digital PDF)
- Port `internal/providers/tng/` parsing
- Date parsing, credit/debit detection, reference token detection
- Fixture: synthetic TNG PDF → expected CSV

### 2.2 RYT Bank (Digital PDF)
- Port `internal/providers/ryt/` parsing
- Account extraction from header
- Fixture: synthetic RYT PDF → expected CSV

### 2.3 HLB (PDF text extraction)
- Port `internal/providers/hlb/` parsing
- Auto-detect credit vs debit format
- Layout heuristics for pdftotext replacement
- Fixture: synthetic HLB PDF → expected CSV

### 2.4 GXBank (Digital PDF)
- Port `internal/providers/gxbank/` parsing
- Account extraction from header
- Fixture: synthetic GXBank PDF → expected CSV

### 2.5 UOB Credit (PDF text extraction)
- Port `internal/providers/uobcredit/` parsing
- Card type detection, statement date inference
- Fixture: synthetic UOB PDF → expected CSV

### 2.6 HSBC Credit (OCR)
- Port `internal/providers/hsbccredit/` parsing
- Tesseract.js OCR pipeline
- Fixture: synthetic HSBC scanned PDF → expected CSV

**Exit criteria:** All 6 providers produce matching CSV from parity fixtures.

---

## Phase 3: Encrypted Storage

### 3.1 Web Crypto Integration
- PBKDF2 key derivation from passphrase
- AES-256-GCM encrypt/decrypt helpers
- Key caching (session-only, never persisted)

### 3.2 Encrypted IndexedDB
- Provider config storage (encrypted)
- Conversion history storage (encrypted, minimal metadata)
- Account mappings storage (encrypted)
- Passphrase prompt on app start (optional)

### 3.3 Config Import/Export
- Import provider config JSON file (encrypted before storage)
- Export encrypted config backup
- Bundle default non-sensitive rules

**Exit criteria:** History and config encrypted at rest, readable with passphrase.

---

## Phase 4: Frontend Integration

### 4.1 Local Conversion Pipeline
- Replace `api.ts` `convertFile()` with local pipeline
- PDF → pdf.js/Tesseract.js → provider parser → rule engine → CSV Blob
- Direct Blob download (no server round-trip)

### 4.2 Progress & UI
- Multi-step progress indicator (Extract → Parse → Filter → Generate)
- OCR progress with page count
- Cancel button
- Error handling with provider-specific messages

### 4.3 Offline Support
- Service Worker for asset caching
- PWA manifest
- Offline-first operation
- Asset versioning and cache invalidation

### 4.4 Security Hardening
- Strict CSP (no external requests during conversion)
- No `eval()` or `new Function()` in conversion path
- Input validation on all parsed data
- Memory cleanup after processing

**Exit criteria:** Full conversion works offline, no network requests, progress UI functional.

---

## Phase 5: Go Server Cleanup

### 5.1 Static-Only Mode
- Remove `/convert` endpoint (or make opt-in via env var)
- Remove provider registry from startup
- Remove PDF/OCR native dependencies
- Keep as optional static file host

### 5.2 Remove Server Conversion Code
- `internal/handlers/convert.go` → remove
- `internal/services/convert.go` → remove
- `internal/providers/` → archive or remove
- `internal/pdfutil/` → remove
- `internal/rule/` → remove (now in browser)

### 5.3 Security Hardening (Server)
- Restrictive temp file permissions (if any remain)
- Minimize/redact logs
- Add security headers (CSP, X-Frame-Options, etc.)

**Exit criteria:** Server is minimal static host, no conversion data reaches it.

---

## Phase 6: Validation & Testing

### 6.1 Parity Tests
- Automated comparison: Go server output vs browser output for all fixtures
- OCR accuracy benchmark (Tesseract.js vs native Tesseract)
- PDF text extraction comparison (pdf.js vs pdftotext)

### 6.2 Privacy Tests
- Network interception: verify no conversion requests emitted
- Service Worker: verify offline operation
- Encrypted storage: verify data unreadable without passphrase

### 6.3 Performance Tests
- Large PDF processing time
- OCR memory usage
- Bundle size validation (< 50MB)

### 6.4 Security Audit
- CSP validation
- Input sanitization review
- Memory cleanup verification

**Exit criteria:** All tests pass, privacy boundary proven.

---

## File Changes Summary

### New Files (Frontend)
- `frontend/src/lib/crypto.ts` — Web Crypto helpers
- `frontend/src/lib/pdf-worker.ts` — PDF.js Web Worker
- `frontend/src/lib/ocr-worker.ts` — Tesseract.js Web Worker
- `frontend/src/lib/convert.ts` — Local conversion pipeline
- `frontend/src/lib/providers/` — 6 provider parsers (TypeScript)
- `frontend/src/lib/rules.ts` — Rule engine port
- `frontend/src/lib/csv.ts` — CSV serializer
- `frontend/src/lib/encrypted-db.ts` — Encrypted IndexedDB wrapper
- `frontend/src/lib/parity/` — Parity fixtures and comparison tests
- `frontend/public/sw.ts` — Service Worker
- `frontend/public/manifest.json` — PWA manifest

### Modified Files
- `frontend/src/lib/api.ts` — Replace with local pipeline
- `frontend/src/lib/components/UploadForm.svelte` — Progress UI, cancel
- `frontend/src/lib/stores/history.ts` — Encrypted storage
- `frontend/package.json` — Add pdf.js, tesseract.js deps
- `frontend/vite.config.js` — Worker bundling, CSP

### Removed Files (Phase 5)
- `internal/handlers/convert.go`
- `internal/services/convert.go`
- `internal/services/universal.go`
- `internal/providers/` (all provider packages)
- `internal/pdfutil/` (all OCR/extraction)
- `internal/rule/engine.go`

### Removed Dependencies (Phase 5)
- `tesseract-ocr`, `poppler-utils`, `imagemagick` (Dockerfile)
- `gosseract`, `ledongthuc/pdf`, `pdfcpu` (go.mod)
