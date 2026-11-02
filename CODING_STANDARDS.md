# Coding standards

Repo-specific rules, derived from the code that already exists here. `AGENTS.md`
points to this file; the verify block at the end is what CI runs.

## Errors: return them, or log them with `slog`

Applies to every `.go` file, test files included.

1. **Return.** Wrap once per call site, lowercase, no trailing period, `%w` last:
   `return nil, fmt.Errorf("csv parsing: %w", err)` — `internal/services/convert.go:80`.
2. **Sentinels** are package-level `errors.New` values compared with `errors.Is`:
   `errFileRequired`, `errFileTooLarge` (`internal/handlers/convert.go:39`),
   `config.ErrTooManySessions` (`internal/handlers/config.go:76`). Export the value only
   when another package has to test it.
3. **Log only where returning is impossible**: the response is already committed, a
   deferred close, or best-effort cleanup. `internal/handlers/config.go:87` reads a
   request body it cannot use, so it answers 400 and logs the cause.
4. **No `_` error sink.** `_ = f()`, `v, _ := f()`, and a deferred error-returning call
   are banned. `.golangci.yml` sets `errcheck.check-blank: true` to enforce it.
5. **A call that cannot fail gets deleted, not blanked.** `multipart.Part.Close` always
   returns nil and `NextPart` drains the previous part, so `internal/handlers/convert.go`
   makes no close calls at all.
6. **Handlers map errors, services wrap them, providers hand them up.**
   Client mistakes become `fuego.BadRequestError` or `fuego.HTTPError`
   (`internal/handlers/convert.go:102`); a failed conversion becomes
   `fuego.InternalServerError` (`:117`).
7. **Tests assert the error.** `Expect(err).ToNot(HaveOccurred())`,
   `Expect(w.Close()).To(Succeed())` — `internal/handlers/convert_test.go`.

| Instead of | Write |
|---|---|
| `_ = part.Close()` | delete the call when it cannot fail, or log the failure |
| `v, _ := w.CreatePart(h)` | `v, err := ...` then `Expect(err).ToNot(HaveOccurred())` |
| `fmt.Errorf("failed")` | `fmt.Errorf("csv parsing: %w", err)` |
| `err.Error() == "file required"` | `errors.Is(err, errFileRequired)` |
| silently swallowing a close/write error | `slog.Warn("...", "error", err)` |

## Logging

- `log/slog` only. `log.Fatal` is reserved for `main` failing to start (`cmd/app/main.go`).
- Every error value goes in the `"error"` key: `slog.Warn("config upload rejected", "error", err)`.
- Services and providers carry context instead of logging statelessly: `slog.With("provider", providerName)`
  then `logger.InfoContext(ctx, "parsing complete", "records", len(reports))` —
  `internal/services/convert.go:56`.
- Levels: `Info` process facts, `Warn` for a rejected request or a failed best-effort step,
  `Error` when the caller receives a 500, `Debug` for per-row decisions.
- Never log statement text, passwords, session ids or raw account numbers; mask with
  `models.MaskAccountNumber` (`internal/providers/gxbank/service.go:57`).

## Layers

| Layer | Does | Must not | Copy from |
|---|---|---|---|
| Handler | parse request, validate transport shape, call one service, answer | touch providers, parse files | `internal/handlers/convert.go` |
| Service | business logic, provider resolution, CSV assembly, logging of flow | know HTTP status codes | `internal/services/convert.go` |
| Provider | parse its own format into `models.ActualBudgetReport` | import `net/http` or the service layer | `internal/providers/tng/service.go` |

New behaviour goes in the lowest layer that can express it; handlers stay thin.

## HTTP handlers

- Read the body once, cap it, and answer before writing a header you cannot take back.
- Commit the response last: header, then body. After the body is written the outcome is
  fixed, so a write failure is logged, never returned (`internal/handlers/response.go`).
- Answer JSON through `writeJSON`, not a bare `json.NewEncoder(w).Encode`.
- Clear sensitive buffers before returning: `defer zero(upload.data)` (`internal/handlers/convert.go:110`).

## Config and sessions

- All loader state is guarded by `Loader.mu`; sessions expire lazily on lookup, and no
  background goroutine polls anything (`internal/config/config.go`).
- Session rules reach a provider through `Reload()` before each request
  (`internal/services/convert.go:providerFor`), never by mutating the shared registry.
- A full session store returns `config.ErrTooManySessions` and the handler answers 429.

## Tests

- One suite file per package: `internal/rule/rule_suite_test.go` (`RegisterFailHandler(Fail)`
  + `RunSpecs(t, "<Pkg> Suite")`), specs in an external `<pkg>_test` package.
- Ginkgo/Gomega: `var _ = Describe(...)`, `It(...)`, `Expect(...)`. `var _ = Describe`
  and unused parameters (`_ context.Context`) discard no error, so they stay.
- Handler tests use `httptest` against `fuego.NewServer()` and `s.Mux.ServeHTTP`
  (`internal/handlers/convert_test.go`).
- Benchmarks live in `*_bench_test.go` and abort with `b.Fatal(err)`, never `_`.
- Every branch that returns an error gets a spec. Test data is synthetic: see
  [Test data is synthetic](#test-data-is-synthetic).

## Test data is synthetic

An agent that receives a pasted statement turns it into fixtures. That statement is a real
person's data, so every field is replaced before it reaches a spec, a `testdata` file or a
string literal. Copy the shapes (`RM34.00`, `20260101`, `TNGD0000`), never the values.

| Field | Placeholder |
|---|---|
| Person, payee, merchant | `Ali Ahmad`, `Ninja Cat Cafe` |
| Address | `1 Jalan Test, 50000 Kuala Lumpur` |
| Item or description | `Payment`, `Transfer`, `Item A` |
| Card number | `1234 5678 9012 3456` |
| Account number | `111111`, `•••• 1234` |
| Reference or transaction id | `20260101111111111111`, `TNGD0000` |
| Amount | `10.00`, `-10.00`, `100.00` |

- Amounts are round ringgit. Cents belong only to a spec about cents (`0.55` in
  `internal/providers/gxbank/service_test.go`), never to the pasted statement's own
  figures: a distinctive price identifies the payer as well as a name does.
- A pasted PDF is never committed. Rebuild `testdata/*.pdf` from synthetic text in the same
  column order — `internal/pdfutil/testdata/fake_statement.pdf` is the working example.
- Before adding a fixture, read every long digit run:
  `rg -n '\b\d{12,}\b' --glob '*_test.go' --glob '*.csv'`. Each hit has to be a placeholder
  from the table above, like `1234 5678 9012 3456` or `111111222222`.

## Verify

```sh
go build ./...                  # compiles
go tool golangci-lint run       # lints and checks gofmt/goimports; 0 issues
CGO_ENABLED=0 go test ./...     # full Ginkgo suite
```
