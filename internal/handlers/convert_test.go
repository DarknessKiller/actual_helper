package handlers_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"

	"actual-helper/internal/handlers"
	"actual-helper/internal/models"
	"actual-helper/internal/providers"
	"actual-helper/internal/services"

	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) ParseCSV(_ context.Context, _ io.Reader) ([]models.ActualBudgetReport, error) {
	return []models.ActualBudgetReport{
		{Account: "Current", Date: "2026-06-13", Payee: "", Notes: "Top Up", Amount: "500.00"},
	}, nil
}
func (m *mockProvider) ParsePDFText(_ context.Context, _ string) ([]models.ActualBudgetReport, error) {
	return nil, nil
}

var _ = Describe("ConvertHandler", func() {
	Describe("via HTTP", func() {
		It("returns 400 when file is missing", func() {
			reg := providers.NewRegistry()
			reg.Register(&mockProvider{name: "test"})
			svc := services.NewConvertService(reg, nil)
			dummyHandler := handlers.NewConvertHandler(svc)

			c := fuego.NewServer()
			handlers.RegisterConvertRoutes(c, dummyHandler)

			req := httptest.NewRequest("POST", "/convert/test", nil)
			w := httptest.NewRecorder()
			c.Mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns 500 for unregistered provider", func() {
			reg := providers.NewRegistry()
			svc := services.NewConvertService(reg, nil)
			dummyHandler := handlers.NewConvertHandler(svc)

			s := fuego.NewServer()
			fuego.Post(s, "/convert/{provider}", dummyHandler.Convert,
				option.Tags("convert"),
			)

			var buf bytes.Buffer
			w := multipart.NewWriter(&buf)
			fw, _ := w.CreateFormFile("file", "test.csv")
			fw.Write([]byte("a,b,c"))
			w.Close()

			req := httptest.NewRequest("POST", "/convert/unknown", &buf)
			req.Header.Set("Content-Type", w.FormDataContentType())
			rr := httptest.NewRecorder()
			s.Mux.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("returns CSV on successful conversion", func() {
			reg := providers.NewRegistry()
			reg.Register(&mockProvider{name: "test"})

			svc := services.NewConvertService(reg, nil)
			dummyHandler := handlers.NewConvertHandler(svc)

			s := fuego.NewServer()
			fuego.Post(s, "/convert/{provider}", dummyHandler.Convert,
				option.Tags("convert"),
			)

			var buf bytes.Buffer
			w := multipart.NewWriter(&buf)
			fw, _ := w.CreateFormFile("file", "test.csv")
			fw.Write([]byte("dummy,csv,data"))
			w.Close()

			req := httptest.NewRequest("POST", "/convert/test", &buf)
			req.Header.Set("Content-Type", w.FormDataContentType())
			rr := httptest.NewRecorder()
			s.Mux.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Header().Get("Content-Type")).To(Equal("text/csv"))
			Expect(rr.Body.String()).To(ContainSubstring("Account,Date,Payee"))
			Expect(rr.Body.String()).To(ContainSubstring("Top Up"))
		})
	})
})
