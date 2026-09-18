package handlers

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func multipartUpload(t *testing.T, fileContent, password string, includeFile bool) *http.Request {
	t.Helper()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if includeFile {
		part, err := w.CreateFormFile("file", "statement.csv")
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := part.Write([]byte(fileContent)); err != nil {
			t.Fatalf("write form file: %v", err)
		}
	}
	if password != "" {
		if err := w.WriteField("password", password); err != nil {
			t.Fatalf("write password field: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/convert/test", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestParseUploadCapturesFileAndPassword(t *testing.T) {
	upload, err := parseUpload(multipartUpload(t, "a,b,c", "secret", true), 1<<20)
	if err != nil {
		t.Fatalf("parseUpload: %v", err)
	}

	if string(upload.data) != "a,b,c" {
		t.Errorf("data = %q, want %q", upload.data, "a,b,c")
	}
	if upload.password != "secret" {
		t.Errorf("password = %q, want %q", upload.password, "secret")
	}
	if upload.contentType == "" {
		t.Error("contentType is empty")
	}
}

func TestParseUploadRejectsOversizeFile(t *testing.T) {
	_, err := parseUpload(multipartUpload(t, "0123456789", "", true), 5)
	if !errors.Is(err, errFileTooLarge) {
		t.Fatalf("err = %v, want errFileTooLarge", err)
	}
}

func TestParseUploadRequiresFile(t *testing.T) {
	_, err := parseUpload(multipartUpload(t, "", "secret", false), 1<<20)
	if !errors.Is(err, errFileRequired) {
		t.Fatalf("err = %v, want errFileRequired", err)
	}
}

func TestParseUploadRejectsNonMultipart(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/convert/test", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	if _, err := parseUpload(req, 1<<20); err == nil {
		t.Fatal("expected error for non-multipart request")
	}
}
