> **Status: COMPLETED** — All tasks implemented. See `2026-06-20-provider-config-design.md` for the design.

# Provider Config System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Replace hardcoded TNG description filters and TNG-specific categories with a shared `internal/config` package that provides a single JSON config file with global + per-provider sections.

**Architecture:** New `internal/config` package with a `Loader` that handles JSON loading, hot-reload via mtime, description filtering (`ShouldSkip`), and category matching (`Match`). TNG provider accepts `*config.Loader` via constructor injection.

**Tech Stack:** Go, Fuego, Ginkgo/Gomega

## Global Constraints

- Provider-agnostic naming (no "tng" in package names or config keys other than provider identifiers)
- Backward compatible when `PROVIDER_CONFIG_PATH` is unset (no filtering, no categories)
- Hot-reload via mtime on each parse call
- Never fail a request on missing/invalid config
- Example template file in repo root

---

### Task 1: Create `internal/config` package — data types and JSON loading

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_suite_test.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Produces: `type Config struct`, `type Loader struct`, `func NewLoader(path string) *Loader`, `func (l *Loader) ensureLoaded()`

- [x] **Step 1: Create directory and test suite file**

```go
// internal/config/config_suite_test.go
package config_test

import (
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Config Suite")
}
```

- [x] **Step 2: Create `internal/config/config.go` with data types**

```go
package config

import (
    "encoding/json"
    "log/slog"
    "os"
    "strings"
    "sync"
    "time"
)

type CategoryRule struct {
    Keyword  string `json:"keyword"`
    Group    string `json:"group"`
    Category string `json:"category"`
}

type ProviderConfig struct {
    ExcludeKeywords []string       `json:"exclude_keywords"`
    IncludeKeywords []string       `json:"include_keywords"`
    Categories      []CategoryRule `json:"categories"`
}

type Config struct {
    Global    ProviderConfig            `json:"global"`
    Providers map[string]ProviderConfig `json:"providers"`
}

type Loader struct {
    path       string
    lastLoaded time.Time
    config     Config
    mu         sync.Mutex
}

func NewLoader(path string) *Loader {
    return &Loader{path: path}
}

func (l *Loader) ensureLoaded() {
    l.mu.Lock()
    defer l.mu.Unlock()

    if l.path == "" {
        return
    }

    info, err := os.Stat(l.path)
    if err != nil {
        l.config = Config{}
        slog.Warn("config file not accessible", "path", l.path, "error", err)
        return
    }

    if !info.ModTime().After(l.lastLoaded) && l.lastLoaded != (time.Time{}) {
        return
    }

    data, err := os.ReadFile(l.path)
    if err != nil {
        slog.Warn("config file unreadable", "path", l.path, "error", err)
        return
    }

    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        slog.Warn("config file invalid JSON", "path", l.path, "error", err)
        return
    }

    l.config = cfg
    l.lastLoaded = info.ModTime()
}

type mergeConfig struct {
    excludeKeywords []string
    includeKeywords []string
    categories      []CategoryRule
}

func (l *Loader) getMerged(providerName string) mergeConfig {
    l.ensureLoaded()

    var mc mergeConfig
    mc.excludeKeywords = append(mc.excludeKeywords, l.config.Global.ExcludeKeywords...)
    mc.includeKeywords = append(mc.includeKeywords, l.config.Global.IncludeKeywords...)
    mc.categories = append(mc.categories, l.config.Global.Categories...)

    if p, ok := l.config.Providers[providerName]; ok {
        mc.excludeKeywords = append(mc.excludeKeywords, p.ExcludeKeywords...)
        mc.includeKeywords = append(mc.includeKeywords, p.IncludeKeywords...)
        mc.categories = append(p.Categories, mc.categories...)
    }

    return mc
}

func (l *Loader) ShouldSkip(ctx context.Context, logger *slog.Logger, providerName, description string) bool {
    mc := l.getMerged(providerName)

    lower := strings.ToLower(description)

    for _, kw := range mc.includeKeywords {
        if strings.Contains(lower, strings.ToLower(kw)) {
            return false
        }
    }

    for _, kw := range mc.excludeKeywords {
        if strings.Contains(lower, strings.ToLower(kw)) {
            return true
        }
    }

    return false
}

func (l *Loader) Match(providerName, description string) (group, category string) {
    mc := l.getMerged(providerName)
    lower := strings.ToLower(description)

    for _, r := range mc.categories {
        if strings.Contains(lower, strings.ToLower(r.Keyword)) {
            return r.Group, r.Category
        }
    }

    return "", ""
}
```

- [x] **Step 3: Write failing tests**

```go
// internal/config/config_test.go
package config_test

import (
    "context"
    "log/slog"
    "os"
    "path/filepath"
    "time"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "github.com/ItsGoYoung/ActualHelperGo/internal/config"
)

var _ = Describe("Config", func() {
    var (
        tmpDir string
        logger *slog.Logger
        ctx    context.Context
    )

    BeforeEach(func() {
        var err error
        tmpDir, err = os.MkdirTemp("", "config-test-*")
        Expect(err).NotTo(HaveOccurred())
        logger = slog.New(slog.DiscardHandler)
        ctx = context.Background()
    })

    AfterEach(func() {
        os.RemoveAll(tmpDir)
    })

    Describe("loading", func() {
        It("loads valid JSON with global and provider sections", func() {
            path := filepath.Join(tmpDir, "config.json")
            content := `{
                "global": {
                    "exclude_keywords": ["Common Noise"],
                    "include_keywords": [],
                    "categories": [{"keyword": "shopee", "group": "Shopping", "category": "Online"}]
                },
                "providers": {
                    "tng": {
                        "exclude_keywords": ["Quick Reload"],
                        "include_keywords": ["Daily Interest"],
                        "categories": [{"keyword": "grab", "group": "Food", "category": "Delivery"}]
                    }
                }
            }`
            err := os.WriteFile(path, []byte(content), 0644)
            Expect(err).NotTo(HaveOccurred())

            loader := config.NewLoader(path)
            Expect(loader.ShouldSkip(ctx, logger, "tng", "Quick Reload Payment")).To(BeTrue())
        })

        It("returns defaults when config file is missing", func() {
            loader := config.NewLoader(filepath.Join(tmpDir, "nonexistent.json"))
            Expect(loader.ShouldSkip(ctx, logger, "tng", "anything")).To(BeFalse())
        })

        It("returns defaults when env var is unset", func() {
            loader := config.NewLoader("")
            Expect(loader.ShouldSkip(ctx, logger, "tng", "anything")).To(BeFalse())
        })

        It("returns defaults on invalid JSON", func() {
            path := filepath.Join(tmpDir, "config.json")
            os.WriteFile(path, []byte("{invalid}"), 0644)
            loader := config.NewLoader(path)
            Expect(loader.ShouldSkip(ctx, logger, "tng", "anything")).To(BeFalse())
        })
    })

    Describe("ShouldSkip", func() {
        It("matches exclude keywords", func() {
            path := filepath.Join(tmpDir, "config.json")
            content := `{"providers":{"tng":{"exclude_keywords":["Quick Reload"]}}}`
            os.WriteFile(path, []byte(content), 0644)
            loader := config.NewLoader(path)
            Expect(loader.ShouldSkip(ctx, logger, "tng", "Quick Reload Payment")).To(BeTrue())
        })

        It("does not skip non-matching descriptions", func() {
            path := filepath.Join(tmpDir, "config.json")
            content := `{"providers":{"tng":{"exclude_keywords":["Quick Reload"]}}}`
            os.WriteFile(path, []byte(content), 0644)
            loader := config.NewLoader(path)
            Expect(loader.ShouldSkip(ctx, logger, "tng", "GrabFood Order")).To(BeFalse())
        })

        It("include overrides exclude", func() {
            path := filepath.Join(tmpDir, "config.json")
            content := `{"providers":{"tng":{"exclude_keywords":["Daily Interest"],"include_keywords":["Daily Interest"]}}}`
            os.WriteFile(path, []byte(content), 0644)
            loader := config.NewLoader(path)
            Expect(loader.ShouldSkip(ctx, logger, "tng", "Daily Interest earned")).To(BeFalse())
        })

        It("global exclude matches", func() {
            path := filepath.Join(tmpDir, "config.json")
            content := `{"global":{"exclude_keywords":["Common Noise"]}}`
            os.WriteFile(path, []byte(content), 0644)
            loader := config.NewLoader(path)
            Expect(loader.ShouldSkip(ctx, logger, "tng", "Common Noise transaction")).To(BeTrue())
        })

        It("matches case-insensitively", func() {
            path := filepath.Join(tmpDir, "config.json")
            content := `{"providers":{"tng":{"exclude_keywords":["quick reload"]}}}`
            os.WriteFile(path, []byte(content), 0644)
            loader := config.NewLoader(path)
            Expect(loader.ShouldSkip(ctx, logger, "tng", "QUICK RELOAD PAYMENT")).To(BeTrue())
        })
    })

    Describe("Match", func() {
        It("returns category from provider section first", func() {
            path := filepath.Join(tmpDir, "config.json")
            content := `{
                "global": {"categories": [{"keyword": "grab", "group": "Global", "category": "Global"}]},
                "providers": {"tng": {"categories": [{"keyword": "grab", "group": "Food", "category": "Delivery"}]}}
            }`
            os.WriteFile(path, []byte(content), 0644)
            loader := config.NewLoader(path)
            group, cat := loader.Match("tng", "GrabFood Order")
            Expect(group).To(Equal("Food"))
            Expect(cat).To(Equal("Delivery"))
        })

        It("falls back to global categories", func() {
            path := filepath.Join(tmpDir, "config.json")
            content := `{"global":{"categories":[{"keyword":"shopee","group":"Shopping","category":"Online"}]}}`
            os.WriteFile(path, []byte(content), 0644)
            loader := config.NewLoader(path)
            group, cat := loader.Match("tng", "Shopee Purchase")
            Expect(group).To(Equal("Shopping"))
            Expect(cat).To(Equal("Online"))
        })

        It("returns empty when no match", func() {
            path := filepath.Join(tmpDir, "config.json")
            os.WriteFile(path, []byte("{}"), 0644)
            loader := config.NewLoader(path)
            group, cat := loader.Match("tng", "Unknown Merchant")
            Expect(group).To(BeEmpty())
            Expect(cat).To(BeEmpty())
        })

        It("matches case-insensitively", func() {
            path := filepath.Join(tmpDir, "config.json")
            content := `{"providers":{"tng":{"categories":[{"keyword":"GRAB","group":"Food","category":"Delivery"}]}}}`
            os.WriteFile(path, []byte(content), 0644)
            loader := config.NewLoader(path)
            group, cat := loader.Match("tng", "grabfood order")
            Expect(group).To(Equal("Food"))
            Expect(cat).To(Equal("Delivery"))
        })
    })

    Describe("hot reload", func() {
        It("reloads when file mtime changes", func() {
            path := filepath.Join(tmpDir, "config.json")
            os.WriteFile(path, []byte(`{"providers":{"tng":{"exclude_keywords":["old"]}}}`), 0644)
            loader := config.NewLoader(path)

            Expect(loader.ShouldSkip(ctx, logger, "tng", "old transaction")).To(BeTrue())

            time.Sleep(10 * time.Millisecond) // ensure different mtime
            os.WriteFile(path, []byte(`{"providers":{"tng":{"exclude_keywords":["new"]}}}`), 0644)

            Expect(loader.ShouldSkip(ctx, logger, "tng", "old transaction")).To(BeFalse())
            Expect(loader.ShouldSkip(ctx, logger, "tng", "new transaction")).To(BeTrue())
        })
    })

    Describe("global-only config", func() {
        It("applies global rules when no provider section exists", func() {
            path := filepath.Join(tmpDir, "config.json")
            content := `{"global":{"exclude_keywords":["Noise"],"categories":[{"keyword":"shop","group":"Shopping","category":"General"}]}}`
            os.WriteFile(path, []byte(content), 0644)
            loader := config.NewLoader(path)

            Expect(loader.ShouldSkip(ctx, logger, "tng", "Noise transaction")).To(BeTrue())
            group, cat := loader.Match("tng", "Shop Now")
            Expect(group).To(Equal("Shopping"))
            Expect(cat).To(Equal("General"))
        })
    })

    Describe("provider-only config", func() {
        It("applies provider rules when no global section exists", func() {
            path := filepath.Join(tmpDir, "config.json")
            content := `{"providers":{"tng":{"exclude_keywords":["TNG Fee"]}}}`
            os.WriteFile(path, []byte(content), 0644)
            loader := config.NewLoader(path)

            Expect(loader.ShouldSkip(ctx, logger, "tng", "TNG Fee charge")).To(BeTrue())
            Expect(loader.ShouldSkip(ctx, logger, "other", "TNG Fee charge")).To(BeFalse())
        })
    })
})
```

- [x] **Step 4: Run tests**

Run: `cd $WORKDIR && go test ./internal/config/ -v`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go internal/config/config_suite_test.go
git commit -m "feat(config): add shared provider config with Loader, ShouldSkip, and Match"
```

---

### Task 2: Update TNG provider to use `config.Loader`

**Files:**
- Modify: `internal/providers/tng/service.go`
- Modify: `internal/providers/tng/service_test.go`
- Modify: `internal/providers/tng/pdf_test.go`
- Remove: `internal/providers/tng/categories.go`
- Remove: `internal/providers/tng/categories_test.go`

- [x] **Step 1: Read current service.go**

- [x] **Step 2: Update TNGProvider struct and constructor**

- [x] **Step 3: Replace hardcoded filters with loader.ShouldSkip**

- [x] **Step 4: Replace categories logic with loader.Match**

- [x] **Step 5: Remove categories.go and categories_test.go**

- [x] **Step 6: Update tests (constructor calls)**

- [x] **Step 7: Run all tests**

- [x] **Step 8: Commit**

---

### Task 3: Wire into main.go and add example config

**Files:**
- Modify: `cmd/app/main.go`
- Create: `provider_config.example.json`

- [x] **Step 1: Read current main.go**

- [x] **Step 2: Update main.go with config.Loader**

- [x] **Step 3: Create example config**

- [x] **Step 4: Build and test**

- [x] **Step 5: Commit**
