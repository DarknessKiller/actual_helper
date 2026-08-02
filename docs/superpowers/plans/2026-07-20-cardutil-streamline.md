# Cardutil Streamline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract duplicated card number regex, account name extraction, and account mapping logic into a shared `cardutil` package, reducing duplication across `hsbccredit`, `hlb`, and `uobcredit` providers.

**Architecture:** New `internal/providers/cardutil` package with three helpers: `ExtractAfterMarker`, `ExtractNearCardType`, `ApplyMapping`. Each provider's `pdf.go` delegates to cardutil instead of duplicating regex and extraction logic. Each provider's `service.go` uses `cardutil.ApplyMapping` instead of inline mapping lookup.

**Tech Stack:** Go, regexp, existing `dateutil.Truncate` for debug logging.

## Global Constraints

- Follow existing project conventions (Handler → Service → Provider architecture)
- Preserve existing behavior — no functional changes
- Use `testify/require` for new tests
- Fake/anonymized test data only
- Conventional Commits format

---

## File Structure

| File | Action | Responsibility |
|---|---|---|
| `internal/providers/cardutil/cardutil.go` | Create | Shared card number helpers |
| `internal/providers/cardutil/cardutil_test.go` | Create | Tests for shared helpers |
| `internal/providers/hsbccredit/pdf.go` | Modify | Remove local `cardNumberRe`, simplify `extractAccountName` |
| `internal/providers/hsbccredit/service.go` | Modify | Replace mapping lookup with `cardutil.ApplyMapping` |
| `internal/providers/hlb/service.go` | Modify | Replace mapping lookup with `cardutil.ApplyMapping`; use `cardutil.WhitespaceRe` |
| `internal/providers/uobcredit/pdf.go` | Modify | Remove local `cardNumberRe`, simplify `extractAccountName` |
| `internal/providers/uobcredit/pdf.go` | Modify | Remove local `cardNumberRe`, simplify `extractAccountName` |
| `internal/providers/uobcredit/service.go` | Modify | Replace mapping lookup with `cardutil.ApplyMapping` |

---

### Task 1: Create cardutil package with helpers

**Files:**
- Create: `internal/providers/cardutil/cardutil.go`
- Create: `internal/providers/cardutil/cardutil_test.go`

**Interfaces:**
- Produces: `cardutil.CardNumberRe`, `cardutil.ExtractAfterMarker`, `cardutil.ExtractNearCardType`, `cardutil.ApplyMapping`

- [ ] **Step 1: Create cardutil.go**

```go
package cardutil

import (
	"log/slog"
	"regexp"
	"strings"

	"actual_helper/internal/dateutil"
)

// CardNumberRe matches card numbers: 4 groups of 4 digits separated by spaces or dashes.
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

- [ ] **Step 2: Create cardutil_test.go**

```go
package cardutil_test

import (
	"testing"

	"actual_helper/internal/providers/cardutil"

	"github.com/stretchr/testify/require"
)

func TestExtractAfterMarker(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		marker   string
		fallback string
		want     string
	}{
		{
			name:     "finds card number after marker",
			text:     "Card Number\n1234-5678-9012-3456",
			marker:   "Card Number",
			fallback: "Fallback",
			want:     "1234-5678-9012-3456",
		},
		{
			name:     "returns fallback when marker not found",
			text:     "No card info here",
			marker:   "Card Number",
			fallback: "Fallback",
			want:     "Fallback",
		},
		{
			name:     "returns fallback when no card number after marker",
			text:     "Card Number\nno digits here",
			marker:   "Card Number",
			fallback: "Fallback",
			want:     "Fallback",
		},
		{
			name:     "handles card number with spaces",
			text:     "Credit Card Number 1234 5678 9012 3456",
			marker:   "Credit Card Number",
			fallback: "Fallback",
			want:     "1234 5678 9012 3456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cardutil.ExtractAfterMarker(tt.text, tt.marker, tt.fallback)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestExtractNearCardType(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		cardTypes []string
		fallback  string
		want      string
	}{
		{
			name:      "finds card number near WORLD MASTERCARD",
			text:      "some text WORLD MASTERCARD 1234-5678-9012-3456 more text",
			cardTypes: []string{"WORLD MASTERCARD", "MASTERCARD", "VISA"},
			fallback:  "Fallback",
			want:      "1234-5678-9012-3456",
		},
		{
			name:      "finds card number near VISA",
			text:      "VISA ending 1234567890123456",
			cardTypes: []string{"WORLD MASTERCARD", "MASTERCARD", "VISA"},
			fallback:  "Fallback",
			want:      "1234 5678 9012 3456",
		},
		{
			name:      "returns fallback when no card type found",
			text:      "no card info here",
			cardTypes: []string{"WORLD MASTERCARD", "MASTERCARD", "VISA"},
			fallback:  "Fallback",
			want:      "Fallback",
		},
		{
			name:      "returns fallback when card type found but no card number nearby",
			text:      "WORLD MASTERCARD no digits",
			cardTypes: []string{"WORLD MASTERCARD"},
			fallback:  "Fallback",
			want:      "Fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cardutil.ExtractNearCardType(tt.text, tt.cardTypes, tt.fallback)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestApplyMapping(t *testing.T) {
	tests := []struct {
		name    string
		mapping map[string]string
		input   string
		want    string
	}{
		{
			name:    "maps existing key",
			mapping: map[string]string{"1234-5678-9012-3456": "My Card"},
			input:   "1234-5678-9012-3456",
			want:    "My Card",
		},
		{
			name:    "returns original when key not found",
			mapping: map[string]string{"1234-5678-9012-3456": "My Card"},
			input:   "9999-8888-7777-6666",
			want:    "9999-8888-7777-6666",
		},
		{
			name:    "returns original when mapping is nil",
			mapping: nil,
			input:   "1234-5678-9012-3456",
			want:    "1234-5678-9012-3456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cardutil.ApplyMapping(tt.mapping, tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}
```

- [ ] **Step 3: Run tests**

Run: `ginkgo run ./internal/providers/cardutil/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/providers/cardutil/
git commit -m "feat(cardutil): extract shared card number helpers"
```

---

### Task 2: Update hsbccredit to use cardutil

**Files:**
- Modify: `internal/providers/hsbccredit/pdf.go`
- Modify: `internal/providers/hsbccredit/service.go`

**Interfaces:**
- Consumes: `cardutil.ExtractAfterMarker`, `cardutil.ApplyMapping`

- [ ] **Step 1: Simplify extractAccountName in pdf.go**

Remove the local `cardNumberRe` regex (line 24) and replace `extractAccountName` with:

```go
func extractAccountName(text string) string {
	return cardutil.ExtractAfterMarker(text, "Card Number", "HSBC Credit Card")
}
```

Add import: `"actual_helper/internal/providers/cardutil"`

Remove import: `"actual_helper/internal/dateutil"` (no longer used directly).

- [ ] **Step 2: Replace mapping lookup in service.go**

In `toActualReports`, replace lines 80-86:

```go
p.mu.RLock()
if p.accountMapping != nil {
    if mapped, ok := p.accountMapping[accountName]; ok {
        accountName = mapped
    }
}
p.mu.RUnlock()
```

With:

```go
p.mu.RLock()
accountName = cardutil.ApplyMapping(p.accountMapping, accountName)
p.mu.RUnlock()
```

Add import: `"actual_helper/internal/providers/cardutil"`

- [ ] **Step 3: Run tests**

Run: `ginkgo run ./internal/providers/hsbccredit/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/providers/hsbccredit/
git commit -m "refactor(hsbccredit): use cardutil for card number extraction and mapping"
```

---

### Task 3: Update hlb to use cardutil

**Files:**
- Modify: `internal/providers/hlb/service.go`

**Interfaces:**
- Consumes: `cardutil.ExtractAfterMarker`, `cardutil.ApplyMapping`

- [ ] **Step 1: Replace mapping lookup in service.go**

In `toActualReports`, replace the inline accountMapping lookup with:

```go
p.mu.RLock()
accountName = cardutil.ApplyMapping(p.accountMapping, accountName)
p.mu.RUnlock()
```

Add import: `"actual_helper/internal/providers/cardutil"`

In `parseCreditPDF`, replace inline card number extraction with:

```go
accountName := cardutil.ExtractAfterMarker(text, "Credit Card Number", "HLB Credit Card")
```

- [ ] **Step 2: Replace whitespace normalization**

In `toActualReports`, replace `whitespacePattern.ReplaceAllString` with `cardutil.WhitespaceRe.ReplaceAllString`.

- [ ] **Step 3: Run tests**

Run: `ginkgo run ./internal/providers/hlb/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/providers/hlb/
git commit -m "refactor(hlb): use cardutil for card number extraction, mapping, and whitespace"
```

---

### Task 4: Update uobcredit to use cardutil

**Files:**
- Modify: `internal/providers/uobcredit/pdf.go`
- Modify: `internal/providers/uobcredit/service.go`

**Interfaces:**
- Consumes: `cardutil.ExtractNearCardType`, `cardutil.ApplyMapping`

- [ ] **Step 1: Simplify extractAccountName in pdf.go**

Remove the local `cardNumberRe` regex (line 20) and replace `extractAccountName` with:

```go
func extractAccountName(text string) string {
	return cardutil.ExtractNearCardType(text, []string{"WORLD MASTERCARD", "MASTERCARD", "VISA"}, "UOB Credit Card")
}
```

Add import: `"actual_helper/internal/providers/cardutil"`

Remove import: `"actual_helper/internal/dateutil"` (no longer used directly).

- [ ] **Step 2: Replace mapping lookup in service.go**

In `toActualReports`, replace lines 79-85:

```go
p.mu.RLock()
if p.accountMapping != nil {
    if mapped, ok := p.accountMapping[accountName]; ok {
        accountName = mapped
    }
}
p.mu.RUnlock()
```

With:

```go
p.mu.RLock()
accountName = cardutil.ApplyMapping(p.accountMapping, accountName)
p.mu.RUnlock()
```

Add import: `"actual_helper/internal/providers/cardutil"`

- [ ] **Step 3: Run tests**

Run: `ginkgo run ./internal/providers/uobcredit/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/providers/uobcredit/
git commit -m "refactor(uobcredit): use cardutil for card number extraction and mapping"
```

---

### Task 5: Run full test suite and verify

**Files:** None (verification only)

- [ ] **Step 1: Run all tests**

Run: `ginkgo run ./...`
Expected: PASS

- [ ] **Step 2: Verify no regressions**

Check that all provider tests still pass. Confirm no local `cardNumberRe` remains in hsbccredit, hlb (uses `cardutil.ExtractAfterMarker`), or uobcredit `pdf.go`.

- [ ] **Step 3: Final commit if needed**

If any fixes were needed, commit them.
