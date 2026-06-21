package services_test

import (
	"bytes"
	"context"
	"io"
	"strings"

	"actual_helper/internal/models"
	"actual_helper/internal/providers"
	"actual_helper/internal/services"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type mockProvider struct {
	name        string
	csvReports  []models.ActualBudgetReport
	csvErr      error
	pdfReports  []models.ActualBudgetReport
	pdfErr      error
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) ParseCSV(_ context.Context, _ io.Reader) ([]models.ActualBudgetReport, error) {
	return m.csvReports, m.csvErr
}
func (m *mockProvider) ParsePDFText(_ context.Context, _ string) ([]models.ActualBudgetReport, error) {
	return m.pdfReports, m.pdfErr
}

var _ = Describe("ConvertService", func() {
	var (
		svc     *services.ConvertService
		reg     *providers.Registry
		ctx     = context.Background()
	)

	BeforeEach(func() {
		reg = providers.NewRegistry()
		svc = services.NewConvertService(reg, nil)
	})

	Describe("ConvertFile", func() {
		It("returns error for unknown provider", func() {
			_, err := svc.ConvertFile(ctx, "unknown", strings.NewReader(""), "", "", "")
			Expect(err).To(MatchError(ContainSubstring(`provider "unknown" not found`)))
		})

		It("parses CSV and returns CSV bytes", func() {
			mock := &mockProvider{
				name: "test",
				csvReports: []models.ActualBudgetReport{
					{Account: "Current", Date: "2026-06-13", Payee: "Test", Amount: "100.00"},
				},
			}
			reg.Register(mock)

			csvBytes, err := svc.ConvertFile(ctx, "test", strings.NewReader("a,b,c"), "test.csv", "text/csv", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(csvBytes).NotTo(BeEmpty())

			output := string(csvBytes)
			Expect(output).To(ContainSubstring("Account,Date,Payee,Notes,Category_Group,Category,Amount,Split_Amount,Cleared"))
			Expect(output).To(ContainSubstring("Current,2026-06-13,Test,,,,"))
		})

		It("routes PDF files to ParsePDFText", func() {
			mock := &mockProvider{
				name: "test",
				pdfErr: nil,
			}
			reg.Register(mock)

			_, err := svc.ConvertFile(ctx, "test", strings.NewReader("%PDF-content"), "test.pdf", "application/pdf", "")
			Expect(err).To(HaveOccurred())
		})

		It("returns multiple records as CSV", func() {
			mock := &mockProvider{
				name: "test",
				csvReports: []models.ActualBudgetReport{
					{Account: "A", Date: "2026-01-01", Payee: "P1", Amount: "10.00"},
					{Account: "A", Date: "2026-01-02", Payee: "P2", Amount: "-5.00"},
				},
			}
			reg.Register(mock)

			csvBytes, err := svc.ConvertFile(ctx, "test", strings.NewReader("a,b,c"), "test.csv", "text/csv", "")
			Expect(err).NotTo(HaveOccurred())

			lines := strings.Split(strings.TrimSpace(string(csvBytes)), "\n")
			Expect(lines).To(HaveLen(3))
			Expect(lines[1]).To(ContainSubstring("P1"))
			Expect(lines[2]).To(ContainSubstring("P2"))
		})
	})

	Describe("ToActualCSV", func() {
		It("writes CSV header", func() {
			reports := []models.ActualBudgetReport{}
			data, err := services.ToActualCSV(reports)
			Expect(err).NotTo(HaveOccurred())

			output := string(data)
			Expect(output).To(Equal("Account,Date,Payee,Notes,Category_Group,Category,Amount,Split_Amount,Cleared\n"))
		})

		It("writes all fields from reports", func() {
			reports := []models.ActualBudgetReport{
				{
					Account:       "Savings",
					Date:          "2026-06-15",
					Payee:         "Alice",
					Notes:         "Birthday gift",
					CategoryGroup: "Income",
					Category:      "Gifts",
					Amount:        "200.00",
					SplitAmount:   "",
					Cleared:       "true",
				},
			}
			data, err := services.ToActualCSV(reports)
			Expect(err).NotTo(HaveOccurred())

			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			Expect(lines[1]).To(Equal("Savings,2026-06-15,Alice,Birthday gift,Income,Gifts,200.00,,true"))
		})

		It("handles empty report list", func() {
			data, err := services.ToActualCSV([]models.ActualBudgetReport{})
			Expect(err).NotTo(HaveOccurred())
			Expect(bytes.HasSuffix(data, []byte("\n"))).To(BeTrue())
		})
	})
})
