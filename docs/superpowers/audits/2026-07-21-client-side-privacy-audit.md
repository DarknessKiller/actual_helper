# Client-Side Privacy and Server-Surface Audit

**Date:** 2026-07-21  
**Scope:** Conversion data flow, temporary files, logging, browser persistence, and feasibility of moving conversion into the browser.

## Executive summary

The core privacy finding is confirmed: conversion is server-side. The browser uploads the source file and optional PDF password to `POST /convert/{provider}`; the Go application then performs parsing, PDF decryption/extraction, OCR, provider mapping, filtering, categorization, and CSV generation. Consequently, source documents, passwords, extracted financial text, transactions, and generated output reach the Go server.

The current application does not intentionally persist uploads in a database or permanent upload store. However, PDF/OCR processing writes plaintext temporary artifacts, and cleanup is deferred/best-effort. Logs and browser history can also expose metadata or sensitive previews. A browser-only design is feasible, but it requires a substantial migration of provider logic and the PDF/OCR pipeline—not merely a frontend API change.

## Priority findings and recommendations

### P0 — Conversion violates the browser-only privacy boundary

**Confirmed evidence**

- `frontend/src/lib/api.ts` uploads the file and PDF password to `POST /convert/{provider}`.
- Go performs all parsing, decryption, provider mapping, filtering, categorization, and CSV generation.
- No browser-side parser, PDF.js, Tesseract.js, IndexedDB, or Web Crypto implementation exists.
- Provider configuration and account mappings are server-side.

**Impact:** Financial source data and credentials leave the browser, even if transport is protected by TLS. The server necessarily has access to plaintext data to perform conversion.

**Recommendation**

1. Define a versioned client-side parser contract around `ActualBudgetReport`.
2. Port or compile provider CSV/mapping/filter/category logic for browser execution.
3. Replace `convertFile` with a local pipeline that returns a CSV `Blob` without calling `fetch`.
4. Remove or disable `/convert` once parity is demonstrated; add a regression test asserting conversion makes no network request.

**Acceptance criteria:** Conversion succeeds offline, no conversion request is emitted, and anonymized fixture outputs match current behavior.

### P0 — Plaintext temporary PDF/OCR artifacts

**Confirmed evidence**

`internal/pdfutil/extract.go` creates plaintext temporary PDF, PNG, and crop files for digital extraction and OCR. Cleanup is deferred/best-effort and artifacts are not encrypted. Files are created with permissive mode (`0644`).

**Impact:** Sensitive documents can be readable by other processes during processing and may remain exposed through OS/container behavior, crash dumps, swap, or failed cleanup. This is transient filesystem exposure, not evidence of intentional permanent upload storage.

**Immediate mitigation (while server conversion exists)**

- Set restrictive permissions (`0600`) on temporary files and directories.
- Prefer a dedicated private temporary directory and minimize artifact lifetime.
- Check and report cleanup failures; avoid logging paths or extracted content.
- Review subprocess arguments/environment and host-level swap, crash-dump, and log policies.

**Target state:** Eliminate server PDF/OCR processing by moving extraction and OCR to browser workers with locally bundled assets.

### P1 — Sensitive values may enter logs

**Confirmed evidence**

Request logs include provider, filename, and size. Debug/warning paths can include transaction descriptions, raw values, card-related previews, or extracted-text previews.

**Impact:** Financial metadata and potentially account/card information may be retained in centralized or host logs beyond request lifetime.

**Recommendation**

- Treat filenames, descriptions, raw parser values, and extracted text as sensitive.
- Remove them from normal, warning, and debug logs; log only stable event types and counts.
- Add redaction tests and document a log-retention policy.
- Ensure production logging cannot be enabled at verbose levels accidentally.

### P1 — Browser history stores identifying metadata in plaintext

**Confirmed evidence**

History uses plaintext `localStorage` and includes provider, original filename, timestamp, and success metadata. It does not store source contents or generated CSV.

**Impact:** Filenames may reveal account holders, institutions, dates, or statement periods; any same-origin script can read the history.

**Recommendation**

- Make history opt-in and off by default.
- Store only the minimum metadata, preferably a user-supplied label rather than the original filename.
- Provide clear/delete-all controls and bound retention.
- If persistence is required, use Web Crypto with a user-held key; do not imply this protects a compromised browser/device.

### P1 — Browser-only migration requires replacing native PDF/OCR dependencies

**Confirmed evidence**

The server uses `ledongthuc/pdf`, `pdftotext`, `pdftoppm`, ImageMagick, Tesseract via CGO, and `pdfcpu` password handling.

**Recommendation**

- Use PDF.js in a Web Worker for digital text extraction and encrypted PDFs.
- Use Tesseract.js/WebAssembly in a worker for scanned statements, bundling required language data locally.
- Reimplement rendering, cropping, cancellation, and progress with browser APIs.
- Validate OCR accuracy and Malaysian language support, especially for HSBC scanned statements.
- Do not attempt to compile the existing filesystem/subprocess layer unchanged.

### P2 — Optional network request and server surface

`/version` causes an additional frontend network request. It is not needed for conversion and should be replaced with build-time metadata or removed. The Go server may remain as an optional static asset host, but it must not receive conversion data or sensitive configuration.

## Conditional or unverified claims

The following should **not** be reported as confirmed vulnerabilities without deployment evidence:

- **No TLS enforcement:** TLS may be terminated by a reverse proxy or hosting infrastructure.
- **No CSRF protection:** The conversion endpoint appears unauthenticated and does not use session cookies. CSRF becomes relevant if deployed behind authenticated browser credentials.
- **No upload size limit:** No explicit application limit was found, but framework/proxy limits were not fully verified.
- **Recoverable files after deletion:** Plausible, but not guaranteed. The confirmed issue is plaintext temporary storage with best-effort cleanup.
- **Third-party asset/model requests:** No current frontend evidence; this is a requirement to verify during migration/deployment.
- **Permanent server storage:** False based on the inspected application. No intentional database or persistent upload storage was found.

## Recommended migration sequence

1. **Freeze the privacy contract:** document that conversion inputs, passwords, extracted text, transactions, and outputs must remain in-browser.
2. **Create parity fixtures:** use synthetic/anonymized CSV and PDF/OCR fixtures and preserve current expected `ActualBudgetReport`/CSV outputs.
3. **Port provider behavior:** move detection, parsing, date/year inference, signs, filtering, categorization, and account mapping behind the client parser contract.
4. **Build the local extraction pipeline:** PDF.js and Tesseract.js/WebAssembly in workers; bundle assets and language data; support cancellation and password disposal.
5. **Generate locally:** create the Actual Budget CSV `Blob` in the browser and download it directly.
6. **Make configuration local:** bundle non-sensitive rules or accept a local import; do not fetch account mappings from a server.
7. **Prove the boundary:** offline test, browser network inspection, and automated tests asserting no conversion `fetch` calls.
8. **Remove server conversion:** delete or isolate `/convert`, provider registry, server conversion config, and native PDF/OCR dependencies after parity validation.
9. **Harden interim deployment:** restrictive temp-file permissions, minimized/redacted logs, bounded request sizes at the deployment boundary, and documented cleanup/retention behavior.

## Verification status

`go test ./...` could not complete because the environment lacks Tesseract/Leptonica development headers. This is an environment/toolchain limitation, not evidence that the findings are false. Re-run the full suite in a build environment with the required native dependencies before implementation sign-off.

## Final verdict

The privacy assessment is valid: the present conversion architecture sends sensitive financial data to the Go server. Achieving a strict browser-only boundary requires moving or reimplementing every conversion layer, including provider parsers, configuration, PDF handling, OCR, and CSV generation. Security-header, TLS, CSRF, upload-limit, and third-party-request statements should remain conditional or unverified until deployment-specific evidence is collected.
