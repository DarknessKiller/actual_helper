package hlb_test

import (
	"context"

	"actual_helper/internal/models"
	hlbprov "actual_helper/internal/providers/hlb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParsePDFText", func() {
	var (
		provider = hlbprov.New(nil, nil, nil, nil)
		ctx      = context.Background()
	)

	It("parses a debit transaction", func() {
		text := `Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      STORE-ABC          KOTA LAMA                                                                     25.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("-25.00"))
		Expect(reports[0].Date).To(Equal("2026-06-15"))
		Expect(reports[0].Notes).To(Equal("STORE-ABC KOTA LAMA"))
	})

	It("parses a credit transaction with CR suffix", func() {
		text := `Tarikh Penyata                    14 JUL 2026
  02 JUL          03 JUL      ONLINE STORE                                                                       50.00    CR`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("50.00"))
		Expect(reports[0].Date).To(Equal("2026-07-02"))
		Expect(reports[0].Notes).To(Equal("ONLINE STORE"))
	})

	It("parses multiple transactions", func() {
		text := `Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      STORE-ABC          KOTA LAMA                                                                     25.00
  20 JUN          22 JUN      FUEL STATION       SEGAMBUT                                                                       75.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(2))
		Expect(reports[0].Amount).To(Equal("-25.00"))
		Expect(reports[1].Amount).To(Equal("-75.00"))
	})

	It("skips summary lines", func() {
		text := `Tarikh Penyata                    14 JUL 2026
PREVIOUS BALANCE FROM LAST STATEMENT                                                                  200.00
NEW TRANSACTION / CHARGES
  15 JUN          16 JUN      STORE-ABC          KOTA LAMA                                                                     25.00
SUB TOTAL                                                                                             500.00
TOTAL BALANCE                                                                                         500.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Notes).To(Equal("STORE-ABC KOTA LAMA"))
	})

	It("returns error for text without statement date", func() {
		_, err := provider.ParsePDFText(ctx, "random text")
		Expect(err).To(HaveOccurred())
	})

	It("returns error for text with header but no transactions", func() {
		text := `Tarikh Penyata                    14 JUL 2026
NEW TRANSACTION / CHARGES`

		_, err := provider.ParsePDFText(ctx, text)
		Expect(err).To(MatchError("no transactions found after filtering"))
	})

	It("handles year boundary: December transaction on January statement", func() {
		text := `Tarikh Penyata                    14 JAN 2027
  25 DEC          26 DEC      HOLIDAY SHOP                                                                        60.00
  02 JAN          03 JAN      TRANSFER BACK                                                                      20.00CR`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(2))
		Expect(reports[0].Date).To(Equal("2026-12-25"))
		Expect(reports[0].Amount).To(Equal("-60.00"))
		Expect(reports[1].Date).To(Equal("2027-01-02"))
		Expect(reports[1].Amount).To(Equal("20.00"))
	})

	It("extracts card number from text for account name", func() {
		text := `Credit Card Number    1234 5678 9012 3456
Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      STORE-ABC          KOTA LAMA                                                                     25.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Account).To(Equal("1234 5678 9012 3456"))
	})

	It("falls back to HLB Credit Card when no card number found", func() {
		text := `Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      STORE-ABC          KOTA LAMA                                                                     25.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Account).To(Equal("HLB Credit Card"))
	})

	It("filters description using exclude keywords", func() {
		provider := hlbprov.New([]string{"STORE"}, nil, nil, nil)
		text := `Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      STORE-ABC          KOTA LAMA                                                                     25.00
  20 JUN          22 JUN      FUEL STATION       SEGAMBUT                                                                       75.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Notes).To(Equal("FUEL STATION SEGAMBUT"))
	})

	It("matches category from description", func() {
		categories := []models.CategoryRule{
			{Keyword: "STORE", Group: "Shopping", Category: "Retail"},
		}
		provider := hlbprov.New(nil, nil, categories, nil)
		text := `Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      STORE-ABC          KOTA LAMA                                                                     25.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].CategoryGroup).To(Equal("Shopping"))
		Expect(reports[0].Category).To(Equal("Retail"))
	})

	It("skips payment received lines", func() {
		text := `Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      STORE-ABC          KOTA LAMA                                                                     25.00
PAYMENT RECEIVED - THANK YOU
  04 JUL          04 JUL      PAYMENT THANK YOU CR                                                                200.00CR`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(2))
		Expect(reports[0].Notes).To(Equal("STORE-ABC KOTA LAMA"))
		Expect(reports[1].Notes).To(Equal("PAYMENT THANK YOU CR"))
	})
})
