# Agent Instructions

## Project

A Go web server (Fuego) that converts bank/fintech transaction files (CSV or PDF) into Actual Budget-compatible CSV format. Designed for multiple providers.

### Tech Stack

* Go
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

`isCredit()` returns true for: `Reload`, `Receive from Wallet`, `DUITNOW_RECEIVEFROM`, `Refund`. Credits are positive amounts, debits are negative.

### Filtered Descriptions (TNG)

Transactions with descriptions containing these are skipped: `Quick Reload Payment`, `Daily Interest`, `Via eWallet to GO+`.

### Shared Mapping

`toActualReports()` is a shared mapper used by both `ParseCSV` and `ParsePDFText`. It handles filtering (non-Success status, filtered descriptions), date parsing, credit/debit sign, categorization, and `ActualBudgetReport` construction.

### Auto-Categorization

* Rules file: JSON at `TNG_CATEGORIES_PATH` env var
* Format: `[{"keyword":"ninja","group":"Delivery","category":"Parcel"}]`
* Case-insensitive, first match wins
* Missing/invalid file logs warning and continues without categories
* Hot-reload via mtime check on each request

### Environment Variables

* `TNG_CATEGORIES_PATH` — path to categories JSON
* `LOG_LEVEL=debug` — enables debug logging

---

## Testing

Use:

* Ginkgo
* Gomega
* httptest

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
