# Version Management & Fuego Config Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract Fuego server config to `internal/config/`, manage version via git tags with ldflags injection, and automate CI + release with GitHub Actions.

**Architecture:** One var + one factory function in `internal/config/` package. Thin `main.go`. GitHub Actions runs CI on push/PR, and creates Docker image + release on tag push.

**Tech Stack:** Go, Fuego, Docker, GitHub Actions

## Global Constraints

- All new files in `internal/config/` package
- No new external dependencies
- Version defaults to `"dev"` locally when not built with ldflags
- GitHub Actions uses `ubuntu-latest`
- Docker build uses the existing `Dockerfile`

---

### Task 1: Create `internal/config/version.go` and `internal/config/server.go`

**Files:**
- Create: `internal/config/version.go`
- Create: `internal/config/server.go`

- [ ] **Step 1: Create `internal/config/version.go`**

```go
package config

var Version = "dev"
```

- [ ] **Step 2: Create `internal/config/server.go`**

```go
package config

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
)

func NewFuegoServer(env Env) *fuego.Server {
	server := fuego.NewServer(
		fuego.WithEngineOptions(
			fuego.WithOpenAPIConfig(fuego.OpenAPIConfig{
				Info: &openapi3.Info{
					Title:       "Actual Helper",
					Description: "Converts bank/fintech transaction files (CSV or PDF) into Actual Budget-compatible CSV format.",
					Version:     Version,
				},
				DisableDefaultServer: env.Environment == "production",
			}),
		),
		fuego.WithAddr(fmt.Sprintf("0.0.0.0:%d", env.Port)),
	)

	if env.Environment == "production" {
		server.OpenAPI.Description().Servers = []*openapi3.Server{
			{
				URL:         env.PublicURL,
				Description: "Production server",
			},
		}
	}

	return server
}
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./cmd/app
```
Expected: clean compile.

- [ ] **Step 4: Run vet and tests**

```bash
go vet ./...
go test ./...
```
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/config/version.go internal/config/server.go
git commit -m "refactor(config): extract Version var and NewFuegoServer factory"
```

---

### Task 2: Update `cmd/app/main.go` to use `config.NewFuegoServer`

**Files:**
- Modify: `cmd/app/main.go`

- [ ] **Step 1: Modify `cmd/app/main.go`**

Replace inline server creation with `config.NewFuegoServer(env)`.

Old imports:
```go
import (
	"fmt"
	"log"

	"actual-helper/internal/bootstrap"
	"actual-helper/internal/handlers"
	rytprov "actual-helper/internal/providers/ryt"
	tngprov "actual-helper/internal/providers/tng"
	"actual-helper/internal/ratelimit"
	"actual-helper/internal/services"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-fuego/fuego"
)
```

New imports:
```go
import (
	"log"

	"actual-helper/internal/bootstrap"
	"actual-helper/internal/config"
	"actual-helper/internal/handlers"
	rytprov "actual-helper/internal/providers/ryt"
	tngprov "actual-helper/internal/providers/tng"
	"actual-helper/internal/ratelimit"
	"actual-helper/internal/services"

	"github.com/go-fuego/fuego"
)
```

Old function body (everything inside `func main()`):
```go
func main() {

	registry, loader, env := bootstrap.Init(map[string]bootstrap.ProviderFactory{
		"tng": tngprov.New,
		"ryt": rytprov.New,
	})

	server := fuego.NewServer(
		fuego.WithEngineOptions(
			fuego.WithOpenAPIConfig(fuego.OpenAPIConfig{
				Info: &openapi3.Info{
					Title:       "Actual Helper",
					Description: "Converts bank/fintech transaction files (CSV or PDF) into Actual Budget-compatible CSV format.",
					Version:     "1.0.0",
				},
				DisableDefaultServer: env.Environment == "production",
			}),
		),
		fuego.WithAddr(fmt.Sprintf("0.0.0.0:%d", env.Port)),
	)

	if env.Environment == "production" {
		server.OpenAPI.Description().Servers = []*openapi3.Server{
			{
				URL:         env.PublicURL,
				Description: "Production server",
			},
		}
	}

	convertService := services.NewConvertService(registry, loader)
	handler := handlers.NewConvertHandler(convertService)
	fuego.Use(server, ratelimit.Middleware)
	handlers.RegisterConvertRoutes(server, handler)

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
```

New function body:
```go
func main() {
	registry, loader, env := bootstrap.Init(map[string]bootstrap.ProviderFactory{
		"tng": tngprov.New,
		"ryt": rytprov.New,
	})

	server := config.NewFuegoServer(env)

	convertService := services.NewConvertService(registry, loader)
	handler := handlers.NewConvertHandler(convertService)
	fuego.Use(server, ratelimit.Middleware)
	handlers.RegisterConvertRoutes(server, handler)

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./cmd/app
```
Expected: clean compile.

- [ ] **Step 3: Run vet and tests**

```bash
go vet ./...
go test ./...
```
Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add cmd/app/main.go
git commit -m "refactor(main): use config.NewFuegoServer"
```

---

### Task 3: Update Dockerfile with ldflags version injection

**Files:**
- Modify: `Dockerfile`

- [ ] **Step 1: Modify `Dockerfile`**

Replace the build RUN line:

Old:
```dockerfile
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o actual_helper ./cmd/app
```

New:
```dockerfile
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X actual-helper/internal/config.Version=$(git describe --tags --always --dirty)" \
    -o actual_helper ./cmd/app
```

- [ ] **Step 2: Commit**

```bash
git add Dockerfile
git commit -m "build(Dockerfile): inject version via ldflags from git describe"
```

---

### Task 4: Create GitHub Actions CI + Release workflow

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Create `.github/workflows/ci.yml`**

Two jobs:
1. **CI** — runs on push/PR to main: test, vet, build
2. **Release** — runs on tag push (`v*`): build Docker image, create GitHub Release

```yaml
name: CI

on:
  push:
    branches: [main]
    tags: ['v*']
  pull_request:
    branches: [main]

permissions:
  contents: write
  packages: write

jobs:
  ci:
    if: github.event_name == 'push' || github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
          cache: true

      - name: Vet
        run: go vet ./...

      - name: Test
        run: go test ./...

      - name: Build
        run: go build -trimpath -ldflags="-s -w -X actual-helper/internal/config.Version=$(git describe --tags --always --dirty)" ./cmd/app

  release:
    if: startsWith(github.ref, 'refs/tags/v')
    runs-on: ubuntu-latest
    needs: [ci]
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
          cache: true

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract version from tag
        id: version
        run: echo "VERSION=${GITHUB_REF#refs/tags/v}" >> $GITHUB_OUTPUT

      - name: Build and push Docker image
        uses: docker/build-push-action@v6
        with:
          context: .
          push: true
          tags: |
            ghcr.io/${{ github.repository }}:${{ steps.version.outputs.VERSION }}
            ghcr.io/${{ github.repository }}:latest
          build-args: |
            VERSION=${{ steps.version.outputs.VERSION }}

      - name: Generate release notes
        run: |
          echo "## Release v${{ steps.version.outputs.VERSION }}" > ${{ runner.temp }}/release-notes.md
          echo "" >> ${{ runner.temp }}/release-notes.md
          echo "Docker image: \`ghcr.io/${{ github.repository }}:${{ steps.version.outputs.VERSION }}\`" >> ${{ runner.temp }}/release-notes.md

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          tag_name: ${{ github.ref_name }}
          body_path: ${{ runner.temp }}/release-notes.md
          generate_release_notes: true
```

- [ ] **Step 2: Update Dockerfile to accept VERSION build arg**

Add ARG at the top of the build stage (before `RUN go mod download`):

```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
ARG VERSION
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X actual-helper/internal/config.Version=${VERSION:-$(git describe --tags --always --dirty)}" \
    -o actual_helper ./cmd/app
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml Dockerfile
git commit -m "ci: add GitHub Actions workflow with CI and release on tag"
```

---

### Task 5: Final verification

- [ ] **Step 1: Full build and test**

```bash
go build ./cmd/app
go vet ./...
go test ./...
```
Expected: all clean.

- [ ] **Step 2: Push first tag to test the workflow (after merge)**

```bash
git tag v0.1.0
git push origin v0.1.0
```
