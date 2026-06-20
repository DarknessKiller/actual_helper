package pdfutil

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
