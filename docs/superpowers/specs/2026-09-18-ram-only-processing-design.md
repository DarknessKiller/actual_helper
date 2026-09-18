# RAM-Only Processing (No Disk Persistence)

## Problem

Uploaded transaction files (statements) contain sensitive financial data. The privacy goal: input and output must live only in RAM and be released when a request completes. Today three things persist statement data outside RAM:

1. **`internal/pdfutil/extract.go` writes uploaded PDF bytes to `/tmp`** for all three extraction methods (`os.MkdirTemp` + `os.WriteFile`), plus per-page PNGs and cropped strips for OCR. `defer os.RemoveAll` unlinks them, but plaintext bank statements exist on disk during (and undeleted at crash) processing.
2. **Multipart uploads > 32 MiB are spooled to disk by Go's `net/http`** because the handler uses `Request.FormFile`, which calls `ParseMultipartForm(32<<20)`. Fuego's `MaxBodySize` is applied too late (after `FormFile`) and never bounds the part spool.
3. **Statement text is written to logs** (text previews, OCR block content, descriptions, amounts, uploaded filename). Logs outlive the process.

On Linux, `/tmp` may be disk-backed; on `os.TempDir` macOS/CI it is disk. There is no way to prove "RAM only" while any of the above exists.

## Design

### Approach

No temp files anywhere in the request path. Byte buffers only; external tools are driven through stdin/stdout pipes. Buffers we own are zeroed after use (best effort; Go strings and third-party internals cannot be zeroed).

### pdfutil: all extraction in memory

| Method | Before | After |
|--------|--------|-------|
| digital | write `/tmp/input.pdf`, `pdf.Open(path)` | `pdf.NewReader(bytes.NewReader(data), len(data))` |
| pdftotext | write `/tmp/input.pdf`, `pdftotext file -` | `pdftotext -layout - -` with `cmd.Stdin = data` |
| OCR | write `/tmp/input.pdf`, `pdftoppm` → PNG files, `identify`/`convert` → strip files, `tesseract file` | `pdfinfo -` for page count; `pdftocairo -png -r 200 -f N -l N -singlefile - -` per page via stdin/stdout; strip split done with stdlib `image/png` (`SubImage` + re-encode); `tesseract stdin stdout` |

Notes:

- `pdftoppm` cannot write to stdout (verified), `pdftocairo` can — both are in the already-installed `poppler-utils`.
- ImageMagick (`identify`/`convert`) is dropped from the OCR path and from the Docker image; stdlib PNG crop replaces it (also removes ImageMagick's own temp/pixel-cache spill hazard).
- The old strip loop never terminated for pages taller than `maxStripHeight` (`y += stripH - stripOverlap` with clamped `stripH`); the rewritten loop advances by a fixed stride and always terminates.
- `ExtractText` signature is `ExtractText(ctx, data []byte, password, method)` so callers hold one copy and can zero it; main's context deadlines (30s/60s/5min) and one-page-at-a-time rendering are preserved. Decrypted bytes are zeroed before returning.

### Handler: streaming multipart, RAM-capped

- Replace `Request.FormFile`/`c.Body()` with `Request.MultipartReader()`; iterate parts, read `file` into a capped in-memory buffer, read `password` (capped 4 KiB).
- Upload cap: `maxUploadBytes = 32 MiB` (package const). Over the cap → `413 Request Entity Too Large`; missing file / non-multipart → `400`.
- Zero the uploaded bytes and the response CSV bytes after the request completes.
- Stop logging the uploaded filename (size only).

### Logging

Remove statement content from logs; keep markers/counts/reasons.

- Remove all `text_preview`/`block`/`preview`/`description`/`raw` content fields from provider logs.
- Drop now-unused `dateutil.Truncate` and its test.

### Files

| File | Action | Purpose |
|------|--------|---------|
| `internal/pdfutil/extract.go` | Rewrite | In-memory extraction, no temp files |
| `internal/handlers/convert.go` | Modify | `MultipartReader`, size cap, buffer zeroing, log scrub |
| `internal/services/convert.go` | Modify | `[]byte` signature (no second copy) |
| `internal/providers/ryt/pdf.go`, `tng/pdf.go`, `cardutil/cardutil.go`, `providerbase/mapper.go`, `hsbccredit/pdf.go`, `hlbcredit/pdf.go`, `uobcredit/pdf.go` | Modify | Remove statement content from logs |
| `internal/dateutil/dateutil.go` (+test) | Modify | Delete unused `Truncate` |
| `Dockerfile` | Modify | Drop `imagemagick` |
| `internal/pdfutil/` tests + fixture | Create | ExtractText works and creates no temp files |
| `internal/handlers/convert_upload_test.go` | Create | Multipart parse: file, password, oversize, missing |

### Tests

| Test | Description |
|------|-------------|
| ExtractText digital | Fake statement fixture → text extracted, no temp files (TMPDIR pointed at empty dir stays empty) |
| ExtractText pdftotext | Same, skipped if `pdftotext` absent |
| ExtractText OCR | Same + strip path forced (`maxStripHeight` lowered), skipped if `pdftocairo`/`tesseract` absent |
| parseUpload happy path | Captures file bytes, filename, content type, password |
| parseUpload oversize | Small limit → `errFileTooLarge` |
| parseUpload no file | Only password part → `errFileRequired` |
| Handler HTTP | Existing 400/500/200 tests unchanged; no path writes files |

## Known limits (not fixable in-process)

- OS swap can move RAM to disk; deployments should disable swap (container default).
- Go strings (extracted text, password) cannot be zeroed; only `[]byte` buffers are.
- Frontend keeps filename+timestamp history in `localStorage` (client disk). Out of scope here; flag to user.

## No other changes

Parsing, filtering, categorization, and output CSV format are untouched.
