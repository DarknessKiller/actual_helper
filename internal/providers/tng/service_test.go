package tng_test

import (
	"context"
	"strings"

	"actual-helper/internal/providers/tng"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TNGProvider", func() {
	var (
		provider = tng.New(nil, nil, nil)
		ctx      = context.Background()
	)

	Describe("ParseCSV", func() {
		It("parses valid CSV rows", func() {
			csv := "F,Status,Transaction Type,Reference,Description,Details,Amount(RM)\n" +
				"13/6/2026,Success,Reload,TXN001,Top Up from Bank,Test,500.00\n"

			reports, err := provider.ParseCSV(ctx, strings.NewReader(csv))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Payee).To(BeEmpty())
			Expect(reports[0].Date).To(Equal("2026-06-13"))
			Expect(reports[0].Amount).To(Equal("500.00"))
			Expect(reports[0].Account).To(Equal("Current"))
		})

		It("skips non-success status rows", func() {
			csv := "F,Status,Transaction Type,Reference,Description,Details,Amount(RM)\n" +
				"13/6/2026,Pending,Purchase,TXN001,Pending TX,Test,99.00\n"

			reports, err := provider.ParseCSV(ctx, strings.NewReader(csv))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(BeEmpty())
		})

		It("does not filter when no exclude keywords are set", func() {
			csv := "F,Status,Transaction Type,Reference,Description,Details,Amount(RM)\n" +
				"13/6/2026,Success,Purchase,TXN001,Quick Reload Payment,Test,50.00\n"

			reports, err := provider.ParseCSV(ctx, strings.NewReader(csv))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
		})

		It("skips rows with insufficient columns", func() {
			csv := "F,Status,Transaction Type,Reference,Description,Details,Amount(RM)\n" +
				"13/6/2026,Success,Purchase\n"

			reports, err := provider.ParseCSV(ctx, strings.NewReader(csv))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(BeEmpty())
		})

		It("returns empty for header-only CSV", func() {
			csv := "F,Status,Transaction Type,Reference,Description,Details,Amount(RM)\n"

			reports, err := provider.ParseCSV(ctx, strings.NewReader(csv))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(BeEmpty())
		})

		It("handles empty input", func() {
			reports, err := provider.ParseCSV(ctx, strings.NewReader(""))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(BeEmpty())
		})

		It("returns negative amount for purchases", func() {
			csv := "F,Status,Transaction Type,Reference,Description,Details,Amount(RM)\n" +
				"14/6/2026,Success,Purchase,TXN002,GrabFood Order,Test,25.50\n"

			reports, err := provider.ParseCSV(ctx, strings.NewReader(csv))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Amount).To(Equal("-25.50"))
		})

		It("handles DUITNOW_RECEIVEFROM as positive", func() {
			csv := "F,Status,Transaction Type,Reference,Description,Details,Amount(RM)\n" +
				"15/6/2026,Success,DUITNOW_RECEIVEFROM,TXN003,Gift from Alice,Test,100.00\n"

			reports, err := provider.ParseCSV(ctx, strings.NewReader(csv))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Amount).To(Equal("100.00"))
		})

		It("handles Refund as positive", func() {
			csv := "F,Status,Transaction Type,Reference,Description,Details,Amount(RM)\n" +
				"19/6/2026,Success,Refund,TXN007,Return Item,Test,45.00\n"

			reports, err := provider.ParseCSV(ctx, strings.NewReader(csv))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Amount).To(Equal("45.00"))
		})

		It("parses date in DD/MM/YYYY format", func() {
			csv := "F,Status,Transaction Type,Reference,Description,Details,Amount(RM)\n" +
				"01/12/2026,Success,Reload,TXN001,Salary,Test,1000.00\n"

			reports, err := provider.ParseCSV(ctx, strings.NewReader(csv))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Date).To(Equal("2026-12-01"))
		})

	})

	Describe("Name", func() {
		It("returns tng", func() {
			Expect(provider.Name()).To(Equal("tng"))
		})
	})
})
