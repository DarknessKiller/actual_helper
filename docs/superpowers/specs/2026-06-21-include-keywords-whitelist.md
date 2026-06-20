# Include-Keywords as Whitelist

## Problem

When `include_keywords` are configured (in `global` or provider-specific config), they should act as a **whitelist**: only rows whose description matches any include keyword pass through; everything else is skipped. Currently they only act as a soft override over `exclude_keywords` — non-matching rows still pass through.

## Design

### Change: `internal/rule/engine.go` — `ShouldSkip`

```go
func (e *Engine) ShouldSkip(description string) bool {
    e.mu.RLock()
    defer e.mu.RUnlock()

    lower := strings.ToLower(description)

    if len(e.includeKeywords) > 0 {
        for _, kw := range e.includeKeywords {
            if strings.Contains(lower, strings.ToLower(kw)) {
                return false
            }
        }
        return true // whitelist: non-matching gets skipped
    }

    for _, kw := range e.excludeKeywords {
        if strings.Contains(lower, strings.ToLower(kw)) {
            return true
        }
    }
    return false
}
```

**Behavior matrix:**

| includeKeywords | excludeKeywords | description matches include | description matches exclude | result |
|----------------|----------------|----------------------------|----------------------------|--------|
| non-empty      | any            | yes                        | irrelevant                 | keep   |
| non-empty      | any            | no                         | irrelevant                 | skip   |
| empty/nil      | non-empty      | N/A                        | yes                        | skip   |
| empty/nil      | non-empty      | N/A                        | no                         | keep   |
| empty/nil      | empty/nil      | N/A                        | N/A                        | keep   |

### Config Merge (unchanged)

In `internal/config/config.go:77-91`, `providerConfig` already merges global + provider-specific keywords into a single flat list. This means both `global.include_keywords` and `provider.include_keywords` contribute to whitelist mode — as confirmed by the user.

### No other changes

This is a one-function change in one file, plus tests. No config, service, or provider changes needed.

## Tests

### Add to `internal/rule/engine_test.go`

| Test | Setup | Expected |
|------|-------|----------|
| Include keyword keeps matching | include=["Grab"], description="GrabFood" | false |
| Include keyword skips non-matching | include=["Grab"], description="Shopee" | true |
| Include overrides exclude (whitelist) | include=["Grab"], exclude=["Grab"], description="GrabFood" | false |
| Include skips non-matching even when exclude would match | include=["Grab"], exclude=["Shopee"], description="Shopee" | true |
| Nil include → exclude still works | include=nil, exclude=["Grab"], description="GrabFood" | true |
| Empty include → exclude still works | include=[], exclude=["Grab"], description="GrabFood" | true |
| No keywords → nothing filtered | all nil | false |

### Existing tests (all must still pass)

All 4 existing `ShouldSkip` tests and all `MatchCategory`/`Reload` tests remain unchanged and passing.

## Files Modified

1. `internal/rule/engine.go` — `ShouldSkip` logic change (~8 lines)
2. `internal/rule/engine_test.go` — add new test cases (~30 lines)
3. `AGENTS.md` — update Filtering Rules section to reflect whitelist behavior

## Verification

```bash
go test ./internal/rule/... -v
go test ./...
```
