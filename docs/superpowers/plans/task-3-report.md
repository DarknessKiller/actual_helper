# Task 3 Report: TNG Provider Uses rule.Engine

**Status:** DONE

**Commit:** `fc8f07e` — `refactor(tng): delegate filtering and categorization to rule.Engine`

## Changes

| File | Change |
|------|--------|
| `internal/providers/tng/service.go` | Replaced `excludeKeywords`, `includeKeywords`, `categories`, `sync.RWMutex` with embedded `*rule.Engine`. Rewrote `New`, `Reload`, `shouldSkip`, `matchCategory` to delegate to engine. Removed `sync` import. |
| `internal/providers/tng/service_test.go` | Removed "skips filtered description rows when exclude keywords match" and "applies categories from rules" test cases. Removed unused `models` import. |
| `internal/providers/tng/pdf_test.go` | Removed "skips transactions with filtered description when exclude keywords match" and "applies categories when provider has category rules" test cases. Removed unused `models` import. |

## Test Results

```
Ran 21 of 21 Specs in 0.018 seconds
SUCCESS! -- 21 Passed | 0 Failed | 0 Pending | 0 Skipped
--- PASS: TestTNGProvider (0.02s)
PASS
ok  	actual-helper/internal/providers/tng	0.717s
```

Full project build (`go build ./...`) — no errors.

## Concerns

None. All tests pass, build is clean, and the TNG provider now delegates filtering/categorization to the shared `rule.Engine`.
