# Streamline Card Number Config Mapping

## Problem

Three credit card providers (`hsbccredit`, `hlbcredit`, `uobcredit`) duplicate:
- `cardNumberRe` — identical regex in each `pdf.go`
- `extractAccountName` — similar logic with different markers
- `accountMapping` field + `Reload()` + lookup in `toActualReports` — identical pattern in each `service.go`

## Solution

Extract shared helpers into `internal/providers/cardutil`.

### New package: `internal/providers/cardutil/cardutil.go`

```go
package cardutil

import (
    "log/slog"
    "regexp"
    "strings"

    "actual_helper/internal/dateutil"
)

// CardNumberRe — shared card number regex (4 groups of 4 digits, spaces/dashes allowed)
var CardNumberRe = regexp.MustCompile(`(\d{4}[\s-]*\d{4}[\s-]*\d{4}[\s-]*\d{4})`)

// ExtractAfterMarker finds marker in text, extracts card number after it.
// Returns fallback if marker or card number not found.
func ExtractAfterMarker(text, marker, fallback string) string {
    idx := strings.Index(text, marker)
    if idx == -1 {
        slog.Debug("card number marker not found", "marker", marker, "preview", dateutil.Truncate(text, 600))
        return fallback
    }

    after := text[idx+len(marker):]
    after = strings.ReplaceAll(after, "\n", " ")
    after = strings.ReplaceAll(after, "-", " ")

    if matches := CardNumberRe.FindString(after); matches != "" {
        return matches
    }

    slog.Debug("card number not found after marker", "marker", marker, "preview", dateutil.Truncate(after, 600))
    return fallback
}

// ExtractNearCardType finds a card type indicator, searches nearby area for card number.
// Returns fallback if not found.
func ExtractNearCardType(text string, cardTypes []string, fallback string) string {
    for _, ct := range cardTypes {
        idx := strings.Index(text, ct)
        if idx == -1 {
            continue
        }

        start := idx
        if start > 50 {
            start -= 50
        }
        end := idx + 200
        if end > len(text) {
            end = len(text)
        }
        area := text[start:end]

        if matches := CardNumberRe.FindString(area); matches != "" {
            return matches
        }
    }

    slog.Debug("card number not found near card type indicators", "card_types", cardTypes)
    return fallback
}

// ApplyMapping looks up account name in mapping, returns original if not found.
func ApplyMapping(mapping map[string]string, name string) string {
    if mapping == nil {
        return name
    }
    if mapped, ok := mapping[name]; ok {
        return mapped
    }
    return name
}
```

### Provider changes

Each provider's `pdf.go` replaces its local `cardNumberRe` and `extractAccountName` with a call to `cardutil`:

| Provider | Before | After |
|---|---|---|
| `hsbccredit` | local `cardNumberRe`, `extractAccountName` finds "Card Number" marker | `cardutil.ExtractAfterMarker(text, "Card Number", "HSBC Credit Card")` |
| `hlbcredit` | local `cardNumberRe`, `extractAccountName` finds "Credit Card Number" marker | `cardutil.ExtractAfterMarker(text, "Credit Card Number", "HLB Credit Card")` |
| `uobcredit` | local `cardNumberRe`, `extractAccountName` finds card type indicators | `cardutil.ExtractNearCardType(text, []string{"WORLD MASTERCARD", "MASTERCARD", "VISA"}, "UOB Credit Card")` |

Each provider's `service.go` replaces the `accountMapping` lookup block with `cardutil.ApplyMapping(p.accountMapping, accountName)`.

### What stays per-provider

- `extractAccountName` function (thin wrapper calling cardutil)
- `accountMapping` field + `Reload()` (keeps interface contract)
- All provider-specific parsing logic (transaction regex, skip patterns, date handling)

### Files modified

| File | Change |
|---|---|
| `internal/providers/cardutil/cardutil.go` | **New** — shared helpers |
| `internal/providers/cardutil/cardutil_test.go` | **New** — tests for shared helpers |
| `internal/providers/hsbccredit/pdf.go` | Remove local `cardNumberRe`, simplify `extractAccountName` |
| `internal/providers/hsbccredit/service.go` | Replace accountMapping lookup with `cardutil.ApplyMapping` |
| `internal/providers/hlbcredit/pdf.go` | Remove local `cardNumberRe`, simplify `extractAccountName` |
| `internal/providers/hlbcredit/service.go` | Replace accountMapping lookup with `cardutil.ApplyMapping` |
| `internal/providers/uobcredit/pdf.go` | Remove local `cardNumberRe`, simplify `extractAccountName` |
| `internal/providers/uobcredit/service.go` | Replace accountMapping lookup with `cardutil.ApplyMapping` |

### Testing

- `cardutil_test.go` — unit tests for `ExtractAfterMarker`, `ExtractNearCardType`, `ApplyMapping`
- Existing provider tests unchanged (behavior preserved)
