package handlers_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"

	"actual-helper/internal/handlers"
	"actual-helper/internal/providers"
	tngprov "actual-helper/internal/providers/tng"
	"actual-helper/internal/services"

	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ConvertHandler", func() {
	Describe("via HTTP", func() {
		It("returns 400 when file is missing", func() {
			reg := providers.NewRegistry()
			svc := services.NewConvertService(reg)
			dummyHandler := handlers.NewConvertHandler(svc)

			c := fuego.NewServer()
			handlers.RegisterConvertRoutes(c, dummyHandler)

			req := httptest.NewRequest("POST", "/convert/tng", nil)
			w := httptest.NewRecorder()
			c.Mux.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("returns 500 for unknown provider", func() {
			reg := providers.NewRegistry()
			svc := services.NewConvertService(reg)
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

			req := httptest.NewRequest("POST", "/convert/tng", &buf)
			req.Header.Set("Content-Type", w.FormDataContentType())
			rr := httptest.NewRecorder()
			s.Mux.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
		})

		It("returns CSV on successful conversion", func() {
			reg := providers.NewRegistry()
			reg.Register(tngprov.New())

			svc := services.NewConvertService(reg)
			dummyHandler := handlers.NewConvertHandler(svc)

			s := fuego.NewServer()
			fuego.Post(s, "/convert/{provider}", dummyHandler.Convert,
				option.Tags("convert"),
			)

			var buf bytes.Buffer
			w := multipart.NewWriter(&buf)
			fw, _ := w.CreateFormFile("file", "test.csv")
			fw.Write([]byte("F,Status,Transaction Type,Reference,Description,Details,Amount(RM)\n13/6/2026,Success,Reload,TXN001,Top Up,Test,500.00"))
			w.Close()

			req := httptest.NewRequest("POST", "/convert/tng", &buf)
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
