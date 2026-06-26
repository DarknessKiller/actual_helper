package pdfutil

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func ExtractText(r io.Reader, password string) (string, error) {
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

	methods := []struct {
		name string
		fn   func([]byte) (string, error)
	}{
		{"digital", extractDigital},
		{"pdftotext", extractWithPdftotext},
		{"ocr", extractWithOCR},
	}

	var best string
	for _, m := range methods {
		t, err := m.fn(data)
		if err != nil {
			slog.Warn("pdf extraction method failed", "method", m.name, "error", err)
			continue
		}
		if len(strings.TrimSpace(t)) > len(strings.TrimSpace(best)) {
			slog.Info("pdf extraction method produced longer text", "method", m.name, "chars", len(strings.TrimSpace(t)))
			best = t
		}
	}

	if strings.TrimSpace(best) == "" {
		return "", fmt.Errorf("all extraction methods returned empty text")
	}

	return best, nil
}

func extractDigital(data []byte) (string, error) {
	tmpDir, err := os.MkdirTemp("", "pdfutil")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "input.pdf")
	if err := os.WriteFile(srcPath, data, 0644); err != nil {
		return "", fmt.Errorf("write temp pdf: %w", err)
	}

	f, reader, err := pdf.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	defer f.Close()

	var text string
	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		pageText, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		text += pageText + "\n"
	}

	return text, nil
}

func extractWithPdftotext(data []byte) (string, error) {
	tmpDir, err := os.MkdirTemp("", "pdftext")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	pdfPath := filepath.Join(tmpDir, "input.pdf")
	if err := os.WriteFile(pdfPath, data, 0644); err != nil {
		return "", fmt.Errorf("write temp pdf: %w", err)
	}

	cmd := exec.Command("pdftotext", "-layout", pdfPath, "-")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext failed: %w", err)
	}

	return out.String(), nil
}

const maxStripHeight = 4000
const stripOverlap = 200

func extractWithOCR(data []byte) (string, error) {
	tmpDir, err := os.MkdirTemp("", "pdfocr")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	pdfPath := filepath.Join(tmpDir, "input.pdf")
	if err := os.WriteFile(pdfPath, data, 0644); err != nil {
		return "", fmt.Errorf("write temp pdf: %w", err)
	}

	outPrefix := filepath.Join(tmpDir, "page")
	cmd := exec.Command("pdftoppm", "-png", "-r", "200", pdfPath, outPrefix)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("pdftoppm failed: %w\n%s", err, string(out))
	}

	matches, err := filepath.Glob(filepath.Join(tmpDir, "page-*.png"))
	if err != nil {
		return "", fmt.Errorf("find page images: %w", err)
	}
	sort.Strings(matches)

	var text string
	for _, pagePath := range matches {
		stripPaths, err := splitIntoStrips(pagePath)
		if err != nil {
			slog.Warn("failed to split page into strips, trying full page", "path", pagePath, "error", err)
			stripPaths = []string{pagePath}
		}

		for _, stripPath := range stripPaths {
			pageText, err := ocrImage(stripPath)
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
	}

	return text, nil
}

func splitIntoStrips(path string) ([]string, error) {
	cmd := exec.Command("identify", "-format", "%h", path)
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
		stripH := maxStripHeight
		if y+stripH > height {
			stripH = height - y
		}

		stripPath := fmt.Sprintf("%s.strip.%d.png", path, len(strips))
		crop := exec.Command("convert", path, "-crop", fmt.Sprintf("%dx%d+0+%d", 0, stripH, y), "+repage", stripPath)
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

func ocrImage(path string) (string, error) {
	cmd := exec.Command("tesseract", path, "stdout", "-l", "eng+msa")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tesseract failed: %w", err)
	}
	return out.String(), nil
}
