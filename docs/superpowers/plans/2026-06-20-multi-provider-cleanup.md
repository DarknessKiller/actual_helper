# Multi-Provider Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the codebase ready to add a second bank/fintech provider without modifying shared infrastructure code.

**Architecture:** Bootstrap accepts a map of provider factories keyed by name, iterates to register all providers. Generic filtering/categorization extracted to `internal/rule.Engine` with its own mutex — TNG provider embeds it instead of owning keyword fields directly. Handler test uses a mock instead of importing the real TNG provider.

**Tech Stack:** Go, Ginkgo/Gomega

## Global Constraints

- No new dependencies (standard library only)
- `internal/providers/provider.go` interfaces unchanged
- `TNGProvider.New()` constructor signature stays compatible with `ProviderFactory`
- All existing tests must pass

---

### Task 1: Gitignore & Remove Stale Files

**Files:**
- Modify: `.gitignore`
- Remove: `doc/openapi.json`
- Remove: `cmd/app/doc/openapi.json`

- [ ] **Step 1: Update `.gitignore`**

Append to `.gitignore`:
```
.vscode/
__debug_bin*
doc/openapi.json
cmd/app/doc/openapi.json
```

- [ ] **Step 2: Remove stale openapi files from git**

```bash
git rm doc/openapi.json cmd/app/doc/openapi.json
```

- [ ] **Step 3: Verify**

Check `git status` — modified `.gitignore`, deleted two openapi files, no untracked `.vscode/`.

- [ ] **Step 4: Commit**

```bash
git add .gitignore
git commit -m "chore: gitignore vscode, debug bins, generated openapi; remove stale openapi files"
```

---

### Task 2: Extract `internal/rule/` Package

**Files:**
- Create: `internal/rule/engine.go`
- Create: `internal/rule/engine_test.go`
- Create: `internal/rule/rule_suite_test.go`

**Interfaces:**
- Produces: `rule.Engine` struct with `NewEngine`, `Reload`, `ShouldSkip`, `MatchCategory`

- [ ] **Step 1: Create `internal/rule/rule_suite_test.go`**

```go
package rule_test

import (
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestRule(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Rule Suite")
}
```

- [ ] **Step 2: Create `internal/rule/engine.go`**

```go
package rule

import (
    "strings"
    "sync"

    "actual-helper/internal/models"
)

type Engine struct {
    excludeKeywords []string
    includeKeywords []string
    categories      []models.CategoryRule
    mu              sync.RWMutex
}

func NewEngine(excludeKeywords, includeKeywords []string, categories []models.CategoryRule) *Engine {
    return &Engine{
        excludeKeywords: copySlice(excludeKeywords),
        includeKeywords: copySlice(includeKeywords),
        categories:      copyCategories(categories),
    }
}

func (e *Engine) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule) {
    e.mu.Lock()
    defer e.mu.Unlock()
    e.excludeKeywords = copySlice(excludeKeywords)
    e.includeKeywords = copySlice(includeKeywords)
    e.categories = copyCategories(categories)
}

func (e *Engine) ShouldSkip(description string) bool {
    e.mu.RLock()
    defer e.mu.RUnlock()

    lower := strings.ToLower(description)

    for _, kw := range e.includeKeywords {
        if strings.Contains(lower, strings.ToLower(kw)) {
            return false
        }
    }
    for _, kw := range e.excludeKeywords {
        if strings.Contains(lower, strings.ToLower(kw)) {
            return true
        }
    }
    return false
}

func (e *Engine) MatchCategory(description string) (string, string) {
    e.mu.RLock()
    defer e.mu.RUnlock()

    lower := strings.ToLower(description)
    for _, r := range e.categories {
        if strings.Contains(lower, strings.ToLower(r.Keyword)) {
            return r.Group, r.Category
        }
    }
    return "", ""
}

func copySlice(s []string) []string {
    if s == nil {
        return nil
    }
    out := make([]string, len(s))
    copy(out, s)
    return out
}

func copyCategories(c []models.CategoryRule) []models.CategoryRule {
    if c == nil {
        return nil
    }
    out := make([]models.CategoryRule, len(c))
    copy(out, c)
    return out
}
```

- [ ] **Step 3: Create `internal/rule/engine_test.go`**

```go
package rule_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"

    "actual-helper/internal/models"
    "actual-helper/internal/rule"
)

var _ = Describe("Engine", func() {
    Describe("ShouldSkip", func() {
        It("returns true when exclude keyword matches", func() {
            e := rule.NewEngine([]string{"Quick Reload"}, nil, nil)
            Expect(e.ShouldSkip("Quick Reload Payment")).To(BeTrue())
        })

        It("returns false when no keyword matches", func() {
            e := rule.NewEngine([]string{"Quick Reload"}, nil, nil)
            Expect(e.ShouldSkip("GrabFood Order")).To(BeFalse())
        })

        It("include keyword overrides exclude", func() {
            e := rule.NewEngine(
                []string{"Daily Interest"},
                []string{"Daily Interest"},
                nil,
            )
            Expect(e.ShouldSkip("Daily Interest earned")).To(BeFalse())
        })

        It("matches case-insensitively", func() {
            e := rule.NewEngine([]string{"quick reload"}, nil, nil)
            Expect(e.ShouldSkip("QUICK RELOAD PAYMENT")).To(BeTrue())
        })

        It("returns false for nil keywords", func() {
            e := rule.NewEngine(nil, nil, nil)
            Expect(e.ShouldSkip("anything")).To(BeFalse())
        })
    })

    Describe("MatchCategory", func() {
        It("returns group and category on match", func() {
            e := rule.NewEngine(nil, nil, []models.CategoryRule{
                {Keyword: "grab", Group: "Food", Category: "Delivery"},
            })
            grp, cat := e.MatchCategory("GrabFood Order")
            Expect(grp).To(Equal("Food"))
            Expect(cat).To(Equal("Delivery"))
        })

        It("returns empty on no match", func() {
            e := rule.NewEngine(nil, nil, nil)
            grp, cat := e.MatchCategory("Unknown")
            Expect(grp).To(BeEmpty())
            Expect(cat).To(BeEmpty())
        })

        It("first match wins", func() {
            e := rule.NewEngine(nil, nil, []models.CategoryRule{
                {Keyword: "grab", Group: "Food", Category: "Delivery"},
                {Keyword: "grab", Group: "Override", Category: "ShouldNotReach"},
            })
            grp, cat := e.MatchCategory("GrabFood")
            Expect(grp).To(Equal("Food"))
            Expect(cat).To(Equal("Delivery"))
        })

        It("matches case-insensitively", func() {
            e := rule.NewEngine(nil, nil, []models.CategoryRule{
                {Keyword: "GRAB", Group: "Food", Category: "Delivery"},
            })
            grp, cat := e.MatchCategory("grabfood")
            Expect(grp).To(Equal("Food"))
            Expect(cat).To(Equal("Delivery"))
        })
    })

    Describe("Reload", func() {
        It("replaces keywords and categories", func() {
            e := rule.NewEngine([]string{"old"}, nil, nil)
            Expect(e.ShouldSkip("old")).To(BeTrue())

            e.Reload([]string{"new"}, nil, nil)
            Expect(e.ShouldSkip("old")).To(BeFalse())
            Expect(e.ShouldSkip("new")).To(BeTrue())
        })
    })
})
```

- [ ] **Step 4: Run engine tests to verify they pass**

```bash
go test ./internal/rule/... -v
```
Expected: 10 tests, all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/rule/
git commit -m "feat(rule): extract shared filtering and categorization engine"
```

---

### Task 3: TNG Provider Uses rule.Engine

**Files:**
- Modify: `internal/providers/tng/service.go`
- Modify: `internal/providers/tng/service_test.go`
- Modify: `internal/providers/tng/pdf_test.go`

**Interfaces:**
- Consumes: `rule.Engine` from Task 2
- Produces: `TNGProvider` with same public API, embedded `*rule.Engine`

- [ ] **Step 1: Update `service.go` — replace keyword fields with engine**

Current fields to remove:
```go
excludeKeywords []string
includeKeywords []string
categories      []models.CategoryRule
mu              sync.RWMutex
```

New field to add after `TNGProvider struct {`:
```go
engine *rule.Engine
```

Add import `"actual-helper/internal/rule"`.

Update `New`:
```go
func New(excludeKeywords, includeKeywords []string, categories []models.CategoryRule) *TNGProvider {
    return &TNGProvider{
        engine: rule.NewEngine(excludeKeywords, includeKeywords, categories),
    }
}
```

Update `Reload`:
```go
func (p *TNGProvider) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule) {
    p.engine.Reload(excludeKeywords, includeKeywords, categories)
}
```

Update `shouldSkip`:
```go
func (p *TNGProvider) shouldSkip(description string) bool {
    return p.engine.ShouldSkip(description)
}
```

Update `matchCategory`:
```go
func (p *TNGProvider) matchCategory(description string) (string, string) {
    return p.engine.MatchCategory(description)
}
```

Remove `copySlice` and `copyCategories` helper functions if they existed (they should have been removed when `New` was simplified — verify no unused functions).

- [ ] **Step 2: Remove duplicate keyword/category tests from `service_test.go`**

Keep these tests:
- "parses valid CSV rows"
- "skips non-success status rows"
- "skips filtered description rows when exclude keywords match" (integration — keep 1)
- "does not filter when no exclude keywords are set"
- "skips rows with insufficient columns"
- "returns empty for header-only CSV"
- "handles empty input"
- "returns negative amount for purchases"
- "handles DUITNOW_RECEIVEFROM as positive"
- "handles Refund as positive"
- "parses date in DD/MM/YYYY format"
- "applies categories from rules" (integration — keep 1)
- "returns tng"

Remove the `import "actual-helper/internal/models"` since the removed category tests used it (keep it if remaining tests still need it).

- [ ] **Step 3: Remove duplicate keyword/category tests from `pdf_test.go`**

Keep these tests:
- "parses a payment as debit (negative amount)"
- "parses a reload as credit (positive amount)"
- "parses DUITNOW_RECEIVEFROM as credit"
- "parses multiple transactions"
- "skips transactions with filtered description when exclude keywords match" (integration — keep 1)
- "returns error for text without transaction section"
- "returns empty for text with header but no transactions"
- "applies categories when provider has category rules" (integration — keep 1)
- "handles date with single-digit day"
- "handles date with double-digit day and month"
- "extracts description without trailing reference noise"
- "parses DuitNow QR transaction type"

Remove the `import "actual-helper/internal/models"` since the removed category tests used it (keep if remaining tests still need it).

- [ ] **Step 4: Run all tests to verify**

```bash
go test ./internal/providers/tng/... -v
```
Expected: tests pass. No data races.

- [ ] **Step 5: Commit**

```bash
git add internal/providers/tng/
git commit -m "refactor(tng): delegate filtering and categorization to rule.Engine"
```

---

### Task 4: Bootstrap Accepts Provider Factories

**Files:**
- Modify: `internal/bootstrap/bootstrap.go`
- Modify: `cmd/app/main.go`
- Test: no existing bootstrap tests, no new test needed (integration tested via handler tests)

**Interfaces:**
- Produces: `bootstrap.ProviderFactory` type, `bootstrap.Init(factories map[string]ProviderFactory)` signature
- Consumes: `config.NewLoader`, `providers.Registry`, provider constructors from main.go

- [ ] **Step 1: Rewrite `internal/bootstrap/bootstrap.go`**

```go
package bootstrap

import (
    "log/slog"
    "os"

    "actual-helper/internal/config"
    "actual-helper/internal/models"
    "actual-helper/internal/providers"
)

type ProviderFactory func(excludeKeywords, includeKeywords []string, categories []models.CategoryRule) providers.Provider

func Init(factories map[string]ProviderFactory) (*providers.Registry, *config.Loader) {
    configPath := os.Getenv("PROVIDER_CONFIG_PATH")
    loader := config.NewLoader(configPath)
    registry := providers.NewRegistry()

    if configPath == "" {
        slog.Warn("PROVIDER_CONFIG_PATH not set, running without filters or categories")
    }

    for name, factory := range factories {
        pc := loader.ProviderConfig(name)
        provider := factory(pc.ExcludeKeywords, pc.IncludeKeywords, pc.Categories)
        registry.Register(provider)
    }

    return registry, loader
}
```

- [ ] **Step 2: Update `cmd/app/main.go`**

```go
package main

import (
    "log"

    "actual-helper/internal/bootstrap"
    "actual-helper/internal/handlers"
    tngprov "actual-helper/internal/providers/tng"
    "actual-helper/internal/services"

    "github.com/go-fuego/fuego"
)

func main() {
    server := fuego.NewServer()

    registry, loader := bootstrap.Init(map[string]bootstrap.ProviderFactory{
        "tng": tngprov.New,
    })

    convertService := services.NewConvertService(registry, loader)
    handler := handlers.NewConvertHandler(convertService)
    handlers.RegisterConvertRoutes(server, handler)

    if err := server.Run(); err != nil {
        log.Fatal(err)
    }
}
```

- [ ] **Step 3: Build and run all tests**

```bash
go build ./...
go test ./... -v
```
Expected: all build and test clean.

- [ ] **Step 4: Commit**

```bash
git add internal/bootstrap/bootstrap.go cmd/app/main.go
git commit -m "feat(bootstrap): accept provider factory map instead of hardcoding TNG"
```

---

### Task 5: Handler Test Uses Mock Provider

**Files:**
- Modify: `internal/handlers/convert_test.go`

**Interfaces:**
- Consumes: `providers.Provider` interface
- Produces: mock provider implementation scoped to test file

- [ ] **Step 1: Rewrite `internal/handlers/convert_test.go`**

```go
package handlers_test

import (
    "bytes"
    "context"
    "io"
    "mime/multipart"
    "net/http"
    "net/http/httptest"

    "actual-helper/internal/handlers"
    "actual-helper/internal/models"
    "actual-helper/internal/providers"
    "actual-helper/internal/services"

    "github.com/go-fuego/fuego"
    "github.com/go-fuego/fuego/option"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

type mockProvider struct {
    name string
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) ParseCSV(_ context.Context, _ io.Reader) ([]models.ActualBudgetReport, error) {
    return []models.ActualBudgetReport{
        {Account: "Current", Date: "2026-06-13", Payee: "", Notes: "Top Up", Amount: "500.00"},
    }, nil
}
func (m *mockProvider) ParsePDFText(_ context.Context, _ string) ([]models.ActualBudgetReport, error) {
    return nil, nil
}

var _ = Describe("ConvertHandler", func() {
    Describe("via HTTP", func() {
        It("returns 400 when file is missing", func() {
            reg := providers.NewRegistry()
            reg.Register(&mockProvider{name: "test"})
            svc := services.NewConvertService(reg, nil)
            dummyHandler := handlers.NewConvertHandler(svc)

            c := fuego.NewServer()
            handlers.RegisterConvertRoutes(c, dummyHandler)

            req := httptest.NewRequest("POST", "/convert/test", nil)
            w := httptest.NewRecorder()
            c.Mux.ServeHTTP(w, req)

            Expect(w.Code).To(Equal(http.StatusBadRequest))
        })

        It("returns 500 for unregistered provider", func() {
            reg := providers.NewRegistry()
            svc := services.NewConvertService(reg, nil)
            dummyHandler := handlers.NewConvertHandler(svc)

            s := fuego.NewServer()
            fuego.Post(s, "/convert/{provider}", dummyHandler.Convert,
                option.Tags("convert"),
            )

            var buf bytes.Buffer
            w := multipart.NewWriter(&buf)
            fw, _ := w.CreateFormFile("file", "test.csv")
            fw.Write([]byte("a,b,c"))
            w.Close()

            req := httptest.NewRequest("POST", "/convert/unknown", &buf)
            req.Header.Set("Content-Type", w.FormDataContentType())
            rr := httptest.NewRecorder()
            s.Mux.ServeHTTP(rr, req)

            Expect(rr.Code).To(Equal(http.StatusInternalServerError))
        })

        It("returns CSV on successful conversion", func() {
            reg := providers.NewRegistry()
            reg.Register(&mockProvider{name: "test"})

            svc := services.NewConvertService(reg, nil)
            dummyHandler := handlers.NewConvertHandler(svc)

            s := fuego.NewServer()
            fuego.Post(s, "/convert/{provider}", dummyHandler.Convert,
                option.Tags("convert"),
            )

            var buf bytes.Buffer
            w := multipart.NewWriter(&buf)
            fw, _ := w.CreateFormFile("file", "test.csv")
            fw.Write([]byte("dummy,csv,data"))
            w.Close()

            req := httptest.NewRequest("POST", "/convert/test", &buf)
            req.Header.Set("Content-Type", w.FormDataContentType())
            rr := httptest.NewRecorder()
            s.Mux.ServeHTTP(rr, req)

            Expect(rr.Code).To(Equal(http.StatusOK))
            Expect(rr.Header().Get("Content-Type")).To(Equal("text/csv"))
            Expect(rr.Body.String()).To(ContainSubstring("Account,Date,Payee"))
            Expect(rr.Body.String()).To(ContainSubstring("Top Up"))
        })
    })
})
```

- [ ] **Step 2: Run handler tests**

```bash
go test ./internal/handlers/... -v
```
Expected: 3 tests, all PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/handlers/convert_test.go
git commit -m "test(handler): use mock provider instead of importing real TNG"
```

---

### Task 6: Clean Up Documentation

**Files:**
- Modify: `docs/superpowers/specs/2026-06-20-tng-categorization-design.md`
- Modify: `docs/superpowers/specs/2026-06-20-provider-config-design.md`
- Modify: `docs/superpowers/plans/2026-06-20-provider-config.md`

- [ ] **Step 1: Add SUPERSEDED notice to old categorization spec**

Prepend to `docs/superpowers/specs/2026-06-20-tng-categorization-design.md`:
```markdown
> **SUPERSEDED** — This design was replaced by the shared provider config system.
> See [provider-config-design.md](2026-06-20-provider-config-design.md) for the current approach.
>
```

- [ ] **Step 2: Add `internal/rule` entry to provider-config-design.md files table**

Find the Files table in `docs/superpowers/specs/2026-06-20-provider-config-design.md` and add a row:
```markdown
| `internal/rule/engine.go` | **New** — shared filtering/categorization engine |
```

- [ ] **Step 3: Mark all checkboxes complete in the plan file**

In `docs/superpowers/plans/2026-06-20-provider-config.md`, replace all `- [ ]` with `- [x]`.

Also add a note at the top:
```markdown
> **Status: COMPLETED** — All tasks implemented. See `2026-06-20-provider-config-design.md` for the design.
```

- [ ] **Step 4: Commit**

```bash
git add docs/
git commit -m "docs: mark categorization design superseded, add rule package to files table, mark plan complete"
```

---

## Verification

After all tasks, run full build + test suite:

```bash
go build ./...
go test ./... -v -count=1
go test -race ./internal/rule/... ./internal/providers/tng/...
```

All 50+ tests pass, no data races, build clean.
