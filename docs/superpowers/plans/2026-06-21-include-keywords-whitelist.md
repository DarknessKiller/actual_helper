# Include-Keywords as Whitelist Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When `include_keywords` are configured, `ShouldSkip` acts as a whitelist — only rows matching an include keyword pass through; everything else is skipped.

**Architecture:** Single function change in `rule.Engine.ShouldSkip` — if `includeKeywords` is non-empty, return `true` (skip) unless description matches any include keyword. When `includeKeywords` is empty/nil, fall back to existing exclude-only logic. Config merging in `internal/config/config.go:77-91` is unchanged (global + provider-specific keywords already merge into a flat list).

**Tech Stack:** Go, Ginkgo/Gomega

---

### Task 1: Update AGENTS.md docs

**Files:**
- Modify: `AGENTS.md:161-163`

- [ ] **Step 1: Update Filtering Rules section**

Replace the TNG-specific Filtering Rules with a generic description matching the new whitelist behavior.

Old:
```markdown
### Filtering Rules (TNG)

The provider's `shouldSkip()` checks `exclude_keywords` and `include_keywords` on each description. If any `include_keyword` matches, the row is kept (overrides excludes). If only `exclude_keywords` match, the row is skipped. No config and no keywords → nothing is filtered.
```

New:
```markdown
### Filtering Rules

Providers use `shouldSkip()` via `rule.Engine`. When `include_keywords` are configured (merged from global + provider config), they act as a **whitelist**: only rows matching any include keyword pass through; everything else is skipped. When `include_keywords` is empty, exclude-only filtering applies (matching `exclude_keywords` are skipped). No config and no keywords → nothing is filtered.
```

- [ ] **Step 2: Verify the change**

Run: `head -165 AGENTS.md | tail -5`
Expected: The new Filtering Rules text is shown.

---

### Task 2: Add whitelist tests (TDD — watch them fail first)

**Files:**
- Modify: `internal/rule/engine_test.go`
- Test: `internal/rule/engine_test.go`

- [ ] **Step 1: Add new ShouldSkip test cases**

Insert these test cases after the existing `It("returns false for nil keywords"...)` block (line 40) and before `Describe("MatchCategory"`):

```go
		It("include keyword whitelist keeps matching rows", func() {
			e := rule.NewEngine(nil, []string{"Grab"}, nil)
			Expect(e.ShouldSkip("GrabFood Order")).To(BeFalse())
		})

		It("include keyword whitelist skips non-matching rows", func() {
			e := rule.NewEngine(nil, []string{"Grab"}, nil)
			Expect(e.ShouldSkip("Shopee Order")).To(BeTrue())
		})

		It("include keyword overrides exclude in whitelist mode", func() {
			e := rule.NewEngine(
				[]string{"Grab"},
				[]string{"Grab"},
				nil,
			)
			Expect(e.ShouldSkip("GrabFood Order")).To(BeFalse())
		})

		It("include keyword skips non-matching even when exclude would match", func() {
			e := rule.NewEngine(
				[]string{"Shopee"},
				[]string{"Grab"},
				nil,
			)
			Expect(e.ShouldSkip("Shopee Order")).To(BeTrue())
		})

		It("empty include slice falls back to exclude logic", func() {
			e := rule.NewEngine([]string{"Grab"}, []string{}, nil)
			Expect(e.ShouldSkip("GrabFood Order")).To(BeTrue())
			Expect(e.ShouldSkip("Shopee Order")).To(BeFalse())
		})
```

The last test verifies that `[]string{}` (empty, not nil) correctly falls back, not triggering whitelist mode.

- [ ] **Step 2: Run tests to verify new ones fail**

Run: `go test ./internal/rule/... -v -count=1 2>&1 | grep -E "(PASS|FAIL|Error)" | head -20`

Expected: The 5 new tests fail — `ShouldSkip` still has old logic so `It("include keyword whitelist skips non-matching rows")` returns `false` instead of `true`, etc.

---

### Task 3: Implement ShouldSkip whitelist logic

**Files:**
- Modify: `internal/rule/engine.go:33-50`

- [ ] **Step 1: Modify ShouldSkip**

Replace the current `ShouldSkip` (lines 33-50):

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
		return true
	}

	for _, kw := range e.excludeKeywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run tests to verify all pass**

Run: `go test ./internal/rule/... -v -count=1`

Expected: All 15 specs pass (10 existing + 5 new).

---

### Task 4: Full test suite verification

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -count=1 2>&1 | tail -20`

Expected: All tests across all packages pass.

---

### Task 5: Commit

- [ ] **Step 1: Commit**

```bash
git add AGENTS.md internal/rule/engine.go internal/rule/engine_test.go
git commit -m "feat: include_keywords act as whitelist when configured"
```
