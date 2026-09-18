# Changelog

Notable changes to Actual Helper. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Releases before 0.8.0 are documented in [GitHub Releases](https://github.com/DarknessKiller/actual_helper/releases).

## [Unreleased]

## [0.8.0] - 2026-09-19

Supersedes `v0.8.0-rc1`.

### Changed

- Conversion is now RAM-only ([#18]): uploaded files, rendered PDF pages, and output CSVs never touch disk.
  - PDF extraction reads from memory (`pdf.NewReader`); `pdftotext`, `pdftocairo`, and `tesseract` are driven through stdin/stdout.
  - OCR strips are cropped with the Go standard library; ImageMagick is no longer part of the runtime image.
  - Multipart uploads are parsed in memory and capped at 32 MiB; larger files return `413` instead of being spooled to `os.TempDir`.
- Provider logs mask account/card numbers, keeping the last 4 digits (`**** **** **** 3456`) ([#19]).

### Fixed

- OCR: pages taller than the strip height no longer loop forever ([#18]).
- OCR: oversized uploads now fail with `413` instead of being silently written to disk ([#18]).

### Security

- No statement data is written to temp files or logs ([#18]).
- Upload, output, and decrypted PDF byte buffers are zeroed after each request ([#18]).
- Statement content (text previews, skipped-row descriptions/amounts, filenames) removed from logs ([#18], [#19]).

### Dependencies

- Go module dependencies updated to the latest versions ([7cd8712]).

[Unreleased]: https://github.com/DarknessKiller/actual_helper/compare/v0.8.0...HEAD
[0.8.0]: https://github.com/DarknessKiller/actual_helper/compare/v0.7.7...v0.8.0
[#18]: https://github.com/DarknessKiller/actual_helper/pull/18
[#19]: https://github.com/DarknessKiller/actual_helper/pull/19
[7cd8712]: https://github.com/DarknessKiller/actual_helper/commit/7cd8712
