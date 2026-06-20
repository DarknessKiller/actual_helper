# Provider Config System Design

## Problem

TNG transactions with certain descriptions are hardcoded to be skipped ("Quick Reload Payment", "Daily Interest", "Via eWallet to GO+"). Users who want GO+ interest earnings (Daily Interest) in their Actual Budget export have no way to include them. Separately, the categories file (`TNG_CATEGORIES_PATH`) is TNG-specific and not reusable by future providers.

## Solution

A shared `internal/config` package that provides a single unified config file per application (not per provider) with:

- **Global section** — rules that apply to all providers
- **Per-provider section** — provider-specific overrides

Replaces both the hardcoded description filters and the `TNG_CATEGORIES_PATH` categories system.

Parsing and matching logic lives in the provider itself (not config), keeping providers fully decoupled from config concerns.

## Config File

**Environment variable:** `PROVIDER_CONFIG_PATH` (path to single JSON file)

**JSON format:**
```json
{
  "global": {
    "exclude_keywords": ["Some common noise"],
    "include_keywords": [],
    "categories": [
      {"keyword": "shopee", "group": "Shopping", "category": "Online"}
    ]
  },
  "providers": {
    "tng": {
      "exclude_keywords": ["Quick Reload Payment", "Via eWallet to GO+"],
      "include_keywords": ["Daily Interest"],
      "categories": [
        {"keyword": "grab", "group": "Food & Dining", "category": "Delivery"},
        {"keyword": "ninja", "group": "Delivery", "category": "Parcel"}
      ]
    }
  }
}
```

If the env var is unset, file missing, or JSON is invalid — log a warning, continue with no filtering and no categories. Never fail the request.

## Merge Rules

| Rule type | Merge behavior |
|---|---|
| `exclude_keywords` | Union — global + provider combined |
| `include_keywords` | Union — if any match, row is kept even if an exclude also matches |
| `categories` | Global rules added first, provider rules appended after (first match wins) |

## Hot Reload

No background goroutines or file watchers. On each request:

1. The service calls `loader.ProviderConfig(name)` — checks file mtime against cached timestamp, re-reads if changed
2. The service pushes the merged rules to the provider via `ConfigurableProvider.Reload()`
3. The next parse uses the latest rules

Changes take effect on the very next request with zero delay.

## Matching (inside provider)

- `shouldSkip()` checks `exclude_keywords` then `include_keywords` — case-insensitive, `strings.Contains`
- `matchCategory()` iterates `categories` rules — case-insensitive, first match wins
- No category match → empty group/category
- Unknown provider name → no rules (empty ProviderConfig returned)

## Architecture

```
bootstrap.Init()
  config.NewLoader(PROVIDER_CONFIG_PATH)
    → loader.ProviderConfig("tng")      ← merged global + tng rules
    → tng.New(exclude, include, cats)   ← plain data, not config object
    → returns (registry, loader)

service.ConvertFile()
  → reloadProvider(name, provider)       ← loader.ProviderConfig(name) → Reload()
  → provider.ParseCSV/ParsePDFText()     ← uses own shouldSkip/matchCategory
```

## Shared Type

`models.CategoryRule` lives in `internal/models` — used by both config (JSON deser) and providers (matching). Avoids circular imports between config and provider packages.

## Files

| File | Status |
|---|---|
| `internal/models/rule.go` | **New** — `CategoryRule` shared type |
| `internal/config/config.go` | **New** — `Config`, `ProviderConfig`, `Loader` with hot-reload via mtime |
| `internal/config/config_test.go` | **New** — loader, merge, hot-reload tests |
| `internal/bootstrap/bootstrap.go` | **New** — creates loader, extracts merged rules, creates providers |
| `internal/providers/provider.go` | **Modify** — added `ConfigurableProvider` interface |
| `internal/providers/tng/service.go` | **Modify** — removed config import, owns keyword lists + category rules, `shouldSkip()`, `matchCategory()` |
| `internal/providers/tng/service_test.go` | **Modify** — constructor takes plain data, not config |
| `internal/providers/tng/pdf_test.go` | **Modify** — same pattern |
| `internal/handlers/convert_test.go` | **Modify** — no config import needed |
| `internal/services/convert.go` | **Modify** — accepts loader, calls `reloadProvider` per-request |
| `cmd/app/main.go` | **Modify** — uses `bootstrap.Init()` |
| `provider_config.example.json` | **New** — example file shipped in repo |

## Tests

- `config_test.go`: Load valid JSON, missing file, invalid JSON, merge global+provider, hot-reload via mtime change
- `service_test.go`: Filtering via keywords, categorization via category rules, no config = no filtering
- `pdf_test.go`: Same patterns as CSV tests
- `convert_test.go`: Handler tests work without config import
- `services/convert_test.go`: Service tests pass nil loader (no reload)
