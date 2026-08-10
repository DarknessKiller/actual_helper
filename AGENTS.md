# Agent Instructions

## Project

A Go web server (Fuego) serving a browser-local converter for bank/fintech CSV and PDF files into Actual Budget-compatible CSV. Raw documents and generated CSV stay on-device; Go provider parsing and CSV conversion compile to WebAssembly.

## Architecture

```text
Browser UI → PDF.js/Tesseract.js → Go provider WebAssembly
Server → static frontend and version endpoint
```

The server must not receive conversion documents. Providers parse provider-specific CSV and PDF text. The WebAssembly build embeds `cmd/wasm/provider_config.json`; Go-side tooling can use `PROVIDER_CONFIG_PATH`.

## Conventions

- Use `strings.LastIndex` for TNG table markers because the header contains the marker.
- `ActualBudgetReport.Payee` is always empty; descriptions go in `Notes`.
- TNG credits are `Reload`, `Receive from Wallet`, `DUITNOW_RECEIVEFROM`, and `Refund`; other transactions are debits.
- Include keywords whitelist rows; empty include keywords means exclude-only filtering. Categories are case-insensitive and first-match-wins.
- Go-side configuration checks file mtime on each call. Browser rules are bundled and require a WebAssembly rebuild when changed.

## Testing

Use Ginkgo, Gomega, and httptest with fake/anonymized data.

## Code Quality

Prefer focused, simple functions. Run `go fmt ./...` and `go vet ./...` before committing. Run `go test ./...` for behavioral validation.

## Data Privacy

Never upload raw user documents for conversion; conversion stays in the browser.
