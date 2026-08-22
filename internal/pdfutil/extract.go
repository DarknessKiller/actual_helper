package pdfutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type ExtractionMethod string

const (
	ExtractionMethodDigital   ExtractionMethod = "digital"
	ExtractionMethodPdftotext ExtractionMethod = "pdftotext"
	ExtractionMethodOCR       ExtractionMethod = "ocr"

	timeoutDigital   = 30 * time.Second
	timeoutPdftotext = 60 * time.Second
	timeoutOCR       = 5 * time.Minute
)

func ExtractText(ctx context.Context, r io.Reader, password string, method ExtractionMethod) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}

	if password != "" {
		var buf bytes.Buffer
		conf := model.NewDefaultConfiguration()
		conf.UserPW = password
		if err := api.Decrypt(bytes.NewReader(data), &buf, conf); err != nil {
			return "", fmt.Errorf("decrypt pdf: %w", err)
		}
		data = buf.Bytes()
	}

	switch method {
	case ExtractionMethodDigital:
		return extractDigital(ctx, data)
	case ExtractionMethodPdftotext:
		return extractWithPdftotext(ctx, data)
	case ExtractionMethodOCR:
		return extractWithOCR(ctx, data)
	default:
		return "", fmt.Errorf("unknown extraction method: %s", method)
	}
}

func extractDigital(ctx context.Context, data []byte) (string, error) {
	tmpDir, err := os.MkdirTemp("", "pdfutil")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "input.pdf")
	if err := os.WriteFile(srcPath, data, 0644); err != nil {
		return "", fmt.Errorf("write temp pdf: %w", err)
	}

	// ponytail: ledongthuc/pdf is pure Go, no subprocess to cancel.
	// Wrap in timeout via goroutine since CGO-free lib can't use exec.CommandContext.
	type result struct {
		text string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		f, reader, err := pdf.Open(srcPath)
		if err != nil {
			ch <- result{err: fmt.Errorf("open pdf: %w", err)}
			return
		}
		defer f.Close()

		var text string
		for i := 1; i <= reader.NumPage(); i++ {
			if ctx.Err() != nil {
				ch <- result{err: ctx.Err()}
				return
			}
			page := reader.Page(i)
			pageText, err := page.GetPlainText(nil)
			if err != nil {
				continue
			}
			text += pageText + "\n"
		}
		ch <- result{text: text}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-ch:
		return r.text, r.err
	}
}

func extractWithPdftotext(ctx context.Context, data []byte) (string, error) {
	tmpDir, err := os.MkdirTemp("", "pdftext")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	pdfPath := filepath.Join(tmpDir, "input.pdf")
	if err := os.WriteFile(pdfPath, data, 0644); err != nil {
		return "", fmt.Errorf("write temp pdf: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, timeoutPdftotext)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", pdfPath, "-")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("pdftotext timeout: %w", ctx.Err())
		}
		return "", fmt.Errorf("pdftotext failed: %w", err)
	}

	return out.String(), nil
}

const maxStripHeight = 4000
const stripOverlap = 200

func extractWithOCR(ctx context.Context, data []byte) (string, error) {
	tmpDir, err := os.MkdirTemp("", "pdfocr")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	pdfPath := filepath.Join(tmpDir, "input.pdf")
	if err := os.WriteFile(pdfPath, data, 0644); err != nil {
		return "", fmt.Errorf("write temp pdf: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, timeoutOCR)
	defer cancel()

	// Page count via pdfinfo (cheap, poppler) so we render one page at a time
	// instead of all at once — keeps peak memory to a single page's PNG.
	pageCount, err := pdfPageCount(ctx, pdfPath)
	if err != nil {
		return "", fmt.Errorf("get page count: %w", err)
	}

	client, err := newOCRClient()
	if err != nil {
		return "", err
	}
	defer client.Close()

	var text string
	for page := 1; page <= pageCount; page++ {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		// Render only this page, OCR it, then delete it before the next —
		// peak disk + memory is one page, not the whole document.
		pagePath := filepath.Join(tmpDir, fmt.Sprintf("page-%d.png", page))
		rcmd := exec.CommandContext(ctx, "pdftoppm", "-png", "-r", "150", "-f", strconv.Itoa(page), "-l", strconv.Itoa(page), pdfPath, filepath.Join(tmpDir, "page"))
		if out, err := rcmd.CombinedOutput(); err != nil {
			if ctx.Err() != nil {
				return "", fmt.Errorf("pdftoppm timeout: %w", ctx.Err())
			}
			slog.Warn("pdftoppm page failed, skipping", "page", page, "error", err, "output", string(out))
			continue
		}

		stripPaths, err := splitIntoStrips(ctx, pagePath)
		if err != nil {
			slog.Warn("failed to split page into strips, trying full page", "page", page, "error", err)
			stripPaths = []string{pagePath}
		}

		for _, stripPath := range stripPaths {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}

			pageText, err := ocrImage(ctx, client, stripPath)
			if stripPath != pagePath {
				os.Remove(stripPath)
			}
			if err != nil {
				slog.Warn("ocr strip skipped", "path", stripPath, "error", err)
				continue
			}
			slog.Debug("ocr strip extracted", "path", stripPath, "chars", len(pageText))
			text += pageText + "\n"
		}

		os.Remove(pagePath)
	}

	return text, nil
}

func splitIntoStrips(ctx context.Context, path string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "identify", "-format", "%h", path)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("identify failed: %w", err)
	}

	var height int
	if _, err := fmt.Sscanf(out.String(), "%d", &height); err != nil {
		return nil, fmt.Errorf("parse height: %w", err)
	}

	if height <= maxStripHeight {
		return []string{path}, nil
	}

	var strips []string
	for y := 0; y < height; {
		if ctx.Err() != nil {
			return strips, ctx.Err()
		}

		stripH := maxStripHeight
		if y+stripH > height {
			stripH = height - y
		}

		stripPath := fmt.Sprintf("%s.strip.%d.png", path, len(strips))
		crop := exec.CommandContext(ctx, "convert", path, "-crop", fmt.Sprintf("%dx%d+0+%d", 0, stripH, y), "+repage", stripPath)
		if out, err := crop.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("convert crop failed: %w\n%s", err, string(out))
		}
		strips = append(strips, stripPath)

		y += stripH - stripOverlap
		if y >= height {
			break
		}
	}

	return strips, nil
}

// pdfPageCount returns the number of pages in a PDF via the pdfinfo CLI
// (poppler-utils, present in the Docker runtime image).
func pdfPageCount(ctx context.Context, pdfPath string) (int, error) {
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, "pdfinfo", pdfPath)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("pdfinfo: %w", err)
	}
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.HasPrefix(line, "Pages:") {
			n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Pages:")))
			if err != nil {
				return 0, fmt.Errorf("parse page count: %w", err)
			}
			return n, nil
		}
	}
	return 0, fmt.Errorf("page count not found in pdfinfo output")
}
