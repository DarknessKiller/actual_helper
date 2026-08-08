//go:build cgo

package pdfutil

import (
	"context"
	"fmt"

	"github.com/otiai10/gosseract/v2"
)

type tesseractClient struct {
	inner *gosseract.Client
}

func newOCRClient() (*tesseractClient, error) {
	c := gosseract.NewClient()
	c.SetLanguage("eng", "msa")
	c.Trim = true
	return &tesseractClient{inner: c}, nil
}

func (c *tesseractClient) Close() error {
	return c.inner.Close()
}

func ocrImage(_ context.Context, c *tesseractClient, path string) (string, error) {
	if err := c.inner.SetImage(path); err != nil {
		return "", fmt.Errorf("set image: %w", err)
	}
	text, err := c.inner.Text()
	if err != nil {
		return "", fmt.Errorf("tesseract text: %w", err)
	}
	return text, nil
}
