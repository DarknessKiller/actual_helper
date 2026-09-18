package pdfutil

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
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

// ExtractText extracts text from a transaction PDF entirely in memory.
// No temporary files are written; external tools are driven via stdin/stdout.
func ExtractText(ctx context.Context, data []byte, password string, method ExtractionMethod) (string, error) {
	if password != "" {
		var buf bytes.Buffer
		conf := model.NewDefaultConfiguration()
		conf.UserPW = password
		if err := api.Decrypt(bytes.NewReader(data), &buf, conf); err != nil {
			return "", fmt.Errorf("decrypt pdf: %w", err)
		}
		zero(data)
		data = buf.Bytes()
	}
	defer zero(data)

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

// zero clears b. Best effort: []byte buffers we own are wiped before they are
// garbage collected, so freed pages do not retain statement data.
func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func extractDigital(ctx context.Context, data []byte) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeoutDigital)
	defer cancel()

	// ponytail: ledongthuc/pdf is pure Go, no subprocess to cancel.
	// Wrap in a goroutine so an expired context still returns.
	type result struct {
		text string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			ch <- result{err: fmt.Errorf("open pdf: %w", err)}
			return
		}

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
	ctx, cancel := context.WithTimeout(ctx, timeoutPdftotext)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", "-", "-")
	cmd.Stdin = bytes.NewReader(data)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("pdftotext timeout: %w", ctx.Err())
		}
		return "", fmt.Errorf("pdftotext failed: %w: %s", err, errOut.String())
	}

	return out.String(), nil
}

const maxStripHeight = 4000
const stripOverlap = 200

var pagesRe = regexp.MustCompile(`(?m)^Pages:\s+(\d+)`)

func extractWithOCR(ctx context.Context, data []byte) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeoutOCR)
	defer cancel()

	// Page count via pdfinfo (cheap, poppler) so we render one page at a time
	// instead of all at once — keeps peak memory to a single page's PNG.
	pageCount, err := pdfPageCount(ctx, data)
	if err != nil {
		return "", fmt.Errorf("get page count: %w", err)
	}

	var text string
	for page := 1; page <= pageCount; page++ {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		// Render only this page. The PNG stays in memory and is dropped
		// before the next page — peak memory is one page, not the document.
		pagePNG, err := renderPagePNG(ctx, data, page)
		if err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			slog.Warn("pdftocairo page failed, skipping", "page", page, "error", err)
			continue
		}

		pageText, err := ocrPNG(ctx, pagePNG)
		if err != nil {
			slog.Warn("ocr page skipped", "page", page, "error", err)
			continue
		}
		slog.Debug("ocr page extracted", "page", page, "chars", len(pageText))
		text += pageText + "\n"
	}

	return text, nil
}

// pdfPageCount returns the number of pages in a PDF via the pdfinfo CLI
// (poppler-utils, present in the Docker runtime image), reading the PDF from stdin.
func pdfPageCount(ctx context.Context, data []byte) (int, error) {
	cmd := exec.CommandContext(ctx, "pdfinfo", "-")
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("pdfinfo: %w", err)
	}

	match := pagesRe.FindSubmatch(out)
	if match == nil {
		return 0, fmt.Errorf("page count not found in pdfinfo output")
	}

	return strconv.Atoi(string(match[1]))
}

// renderPagePNG renders one page with pdftocairo, reading the PDF from stdin
// and writing the PNG to stdout (pdftoppm cannot write to stdout; pdftocairo can).
func renderPagePNG(ctx context.Context, data []byte, page int) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "pdftocairo", "-png", "-r", "200",
		"-f", strconv.Itoa(page), "-l", strconv.Itoa(page), "-singlefile",
		"-", "-")
	cmd.Stdin = bytes.NewReader(data)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftocairo: %w: %s", err, errOut.String())
	}
	if out.Len() == 0 {
		return nil, fmt.Errorf("pdftocairo produced no image")
	}

	return out.Bytes(), nil
}

// ocrPNG OCRs one page image, splitting pages taller than maxStripHeight into
// overlapping strips first. Strips are cropped in Go (stdlib image/png), so no
// ImageMagick and no temp files.
func ocrPNG(ctx context.Context, pagePNG []byte) (string, error) {
	img, err := png.Decode(bytes.NewReader(pagePNG))
	if err != nil {
		return "", fmt.Errorf("decode page image: %w", err)
	}

	bounds := img.Bounds()
	if bounds.Dy() <= maxStripHeight {
		return ocrImage(ctx, pagePNG)
	}

	sub, ok := img.(interface {
		SubImage(r image.Rectangle) image.Image
	})
	if !ok {
		return "", fmt.Errorf("decode page image: unsupported image type %T", img)
	}

	var text string
	for y := bounds.Min.Y; y < bounds.Max.Y; {
		if ctx.Err() != nil {
			return text, ctx.Err()
		}

		stripHeight := maxStripHeight
		if y+stripHeight > bounds.Max.Y {
			stripHeight = bounds.Max.Y - y
		}

		strip := sub.SubImage(image.Rect(bounds.Min.X, y, bounds.Max.X, y+stripHeight))
		var buf bytes.Buffer
		if err := png.Encode(&buf, strip); err != nil {
			return "", fmt.Errorf("encode strip: %w", err)
		}

		stripText, err := ocrImage(ctx, buf.Bytes())
		if err != nil {
			slog.Warn("ocr strip skipped", "error", err)
		} else {
			text += stripText + "\n"
		}

		if y+stripHeight >= bounds.Max.Y {
			break
		}
		y += maxStripHeight - stripOverlap
	}

	return text, nil
}
