package uobcredit_test

import (
	"context"

	"actual_helper/internal/models"
	uobcreditprov "actual_helper/internal/providers/uobcredit"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParsePDFText", func() {
	var (
		provider = uobcreditprov.New(nil, nil, nil, nil)
		ctx      = context.Background()
	)

	It("parses a debit transaction", func() {
		text := `Statement Date    16 JUL 2026
          15 JUL              PURCHASE AT MERCHANT ABC                                                                                45.90`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("-45.90"))
		Expect(reports[0].Date).To(Equal("2026-07-15"))
		Expect(reports[0].Notes).To(Equal("PURCHASE AT MERCHANT ABC"))
	})

	It("parses a credit transaction with CR suffix", func() {
		text := `Statement Date    16 JUL 2026
          04 JUL              PAYMENT RECEIVED                                                                 500.00 CR`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("500.00"))
		Expect(reports[0].Date).To(Equal("2026-07-04"))
	})

	It("parses multiple transactions", func() {
		text := `Statement Date    16 JUL 2026
          04 JUL              PAYMENT RECEIVED                                                                500.00 CR
          15 JUL              PURCHASE AT MERCHANT ABC                                                                                45.90
          16 JUL              ONLINE SUBSCRIPTION                                                                                     29.90`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(3))
		Expect(reports[0].Amount).To(Equal("500.00"))
		Expect(reports[0].Date).To(Equal("2026-07-04"))
		Expect(reports[1].Amount).To(Equal("-45.90"))
		Expect(reports[2].Amount).To(Equal("-29.90"))
	})

	It("skips summary lines", func() {
		text := `Statement Date    16 JUL 2026
          15 JUL              PURCHASE AT MERCHANT ABC                                                                                45.90
SUB-TOTAL                                                                301.76
MINIMUM PAYMENT DUE                                                                                    301.76`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Notes).To(Equal("PURCHASE AT MERCHANT ABC"))
	})

	It("returns error for text without statement date", func() {
		_, err := provider.ParsePDFText(ctx, "random text")
		Expect(err).To(HaveOccurred())
	})

	It("returns error for text with header but no transactions", func() {
		text := `Statement Date    16 JUL 2026`

		_, err := provider.ParsePDFText(ctx, text)
		Expect(err).To(MatchError("no transactions found after filtering"))
	})

	It("handles year boundary: December transaction on January statement", func() {
		text := `Statement Date    05 JAN 2027
          25 DEC              HOLIDAY STORE                                                                 60.00
          02 JAN              REFUND PROCESSED                                                                                       20.00 CR`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(2))
		Expect(reports[0].Date).To(Equal("2026-12-25"))
		Expect(reports[0].Amount).To(Equal("-60.00"))
		Expect(reports[1].Date).To(Equal("2027-01-02"))
		Expect(reports[1].Amount).To(Equal("20.00"))
	})

	It("extracts full card number from text for account name", func() {
		text := `WORLD MASTERCARD              1234-5678-9012-3456
Statement Date    16 JUL 2026
          15 JUL              PURCHASE AT MERCHANT ABC                                                                                45.90`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Account).To(Equal("1234-5678-9012-3456"))
	})

	It("falls back to UOB Credit Card when card number is masked", func() {
		text := `WORLD MASTERCARD              **           -8648**
Statement Date    16 JUL 2026
          15 JUL              PURCHASE AT MERCHANT ABC                                                                                45.90`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Account).To(Equal("UOB Credit Card"))
	})

	It("filters description using exclude keywords", func() {
		provider := uobcreditprov.New([]string{"SUBSCRIPTION"}, nil, nil, nil)
		text := `Statement Date    16 JUL 2026
          15 JUL              PURCHASE AT MERCHANT ABC                                                                                45.90
          16 JUL              ONLINE SUBSCRIPTION                                                                                     29.90`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Notes).To(Equal("PURCHASE AT MERCHANT ABC"))
	})

	It("matches category from description", func() {
		categories := []models.CategoryRule{
			{Keyword: "MERCHANT", Group: "Shopping", Category: "Retail"},
		}
		provider := uobcreditprov.New(nil, nil, categories, nil)
		text := `Statement Date    16 JUL 2026
          15 JUL              PURCHASE AT MERCHANT ABC                                                                                45.90`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].CategoryGroup).To(Equal("Shopping"))
		Expect(reports[0].Category).To(Equal("Retail"))
	})

	It("skips credit limit previous balance entry", func() {
		text := `Statement Date    16 JUL 2026
          04 JUL              CREDIT LIMIT PREVIOUS BAL PAYMENT RECEIVED                                                               326.76 CR
          15 JUL              PURCHASE AT MERCHANT ABC                                                                                45.90`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Notes).To(Equal("PURCHASE AT MERCHANT ABC"))
	})
})
