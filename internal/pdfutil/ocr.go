package pdfutil

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ocrImage runs the tesseract CLI subprocess. Unlike the CGO gosseract
// binding, a subprocess is cleanly killable via context cancellation — no
// leaked C memory, no thread-safety hazards, no goroutine races. The image
// arrives on stdin and text leaves on stdout, so nothing touches disk.
func ocrImage(ctx context.Context, imageData []byte) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "tesseract", "stdin", "stdout", "-l", "eng+msa")
	cmd.Stdin = bytes.NewReader(imageData)
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
