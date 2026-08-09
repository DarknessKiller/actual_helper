# Security Audit: Client-Side Financial Data Processing

## Scope

Assess migration from the current Go server conversion pipeline to browser-only CSV/PDF parsing and OCR, with no sensitive data leaving the browser, encrypted local persistence, and an optional/removed server.

## Current data-flow findings

- `frontend/src/lib/api.ts:1-12` sends the selected file and optional PDF password as multipart `POST /convert/{provider}`.
- `internal/handlers/convert.go:30-63` receives the upload, logs filename and size, and returns generated CSV. Multipart parsing may also spool oversized parts to Go's temporary-file storage before the service reads them; no request/body size limit is configured in the inspected code.
- `internal/services/convert.go:26-86` reads the complete CSV or PDF into server memory and invokes provider parsing.
- `internal/pdfutil/extract.go:31-54` decrypts PDFs in server memory when a password is supplied.
- Digital PDF extraction writes the PDF to a mode-0644 temporary file (`extract.go:60-70`); pdftotext and OCR do likewise (`111-121`, `142-151`). Temporary directories are removed on normal return, but cleanup is not guaranteed on process crash, host failure, or forced termination.
- pdftotext, pdftoppm, ImageMagick `identify`/`convert`, and native Tesseract process/access financial document content on the server. OCR uses CGO/native Tesseract (`ocr_cgo.go`) and requires runtime OCR packages in `Dockerfile:23-26`.
- Provider parsing receives extracted text and produces `ActualBudgetReport`; CSV output is held in server memory and sent back to the browser.
- `internal/config/config.go:41-74` reads `PROVIDER_CONFIG_PATH` from server disk and hot-reloads it. Rules/account mappings are not sent to the browser today.
- Server logs include filename, upload size, provider, output size, and (at debug/warning level) extracted text previews/blocks and card-number-adjacent parsing previews. Logs are therefore a potential financial-data disclosure channel.
- Frontend history stores filenames, provider, timestamps, and success state in unencrypted `localStorage` (`frontend/src/lib/stores/history.ts:9-32`). The source does not persist file contents or generated CSV, but browser extensions, same-origin XSS, profiles/backups, and shared devices can read this metadata.
- Frontend fetches `/version` (`frontend/src/App.svelte:13-20`), which leaks only version but means the current UI still depends on the server origin.
- No application TLS, CSP, upload size limit, CSRF protection, security headers, or authenticated boundary is visible in the inspected code. Rate limiting exists, but it is an in-process IP limiter and does not provide confidentiality or resource isolation.

## Migration security implications

### Benefits

- Browser-only parsing removes the highest-risk transfer/storage path: upload bodies, passwords, extracted text, rendered pages, OCR output, and generated CSV need not cross a network or enter server memory/disk/logs.
- Removing the server eliminates native subprocess attack surface (`pdftotext`, `pdftoppm`, ImageMagick, Tesseract), server temp-file exposure, and provider-config file exposure.
- Offline-capable processing can provide a stronger privacy claim than merely trusting a hosted server.

### New or retained risks

- WASM/pdf.js/Tesseract.js become a client-side supply-chain and parser attack surface. Pin and audit dependencies, self-host immutable assets, verify integrity, avoid runtime CDN imports, and sandbox workers with strict CSP. Treat PDFs/CSVs as hostile parser input; enforce page, pixel, file-size, decompression, OCR-time, and memory budgets.
- Browser isolation is not absolute: extensions, malicious same-origin scripts, XSS, compromised dependencies, DevTools, OS malware, crash dumps, browser caches, download directories, and screenshots can access plaintext. “Never leaves the browser” must explicitly exclude third-party telemetry and remote assets, and must be backed by network tests/CSP.
- Web Workers improve responsiveness but are not a confidentiality boundary. Zeroize transferable buffers where practical, revoke object URLs, drop references after conversion, and avoid retaining plaintext in reactive state or error messages.
- Client-side configuration is public. Any provider rules, categories, and account mappings shipped in the bundle are readable and mutable by users; this is acceptable only if they are treated as non-secret policy. Never put credentials, signing keys, or private mappings in frontend config.
- E2E encryption at rest is not automatic. If the decryption key is stored beside IndexedDB/localStorage data, compromise of the browser profile defeats it. Prefer no persistence of source/output data; if persistence is required, use IndexedDB plus Web Crypto (AES-GCM with unique nonces, authenticated metadata, versioned format, key derived with PBKDF2/Argon2id from user-held secret), and document password-loss recovery. Do not claim encryption for the existing plaintext history.
- Generated CSV downloads remain plaintext on the OS filesystem. Offer a memory-only export path where possible and clearly warn users that browser downloads are outside application control.
- A fully static frontend removes server conversion but still needs HTTPS for hosted delivery. For offline/packaged use, authenticate release artifacts and prevent unexpected updates. Service workers/cache can retain old vulnerable bundles; version and purge caches carefully.
- If a server remains for static hosting, version/config delivery, telemetry, or optional fallback, ensure no file/password fields are sent by default, disable analytics/logging of content, separate conversion endpoints, and enforce CSP, HTTPS, security headers, and strict request policies. A fallback server would weaken the privacy promise unless explicitly opt-in.
- OCR quality and parser differences may alter financial amounts/signs. Security includes integrity: show a preview, allow user verification, and ideally provide deterministic test fixtures and a signed build. Do not silently upload failed/ambiguous documents for “fallback OCR.”

## Recommended target architecture

1. Build a static client with self-hosted, pinned pdf.js and Tesseract.js WASM/language data; no analytics, remote fonts, CDN scripts, or network calls during conversion.
2. Keep provider parsing/mapping as pure client modules; ship only public rules. Generate CSV with browser APIs and process `File.arrayBuffer()` in memory.
3. Use dedicated Web Workers, strict CSP (`default-src 'self'; worker-src 'self' blob:`, with the narrowest viable script/connect/img policies), Trusted Types where supported, and sandboxed rendering if HTML previews are added.
4. Enforce bounded input and output, reject unsupported MIME/signatures, cap PDF pages/render dimensions/OCR work, and clean up buffers/object URLs on success and error.
5. Remove or disable `/convert` in privacy mode. If retained, make it an explicit opt-in feature with a prominent disclosure and separate origin/path.
6. Replace plaintext localStorage history with non-sensitive in-memory history by default. If encrypted persistence is needed, specify key lifecycle, cryptographic parameters, deletion, and threat model before implementation.
7. Add automated browser/network tests proving selected bytes and password never appear in requests, logs, service-worker caches, IndexedDB (unless explicitly encrypted), or telemetry.
8. Add security documentation that distinguishes browser privacy from OS/browser-extension security and plaintext export/download behavior.

## Overall assessment

The migration materially improves confidentiality and removes substantial server-side attack surface, but it is not a complete privacy guarantee by itself. The current system definitely transmits sensitive files/passwords and writes PDFs/images to server temp storage; the current frontend also stores metadata unencrypted. A browser-only design is recommended, provided the build/dependency supply chain, parser resource limits, CSP/network behavior, key management, and plaintext exports are treated as first-class security requirements. Do not advertise “no sensitive data leaves the browser” until the deployed bundle is self-contained and network-tested.
