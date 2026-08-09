# Privacy-First Client-Side Architecture Audit

## Status

Individual check: `what-go-server-funct`

## Scope

Assess which current Go server functionality can move to the browser, with goals of local-only parsing/OCR, no sensitive file data leaving the browser, encrypted local persistence, and an optional/removed server.

## Current data flow

1. The Svelte `UploadForm` selects a provider and a local CSV/PDF, then calls `convertFile()`.
2. `frontend/src/lib/api.ts` places the file and optional PDF password in `FormData` and sends `POST /convert/{provider}`. The raw file, password, and provider therefore leave the browser.
3. `internal/handlers/convert.go` receives the multipart stream, logs provider/filename/size, and passes the open file to `ConvertService`.
4. `internal/services/convert.go` looks up the provider, hot-reloads provider rules, dispatches by MIME type, and returns Actual Budget CSV bytes.
5. CSV bytes are read into server memory and passed to provider parsing. PDFs are read into memory, optionally decrypted, extracted, and then parsed.
6. `internal/services/universal.go` serializes `ActualBudgetReport` values to CSV; the HTTP handler returns the CSV as a download.
7. The browser consumes the response blob and downloads it. History stores only metadata in plaintext `localStorage`.

## Server-side functionality and client replacement

| Functionality | Current implementation | Client-side replacement | Assessment |
|---|---|---|---|
| File selection/drag-drop | Svelte/browser APIs | Existing browser APIs | Already client-side |
| CSV reading | Go `encoding/csv` in each provider | Web File API + JS CSV parser or small custom parser | Straightforward |
| PDF decryption | pdfcpu in Go service | pdf.js password callback; test supported encryption variants | Feasible, compatibility must be tested |
| Digital PDF extraction | `ledongthuc/pdf` using temp file | pdf.js text content extraction | Feasible; layout/order may differ and provider fixtures need parity tests |
| `pdftotext -layout` | External subprocess and temp PDF | pdf.js text items plus provider-specific layout reconstruction | Feasible but not drop-in; HLB/UOB layout-sensitive parsing needs validation |
| PDF rendering | `pdftoppm` subprocess | pdf.js page rendering to canvas/ImageBitmap | Feasible |
| Image cropping/strips | ImageMagick `identify`/`convert` and temp PNGs | Canvas/offscreen canvas slicing | Feasible |
| OCR | cgo Tesseract (`eng`, `msa`) through gosseract | Tesseract.js/WASM with equivalent language data | Feasible, slower/larger assets, worker/CSP/cache design required |
| Provider parsing/mapping | Six Go provider packages implementing `Provider` | Port provider algorithms to TypeScript, or compile parsing core to WASM | Possible, largest migration risk; no current shared portable core |
| Filtering/categories/account mappings | Hot-reloaded Go JSON config merged by `config.Loader` | Bundled/default config plus user-imported local JSON; merge in browser | Feasible; server config cannot be fetched without creating a data/control dependency |
| Actual Budget CSV serialization | Go reflection + `encoding/csv` | Explicit TypeScript column schema + CSV escaping | Straightforward; avoid reflection semantics drift |
| Output download | HTTP response blob | Browser `Blob`/object URL/download | Existing pattern can be reused without fetch |
| Version endpoint/static hosting | Go serves embedded frontend and `/version` | Static hosting, PWA, or local file/app shell | Server becomes optional |
| Rate limiting/API/error transport | Fuego endpoint and middleware | Not needed for local conversion; retain only if optional remote mode remains | Removable in local-only mode |

## File I/O and sensitive-data exposure

### Go file paths

- `ConvertHandler` reads multipart data through the HTTP request.
- `ConvertService` reads CSV/PDF bytes with `io.ReadAll`, so sensitive content is held in server memory.
- `pdfutil.ExtractText` reads the complete PDF, and encrypted PDFs are decrypted into a new in-memory buffer.
- Digital extraction writes `input.pdf` under `os.MkdirTemp("", "pdfutil")`, then removes the directory.
- `pdftotext` writes `input.pdf` under `pdftext`, then removes it.
- OCR writes `input.pdf` under `pdfocr`, `pdftoppm` emits page PNGs, and long pages may produce additional strip PNGs via ImageMagick; cleanup is deferred, but plaintext exists on disk during processing.
- Frontend development/static serving uses disk (`frontend/dist`) when embedded assets are unavailable; this is application code, not transaction-file persistence.
- Config loader calls `os.Stat`/`os.ReadFile` on `PROVIDER_CONFIG_PATH`; provider rules are server-local and not exposed by an endpoint.
- Tests write temporary config files, but are not production flow.

### Other exposure

- The handler logs provider, original filename, and byte size. It does not log transaction contents, but filenames can contain account/person data.
- The PDF password is sent over the conversion request and exists in server request handling; HTTPS protects transport only.
- No server database or intentional uploaded-file retention was found. Temp files are best-effort cleaned with `defer`; crashes/forced termination can leave residue in the OS temp directory.
- The browser stores conversion history (id, provider, filename, timestamp, success) in plaintext `localStorage`; it does not store source files or output CSV currently.
- `/version` is the only frontend fetch besides conversion.

## OCR pipeline

HSBC selects `ExtractionMethodOCR`. The server pipeline is:

`PDF bytes -> optional pdfcpu decrypt -> temp PDF -> pdftoppm PNG pages -> ImageMagick height/crops -> gosseract/Tesseract (eng + msa) -> provider ParsePDFText -> ActualBudgetReport`.

Other providers use digital extraction (`ledongthuc/pdf`) or `pdftotext -layout`. Browser replacement is `File/ArrayBuffer -> pdf.js (password callback) -> text extraction OR canvas rendering -> Tesseract.js worker -> provider parser`. Tesseract.js language data and WASM must be pinned and loaded locally if “no sensitive data leaves browser” includes third-party asset/CDN isolation. OCR output remains sensitive and must not be sent to telemetry.

## Provider/config implications

- Providers are entirely Go-side today: `tng`, `ryt`, `hsbccredit`, `hlb`, `uobcredit`, and `gxbank`; the interface includes `ParseCSV`, `ParsePDFText`, and `ExtractionMethod`.
- Several providers reject CSV or are PDF-specific, so a browser implementation must preserve those intentional errors and extraction method choices.
- Provider parsing contains date inference, statement/header detection, sign/credit rules, reference-token filtering, account/card mapping, and provider-specific table layouts. These are business rules, not generic PDF parsing.
- `config.Loader` merges global and provider rules on every request based on mtime. A static browser app cannot hot-reload a server file. Replace with an explicit local config import/editor, bundled config, or optional local-file access; never fetch sensitive transaction data to obtain config.
- Config can contain account mappings and category rules, which are not inherently transaction data but can reveal account names. Encrypt if persisted locally.

## Existing frontend findings

- `UploadForm.svelte` has provider selection, file selection, PDF password input, status/error handling, and download flow, but no parsing or conversion logic.
- `api.ts` is the sole transaction network path and must be removed/branched for local mode.
- `App.svelte` fetches `/version`; remove or make optional for a serverless build.
- History uses plaintext `localStorage` in `history.ts`; use IndexedDB with Web Crypto encryption if retaining history/output/config.
- No `pdf.js`, Tesseract.js, WASM, crypto, IndexedDB, service worker, or offline processing dependency exists in `frontend/package.json`.
- `frontend/embed.go` and Go frontend routes couple normal deployment to the Go server but do not process transaction files.

## Recommended target architecture

1. Make a browser conversion core with explicit stages: `File -> bytes -> PDF/CSV extraction -> provider parser -> report model -> CSV`.
2. Port provider rules to TypeScript first (or isolate a carefully bounded portable parsing core compiled to WASM). Keep provider-specific parsing separate from extraction adapters.
3. Use pdf.js locally for password-protected/digital PDFs and Tesseract.js workers for OCR. Bundle or self-host all WASM/model assets; disable analytics and third-party network calls.
4. Keep source bytes, passwords, OCR text, reports, and output in memory where possible. Revoke object URLs and clear buffers after conversion.
5. Replace hot reload with user-selected local config and an in-memory merged rule set. Validate config schema client-side.
6. Generate CSV with deterministic explicit columns and download directly. Add golden parity tests against existing provider fixtures before removing Go paths.
7. Encrypt any retained history, config, or output using Web Crypto (AES-GCM with a user-held key/passphrase; never persist the raw key). Treat browser storage as potentially accessible to same-origin script/XSS.
8. Ship as static/offline-capable frontend/PWA. Keep Go as an optional compatibility API only behind an explicit remote mode, clearly warning that files leave the device.

## Risks and validation gates

- PDF text item ordering/coordinates can differ from `pdftotext` and `ledongthuc/pdf`; compare all provider fixtures.
- pdf.js password/decryption support and malformed/encrypted statement compatibility require real-format tests using anonymized fixtures.
- Tesseract.js accuracy/performance and `eng`/`msa` model parity require HSBC OCR acceptance tests; browser memory limits matter for multi-page PDFs.
- WASM/model downloads can violate privacy if CDN or telemetry endpoints are used; enforce CSP and inspect network traffic offline.
- Client-side encryption protects stored data, not a compromised browser origin or XSS; use strict CSP, dependency pinning, and no third-party scripts.
- Do not claim “no data leaves the browser” while an optional remote endpoint is enabled by default.

## Conclusion

All transaction processing and most supporting server functionality can be replaced client-side, but this is a substantial provider-parser port rather than a frontend-only change. The current app is explicitly server-side: raw files and PDF passwords are uploaded, server temp files are used for PDF/OCR, and provider configuration/rules live in Go. The lowest-risk migration is a local browser mode with parity tests and an opt-in remote fallback, followed by removing `/convert`, server PDF dependencies, provider registry, config hot reload, and server-side file-processing code once parity and offline privacy checks pass.
