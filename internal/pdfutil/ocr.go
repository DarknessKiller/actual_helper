package pdfutil

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ocrClient wraps the tesseract CLI subprocess. Unlike the CGO gosseract
// binding, a subprocess is cleanly killable via context cancellation — no
// leaked C memory, no thread-safety hazards, no goroutine races.
type ocrClient struct{}

func newOCRClient() (*ocrClient, error) {
	return &ocrClient{}, nil
}

func (c *ocrClient) Close() error { return nil }

func ocrImage(ctx context.Context, _ *ocrClient, path string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "tesseract", path, "stdout", "-l", "eng+msa")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("tesseract: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}
