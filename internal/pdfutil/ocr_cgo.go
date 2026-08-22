//go:build cgo

package pdfutil

import (
	"context"
	"fmt"

	"github.com/otiai10/gosseract/v2"
)

type tesseractClient struct {
	inner  *gosseract.Client
	leaked bool // true if a goroutine may still be using inner; skip Close
}

func newOCRClient() (*tesseractClient, error) {
	c := gosseract.NewClient()
	c.SetLanguage("eng", "msa")
	c.Trim = true
	return &tesseractClient{inner: c}, nil
}

func (c *tesseractClient) Close() error {
	if c.leaked {
		// ponytail: leaked gosseract C call can't be cancelled; closing now
		// would free C buffers the goroutine still reads. Leak the client
		// until process exit rather than race the C library.
		return nil
	}
	return c.inner.Close()
}

func ocrImage(ctx context.Context, c *tesseractClient, path string) (string, error) {
	type result struct {
		text string
		err  error
	}
	done := make(chan result, 1)

	go func() {
		if err := c.inner.SetImage(path); err != nil {
			done <- result{"", fmt.Errorf("set image: %w", err)}
			return
		}
		text, err := c.inner.Text()
		if err != nil {
			done <- result{"", fmt.Errorf("tesseract text: %w", err)}
			return
		}
		done <- result{text, nil}
	}()

	select {
	case <-ctx.Done():
		c.leaked = true // goroutine may still be in SetImage/Text
		return "", ctx.Err()
	case r := <-done:
		return r.text, r.err
	}
}
