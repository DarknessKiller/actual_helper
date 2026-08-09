# Privacy-First Client-Side Architecture Audit

## Scope

Assess whether `actual_helper` can move from server-side conversion to browser-only PDF/CSV parsing and OCR, with no sensitive data sent to the server, encrypted local persistence, and an optional/removed Go server.

## Current data flow

1. The Svelte `UploadForm` accepts a local `File` and optional PDF password.
2. `frontend/src/lib/api.ts` creates `FormData` containing the original file and password and sends it with `fetch` to `POST /convert/{provider}`.
3. `internal/handlers/convert.go` parses the multipart upload (`FormFile`), logs provider, filename, and size, then passes the file reader and password to `ConvertService`.
4. `internal/services/convert.go` reloads provider rules, reads CSV bytes or calls `pdfutil.ExtractText`, invokes the provider parser, and serializes `ActualBudgetReport` records to CSV.
5. The server returns the generated CSV. The browser turns the response into a Blob, downloads it, and stores only conversion metadata in localStorage.
6. The browser also requests `GET /version` on startup. Static frontend assets are served by the Go server (embedded in release builds or read from `frontend/dist` in development).

**Conclusion:** sensitive source files, PDF passwords, extracted text, transaction descriptions, amounts, and generated CSV contents currently leave the browser and are processed by the Go server. This is not a privacy-first architecture.

## Server-side file and process I/O

### Upload and conversion

- `internal/handlers/convert.go`: multipart `FormFile`; request body parsing may use framework multipart buffering/spooling. The handler does not explicitly persist the upload, but it is server-resident during processing.
- `internal/services/convert.go`: `io.ReadAll(file)` for CSV; PDF bytes are read by `pdfutil.ExtractText`.
- No application database or permanent transaction-file storage was found.

### PDF digital extraction

- `internal/pdfutil/extract.go` writes PDF bytes to an OS temporary directory (`os.MkdirTemp`, `os.WriteFile`), parses with `ledongthuc/pdf`, and removes the directory with `os.RemoveAll`.
- This is temporary plaintext sensitive data on server disk and may be recoverable depending on filesystem/container behavior.

### PDF text extraction

- The same file writes bytes to an OS temporary directory, invokes `pdftotext -layout`, captures extracted text in memory, and removes the temporary directory.

### OCR

- OCR providers (currently HSBC Credit) write the PDF to an OS temporary directory.
- `pdftoppm` renders pages to PNG files; ImageMagick `identify` inspects image dimensions; large pages are cropped with `convert` into additional PNG files.
- Tesseract is called through `gosseract` (`CGO` build) against those PNG files, producing text in server memory. Temporary strip files are removed individually and the directory is removed on return.
- `ocr_stub.go` makes OCR unavailable in non-CGO builds; OCR is not browser-based and no Tesseract.js pipeline exists.
- Subprocesses are context-cancellable, but temporary artifacts are not encrypted.

### Configuration and other disk I/O

- `internal/config/config.go`: `os.Stat` checks `PROVIDER_CONFIG_PATH` mtime on each request; `os.ReadFile` reads provider rules. Config can contain account mappings and categorization/filtering rules.
- Tests write temporary config files, but production has no config write path.
- Frontend static serving checks `frontend/dist` with `os.Stat`; release builds use embedded assets.
- Rate-limit settings are environment variables only.

## Provider/config behavior relevant to client migration

- Providers implement parsing and mapping in `internal/providers/*`; they expose `ParseCSV`, `ParsePDFText`, and an extraction method. The provider-specific business rules can be ported, but they are Go implementations with no shared browser/WASM interface.
- Current supported formats are effectively PDF for all listed providers; CSV support exists at the service/provider interface but the README/provider behavior should be checked per provider before claiming universal CSV support.
- Extraction methods are selected server-side by provider (`digital`, `pdftotext`, or `ocr`), not by the browser.
- `PROVIDER_CONFIG_PATH` is a server filesystem path. Hot reload merges global and provider rules before every request. A browser-only design needs a client-delivered configuration bundle or user-imported configuration. Shipping sensitive account mappings in public frontend assets would expose them to anyone who can fetch the app.
- Filtering, categorization, account mapping, date/sign handling, and Actual CSV formatting live in Go providers/services and must be reimplemented or compiled to WASM/JavaScript.

## Frontend findings

- `frontend/src/lib/api.ts` explicitly uploads the original file and password to `/convert/{provider}`.
- `UploadForm.svelte` has no local parsing, PDF rendering, OCR, encryption, wipe/cleanup, offline mode, or server opt-out.
- `App.svelte` calls `/version`; this is non-sensitive metadata but still creates a server request.
- `frontend/src/lib/stores/history.ts` stores filename, provider, timestamp, success, and random ID in plaintext `localStorage`. It does not store transaction contents, but filenames can contain account/statement identity.
- `useApiStore.ts` stores an API call count in plaintext localStorage; it appears unrelated to the active conversion flow.
- Generated CSV is deliberately downloaded to the user's normal filesystem. Browser download storage is outside the application's encryption boundary.
- No IndexedDB, Web Crypto, service worker, CSP/privacy controls, telemetry, analytics, or other outbound endpoints were found.

## Feasibility of requested target

### Browser parsing/OCR

Feasible, but it is a substantial port rather than a wiring change:

- CSV parsing and Actual CSV generation are straightforward in TypeScript.
- `pdf.js` can replace digital PDF extraction in the browser.
- OCR can use Tesseract.js/WASM, but the current pipeline also depends on PDF-to-PNG rendering, image cropping, language data (`eng`, `msa`), and provider-specific OCR normalization. Worker execution and progress/cancellation are needed to keep the UI usable.
- Password-protected PDFs need browser-compatible decryption support; pdf.js password handling should be validated for all current statements.
- Existing Go provider tests are useful behavioral fixtures, but a shared conformance corpus is needed to ensure parity.

### No sensitive data leaves browser

Achievable only if conversion never calls `/convert`, `/version`, or any telemetry endpoint, and all libraries/assets/models are either bundled or fetched from non-sensitive public URLs before processing. A strict offline/PWA build is the strongest guarantee. Browser extensions, service workers, crash reporting, CDN logs, and third-party assets must be excluded or audited.

### End-to-end encryption at rest

Not currently implemented. A browser-only design can encrypt application-managed history/config in IndexedDB with Web Crypto (for example, AES-GCM with a user-derived key), but it cannot transparently encrypt the browser's normal download directory. The generated CSV should therefore be treated as user-controlled plaintext unless an encrypted export/container is introduced. Server-side at-rest encryption is irrelevant if no sensitive data is sent, but server logs and config still need normal host protections for a deployment that retains the server.

### Optional/removed Go server

The Go server is not required for conversion logic after the port. It can remain as a static asset host, but `/convert` and `/version` should be removed/disabled in privacy mode. A fully offline static build can be hosted by any static server or opened through a suitable local app/PWA strategy. The embedded Go frontend path is a packaging/deployment concern, not a conversion dependency.

## Recommended migration shape

1. Extract provider rules and mapping into a browser-neutral specification; keep a conformance test corpus shared by Go and TypeScript/WASM implementations.
2. Add browser-local CSV conversion first, then pdf.js digital extraction, then provider OCR via Tesseract.js worker.
3. Replace `convertFile()` with a local conversion pipeline; never send `File` or password over `fetch`.
4. Bundle or self-host pdf.js, Tesseract.js WASM, trained data, and provider configuration to support offline operation and avoid third-party data disclosure.
5. Move history from plaintext localStorage to encrypted IndexedDB, with an explicit key lifecycle and “clear all” deletion. Do not claim downloaded CSV files are encrypted.
6. Add CSP, no analytics/telemetry, no external asset origins, and tests that fail if conversion performs network requests.
7. Keep Go only as an optional static host during migration; remove conversion routes and server-side PDF/OCR dependencies once browser parity is proven.

## Risk summary

- **Current privacy risk: high.** Original financial documents and passwords are uploaded to the server.
- **Current persistence risk: medium.** Temporary plaintext PDF/PNG files exist on server disk; browser history metadata is plaintext localStorage; downloads are plaintext user files.
- **Migration risk: high engineering effort.** OCR/provider parity, encrypted local state, offline asset packaging, and browser performance are the main work items.
- **Architecture verdict:** the target is technically feasible, but the current codebase is server-centric and requires a new client conversion pipeline plus removal of network conversion calls; it is not achievable through configuration alone.
