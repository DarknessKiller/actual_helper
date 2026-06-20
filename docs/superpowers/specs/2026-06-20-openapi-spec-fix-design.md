# OpenAPI Spec Fix for Provider Convert API

## Problem

The generated OpenAPI spec at `cmd/app/doc/openapi.json` has empty `paths` because:

1. The handler uses `fuego.ContextNoBody` — Fuego infers no request body, so no schema generated
2. The handler returns `(any, error)` and writes CSV directly to the response writer — Fuego cannot infer a structured response
3. `fuego.NewServer()` is called with no OpenAPI config — title/version/description are default boilerplate

## Solution

Keep the existing runtime behavior (manual multipart parsing, direct CSV writing). Add OpenAPI metadata via Fuego's option functions so the generated spec accurately describes the API.

### Changes

#### 1. `internal/handlers/convert.go`

- Define `ConvertRequestBody` struct with `json` and `validate` tags for OpenAPI schema generation
- Define `ConvertResponseBody` empty struct for CSV binary response type
- Add `option.RequestBody` with `multipart/form-data` content type
- Add `option.AddResponse(200, ...)` with `text/csv` content type

#### 2. `cmd/app/main.go`

- Pass `fuego.WithOpenAPIConfig` with proper `Info` (title: "Actual Helper API", version: "1.0.0")

### Generated Spec

The spec will contain:
- `info.title`: "Actual Helper API"
- `info.version`: "1.0.0"
- `/convert/{provider}` POST operation with:
  - `provider` path parameter (string, required)
  - `multipart/form-data` request body with `file` (required, binary) and `password` (optional) fields
  - 200 response with `text/csv` content (binary download)
  - Standard 400/500 error responses (auto-registered by Fuego)

### Trade-offs

`[]byte` generates `format: byte` (base64) instead of `format: binary` (raw). Most Swagger UI tools render any `string` type in a multipart context as a file upload button, so this is acceptable. A custom schema customizer would be needed for `format: binary`, adding complexity for cosmetic gain.

## Testing

No behavioral changes — handler logic is untouched. Existing handler tests verify:
- 400 when file is missing
- 500 for unregistered provider
- 200 CSV on successful conversion

Run with `go test ./internal/handlers/...`.
