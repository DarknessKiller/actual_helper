# Agent Instructions

## Project

A Go web server (Fuego) that converts bank/fintech transaction files (CSV or PDF) into Actual Budget-compatible CSV format. Designed for multiple providers.

Conversion is server-side: the browser uploads the raw file to `POST /convert/{provider}` and the server parses it and returns Actual Budget CSV. Raw transaction documents do reach the server — there is no client-side parsing.

### Tech Stack

* Go 1.27 (module directive `go 1.27`, toolchain `go1.27.0`)
* Fuego
* Ginkgo/Gomega

---

## Architecture

Follow strict layer separation:

```text
Handler
  ↓
Service
  ↓
Provider (interface)
```

### Handlers

Responsibilities:

* Parse HTTP requests
* Validate request payloads
* Call services
* Return HTTP responses

Handlers MUST NOT:

* Access providers directly
* Contain business logic
* Execute file processing

### Services

Responsibilities:

* Business logic
* Validation beyond transport concerns
* Orchestration between providers
* File processing (PDF/CSV routing)

Services MAY:

* Access the provider registry
* Access the config loader
* Perform transformations
* Log process flow
* Return domain errors

### Providers

Responsibilities:

* Parse provider-specific file formats (CSV, PDF)
* Map provider-specific fields to ActualBudgetReport
* Persistence logic for provider-specific parsing rules

Providers MUST NOT:

* Contain HTTP concepts
* Know about the service layer
* Handle file format detection (PDF vs CSV)

### Models

* `ActualBudgetReport` is the single output model in `internal/models`
* Each provider keeps its own input model in its package
* Avoid duplicate DTOs

---

## Conventions

### PDF Extraction Pattern

`ledongthuc/pdf` extracts table content **column-by-column** — each cell value is on its own line, not row-by-row. Test data must use columnar format:

```
1/5/2026
Success
Payment
111111
Merchant A
222222
RM34.00
RM5.10
```

Not row-wise:
```
1/5/2026 Success Payment 111111 ...
```

### Marker Detection

Use `strings.LastIndex` to skip header lines containing the marker as a substring. The TNG PDF has "TNG WALLET TRANSACTION HISTORY" (header) before "TNG WALLET TRANSACTION" (table). Using `strings.Index` matches the header first — use `LastIndex` to find the actual table.

### Reference Token Detection

`isReferenceToken` catches:
- 10+ all-digit tokens
- 14+ chars with 8+ leading digits + letters (YYYYMMDD-prefixed reference IDs)
- Letter prefix followed by only digits (e.g. "ABC123")
- Known prefixes: TNGD, TNGQR, TNGOW

It does NOT catch short all-digit tokens (table/order numbers like "1314") or tokens with interspersed non-digit characters after the first digit.

### Payee

Always empty in `ActualBudgetReport`. Description value goes in `Notes`.

### Credit/Debit Detection (TNG)

`isCredit()` returns true for: `Reload`, `Receive from Wallet`, `DUITNOW_RECEIVEFROM`, `Refund`, `GO+ Daily Earnings`, `GO+ Cash In`. Credits are positive amounts, debits are negative.

### Config Lifecycle (not provider activation)

Providers are statically compiled into the binary and **always active** (registered in `cmd/app/main.go`). Config is **tuning only** — exclude/include keywords, category rules, account mappings — applied via `ConfigurableProvider.Reload`. The lifecycle endpoints below manage config, never provider activation. There is no provider unload, no plugin/.so/WASM.

Config is **empty by default**: nothing is auto-loaded at startup and nothing is read from disk per request. The `config.Loader` holds the active config in memory; `ProviderConfig(name)` reads memory under a mutex with no file I/O.

Endpoints:

* `GET /config` — download the sample config file at `PROVIDER_CONFIG_PATH` (`application/json`); 404 when the path is empty or unreadable. The file is read on demand and **never auto-applied**.
* `POST /config` — upload a user config JSON `{global, providers}`. Validated by unmarshalling into the config struct; invalid JSON or empty body → 400 (the existing config stays untouched). On success the merged rules for every registered `ConfigurableProvider` are pushed via `Reload(...)` — hot reload, applies to the next request.
* `DELETE /config` — unload: applies empty tuning (`Reload(nil, nil, nil, nil)`) to every `ConfigurableProvider` and clears the in-memory config.

Format:
```json
{
  "global": {
    "exclude_keywords": ["Global Noise"],
    "include_keywords": [],
    "categories": [{"keyword": "shopee", "group": "Shopping", "category": "Online"}]
  },
  "providers": {
    "tng": {
      "exclude_keywords": ["Quick Reload Payment", "Via eWallet to GO+"],
      "include_keywords": ["Daily Interest"],
      "categories": [
        {"keyword": "grab", "group": "Food & Dining", "category": "Delivery"}
      ]
    }
  }
}
```

Per-provider rules are merged over `global`: `exclude_keywords`/`include_keywords` unions; `categories` = global first, then provider-specific, first match wins; `account_mappings` = provider-specific only.

`include_keywords` act as a whitelist when configured: only rows matching any include keyword pass through; everything else is skipped. When `include_keywords` is empty, exclude-only filtering applies.

### Hot-Reload

No background goroutines and no file-mtime polling (the old per-request `os.Stat`/`ReadFile`/`json.Unmarshal` path is gone). The service calls `loader.ProviderConfig(name)` — a memory read under the loader's mutex — and pushes the merged rules to the provider via `ConfigurableProvider.Reload()` **before each request** (`services/convert.go:reloadProvider`). A `POST /config` therefore takes effect on the next request with zero delay.

Rejected uploads — invalid JSON is returned as a 400 by the handler and leaves the current in-memory config untouched; providers keep their existing tuning.

### Shared Mapping

`toActualReports()` is a shared mapper used by both `ParseCSV` and `ParsePDFText`. It handles filtering (non-Success status, filtered descriptions via `shouldSkip`), date parsing, credit/debit sign, categorization (via `matchCategory`), and `ActualBudgetReport` construction.

### Filtering Rules

Providers use `shouldSkip()` via `rule.Engine`. When `include_keywords` are configured (merged from global + provider config), they act as a **whitelist**: only rows matching any include keyword pass through; everything else is skipped. When `include_keywords` is empty, exclude-only filtering applies (matching `exclude_keywords` are skipped). No config and no keywords → nothing is filtered.

### Auto-Categorization

Providers use `matchCategory()` via `rule.Engine`. Case-insensitive, first match wins. Rules come from the merged `ProviderConfig` (global + provider-specific).

### Sample Provider

`internal/providers/sample/` is a source-only reference implementation of the full contract (`providers.Provider`, `providers.ConfigurableProvider`, `io.Closer`). It parses deterministically with synthetic data — no network, no credentials, no real personal data — and is **not** registered in `cmd/app/main.go`. To activate it, add its factory to the bootstrap map (see `docs/providers.md`). It demonstrates the real contract; it is not a separate architecture.

### Environment Variables

* `PROVIDER_CONFIG_PATH` — path to the sample config file served by `GET /config` (default empty → 404). Never auto-applied.
* `LOG_LEVEL=debug` — enables debug logging

---

## Testing

Use:

* Ginkgo
* Gomega
* httptest

### OCR via tesseract CLI subprocess

OCR (`hsbccredit`) shells out to the `tesseract` CLI via `exec.CommandContext`, not a CGO binding. This makes OCR cleanly cancellable on context timeout (no stuck-forever, no leaked C memory). The project builds with `CGO_ENABLED=0` — no tesseract dev headers needed for compilation. The runtime image must have `tesseract-ocr` + language data installed (see Dockerfile runtime stage).

### Data Privacy

All test data MUST use fake/anonymized data — no real names, reference IDs, account numbers, or other personal information. Real personal data should never be committed to the repository.

Every implementation should include tests for:

### Success Paths

* Expected behavior
* Happy path requests

### Failure Paths

* Validation failures
* Repository failures
* Service errors
* Upstream errors

### Edge Cases

* Empty inputs
* Missing fields
* Duplicate data
* Boundary conditions
* Nil/zero values

### Handler Tests

Use `httptest` to verify:

* Status codes
* Response bodies
* Error handling
* Request validation

---

## Task Workflow

For EVERY assigned task:

1. Use brainstorming skill, write spec to `docs/superpowers/specs/`
2. Complete implementation
3. Update task status
4. Verify tests pass

---

## Code Quality Rules

* Prefer simple solutions.
* Keep functions focused.
* Follow existing project conventions.
* Avoid premature abstractions.
* Do not introduce new dependencies without justification.
* Add tests for all new behavior.
* Preserve backward compatibility unless explicitly instructed otherwise.

---

## Deliverables

For each task provide:

1. Summary of changes
2. Files modified
3. Tests added
4. Test results
5. Follow-up recommendations
