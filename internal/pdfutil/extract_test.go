package pdfutil_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"actual_helper/internal/pdfutil"
)

func TestExtractText_ContextTimeout(t *testing.T) {
	fakePDF := []byte("%PDF-1.4\n1 0 obj<</Type/Catalog>>endobj\n%%EOF")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond)

	_, err := pdfutil.ExtractText(ctx, bytes.NewReader(fakePDF), "", pdfutil.ExtractionMethodDigital)
	if err == nil {
		t.Fatal("expected error from expired context")
	}
}
