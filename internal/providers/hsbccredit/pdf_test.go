package hsbccredit_test

import (
	"context"

	"actual_helper/internal/models"
	hsbccreditprov "actual_helper/internal/providers/hsbccredit"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParsePDFText", func() {
	var (
		provider = hsbccreditprov.New(nil, nil, nil, nil)
		ctx      = context.Background()
	)

	It("parses a credit transaction (payment)", func() {
		text := `Statement Date 04 Jun 2026
Post date | Transaction date | Transaction details | Amount (RM)
17 MAY 17 MAY PAYMENT - RECEIVED 259.72CR
Your charge(s) for this month RM259.72`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("259.72"))
		Expect(reports[0].Date).To(Equal("2026-05-17"))
		Expect(reports[0].Payee).To(BeEmpty())
		Expect(reports[0].Notes).To(Equal("PAYMENT - RECEIVED"))
	})

	It("parses a debit transaction (purchase) as negative", func() {
		text := `Statement Date 04 Jun 2026
Post date | Transaction date | Transaction details | Amount (RM)
06 MAY 05 MAY AP Online Retail MY 8.50
Your charge(s) for this month RM8.50`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("-8.50"))
		Expect(reports[0].Date).To(Equal("2026-05-05"))
		Expect(reports[0].Notes).To(Equal("AP Online Retail MY"))
	})

	It("parses multiple transactions", func() {
		text := `Statement Date 04 Jun 2026
Post date | Transaction date | Transaction details | Amount (RM)
06 MAY 05 MAY AP Online Retail 8.50
15 MAY 14 MAY Digital Service 38.00
Your charge(s) for this month RM50.50`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(2))
		Expect(reports[0].Amount).To(Equal("-8.50"))
		Expect(reports[1].Amount).To(Equal("-38.00"))
	})

	It("skips summary lines", func() {
		text := `Statement Date 04 Jun 2026
Post date | Transaction date | Transaction details | Amount (RM)
Your Previous Statement Balance 259.72
Credit limit used last statement RM259.72
Your Credit Limit: RM5,000
17 MAY 17 MAY PAYMENT - RECEIVED 259.72CR
Your charge(s) for this month RM259.72
Total credit limit used RM259.72
Your statement balance 259.72`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("259.72"))
		Expect(reports[0].Notes).To(Equal("PAYMENT - RECEIVED"))
	})

	It("returns error for text without statement date", func() {
		_, err := provider.ParsePDFText(ctx, "random text")
		Expect(err).To(HaveOccurred())
	})

	It("returns error for text with header but no transactions", func() {
		text := `Statement Date 04 Jun 2026
Post date | Transaction date | Transaction details | Amount (RM)`

		_, err := provider.ParsePDFText(ctx, text)
		Expect(err).To(MatchError("no transactions found after filtering"))
	})

	It("handles year boundary: December transaction on January statement", func() {
		text := `Statement Date 04 Jan 2027
Post date | Transaction date | Transaction details | Amount (RM)
25 DEC 24 DEC Online Shopping 50.00
02 JAN 01 JAN Ride Service 15.00CR`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(2))
		Expect(reports[0].Date).To(Equal("2026-12-24"))
		Expect(reports[0].Amount).To(Equal("-50.00"))
		Expect(reports[1].Date).To(Equal("2027-01-01"))
		Expect(reports[1].Amount).To(Equal("15.00"))
	})

	It("filters description using exclude keywords", func() {
		provider := hsbccreditprov.New([]string{"Online"}, nil, nil, nil)
		text := `Statement Date 04 Jun 2026
Post date | Transaction date | Transaction details | Amount (RM)
06 MAY 05 MAY AP Online Retail 8.50
10 MAY 07 MAY Ride Service 4.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("-4.00"))
	})

	It("matches category from description", func() {
		categories := []models.CategoryRule{
			{Keyword: "Retail", Group: "Shopping", Category: "Online"},
		}
		provider := hsbccreditprov.New(nil, nil, categories, nil)
		text := `Statement Date 04 Jun 2026
Post date | Transaction date | Transaction details | Amount (RM)
06 MAY 05 MAY AP Retail Purchase MY 8.50`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].CategoryGroup).To(Equal("Shopping"))
		Expect(reports[0].Category).To(Equal("Online"))
	})

	It("extracts card number from text for account name", func() {
		text := `Card Number : 1234 5678 9012 3456
Statement Date 04 Jun 2026
Post date | Transaction date | Transaction details | Amount (RM)
17 MAY 17 MAY PAYMENT - RECEIVED 259.72CR
Your charge(s) for this month RM259.72`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Account).To(Equal("1234 5678 9012 3456"))
	})

	It("falls back to HSBC Credit Card when no card number found", func() {
		text := `Statement Date 04 Jun 2026
Post date | Transaction date | Transaction details | Amount (RM)
17 MAY 17 MAY PAYMENT - RECEIVED 259.72CR
Your charge(s) for this month RM259.72`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Account).To(Equal("HSBC Credit Card"))
	})

	It("handles card number with dashes", func() {
		text := `Card Number : 1234-5678-9012-3456
Statement Date 04 Jun 2026
Post date | Transaction date | Transaction details | Amount (RM)
17 MAY 17 MAY PAYMENT - RECEIVED 259.72CR
Your charge(s) for this month RM259.72`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Account).To(Equal("1234 5678 9012 3456"))
	})

	It("returns error when all rows are zero amount", func() {
		text := `Statement Date 04 Jun 2026
Post date | Transaction date | Transaction details | Amount (RM)
06 MAY 05 MAY AP Online Retail 0.00`

		_, err := provider.ParsePDFText(ctx, text)
		Expect(err).To(MatchError("no transactions found after filtering"))
	})
})
