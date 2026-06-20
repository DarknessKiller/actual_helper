# Task 4 Report: Bootstrap Accepts Provider Factory Map

**Status:** DONE

## Commits

- `b020c7a` - `feat(bootstrap): accept provider factory map instead of hardcoding TNG`

## Files Modified

| File | Change |
|------|--------|
| `internal/bootstrap/bootstrap.go` | Replaced hardcoded TNG init with factory map pattern; added `ProviderFactory` type; modified `Init()` signature |
| `cmd/app/main.go` | Updated to pass factory map with `"tng": tngprov.New` to `bootstrap.Init()` |
| `internal/providers/tng/service.go` | Changed `New()` return type from `*TNGProvider` to `providers.Provider`; added `providers` import |

## Build + Test Results

- `go build ./...` — **OK** (no output)
- `go test ./... -count=1` — **ALL PASS** (6 test suites: config, handlers, tng, rule, services)

## Concerns

- **Type system deviation from spec:** The task spec directly assigns `tngprov.New` to `bootstrap.ProviderFactory`, but Go requires exact function type matching. Since `tngprov.New` returns `*TNGProvider` and `ProviderFactory` expects `providers.Provider`, the return type of `tngprov.New` was changed to `providers.Provider` to make the assignment compile. This is a minor, necessary variance from the spec and does not affect behavior.
