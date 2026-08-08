//go:build !cgo

package pdfutil

import (
	"context"
	"fmt"
)

type tesseractClient struct{}

func newOCRClient() (*tesseractClient, error) {
	return nil, fmt.Errorf("OCR requires CGO (install tesseract-ocr-dev)")
}

func (c *tesseractClient) Close() error { return nil }

func ocrImage(_ context.Context, _ *tesseractClient, _ string) (string, error) {
	return "", fmt.Errorf("OCR requires CGO (install tesseract-ocr-dev)")
}
