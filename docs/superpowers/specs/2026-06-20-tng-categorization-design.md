# TNG Auto-Categorization Design

## Problem

TNG transaction exports contain payee/description text but no category mapping. Users currently get empty `CategoryGroup` and `Category` fields in the output CSV, requiring manual categorization in Actual Budget.

## Solution

A keyword-based rule system configurable via JSON file. The TNG provider loads rules at startup (hot reloadable via mtime check), matches each transaction's Description against the keyword list, and sets `CategoryGroup` + `Category` on the output.

## Config

**Environment variable:** `TNG_CATEGORIES_PATH` (path to JSON file)

**JSON format:**
```json
[
  { "keyword": "grab", "group": "Food & Dining", "category": "Delivery" },
  { "keyword": "foodpanda", "group": "Food & Dining", "category": "Delivery" },
  { "keyword": "shopee", "group": "Shopping", "category": "Online" },
  { "keyword": "7-eleven", "group": "Food & Dining", "category": "Convenience" },
  { "keyword": "top up", "group": "Income", "category": "eWallet" }
]
```

If the env var is unset, file missing, or JSON is invalid — log a warning, continue with no categories. Never fail the request.

## Hot Reload

On each `ParseCSV()` call, before matching, compare the file's last modification time against a cached timestamp. If the file changed, reload the rules. This means users can edit the JSON file without restarting the server.

## Matching

- Match against **Description** field (the payee text), case-insensitive
- **First match wins** (JSON order = priority)
- No match → `CategoryGroup` and `Category` remain empty (current behavior)

## Architecture

```
TNGProvider (struct)
  ├── categoriesPath string      ← from TNG_CATEGORIES_PATH env
  ├── rules []categoryRule       ← cached rules
  ├── lastModTime time.Time      ← file mtime for hot-reload check
  └── ParseCSV()
        ├── ensureRulesLoaded()  ← check mtime, reload if changed
        └── for each row: match Description → set CategoryGroup/Category
```

No new packages. Everything stays in `internal/providers/tng/`.

## Files

| File | Change |
|---|---|
| `internal/providers/tng/categories.go` | **New** — `categoryRule` struct, `loadRules(path)`, JSON deser, `ensureRulesLoaded()` with mtime check |
| `internal/providers/tng/categories_test.go` | **New** — load/matching/hot-reload tests |
| `internal/providers/tng/service.go` | Add `categoriesPath`, `rules`, `lastModTime` fields. Init from env in `New()`. Call `ensureRulesLoaded()` in ParseCSV |
| `internal/providers/tng/service_test.go` | Add category matching test cases |

## Tests

- Load valid JSON → rules populated correctly
- Missing file → no rules, no error
- Invalid JSON → no rules, no error
- Keyword matches case-insensitively → correct group/category
- First match wins when multiple keywords overlap
- No match → empty group/category
- Empty Description → no match
- Hot reload: change file mtime between calls → rules refresh
