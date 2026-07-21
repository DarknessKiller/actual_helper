package hlb_test

import (
	"context"
	"strings"

	hlbprov "actual_helper/internal/providers/hlb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HLBProvider", func() {
	Describe("Name", func() {
		It("returns hlb", func() {
			provider := hlbprov.New(nil, nil, nil, nil)
			Expect(provider.Name()).To(Equal("hlb"))
		})
	})

	Describe("ParseCSV", func() {
		It("returns error because hlb only supports PDF", func() {
			provider := hlbprov.New(nil, nil, nil, nil)
			_, err := provider.ParseCSV(context.Background(), strings.NewReader("a,b,c\n1,2,3"))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ParsePDFText with account mapping", func() {
		It("maps account name from config using card number", func() {
			accountMappings := map[string]string{
				"1234 5678 9012 3456": "HLB Credit",
			}
			provider := hlbprov.New(nil, nil, nil, accountMappings)
			text := `Credit Card Number    1234 5678 9012 3456
Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      STORE-ABC          KOTA LAMA                                                                     25.00`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Account).To(Equal("HLB Credit"))
		})

		It("falls back to HLB Credit Card when no card number in PDF", func() {
			provider := hlbprov.New(nil, nil, nil, nil)
			text := `Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      STORE-ABC          KOTA LAMA                                                                     25.00`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Account).To(Equal("HLB Credit Card"))
		})
	})

	Describe("ParsePDFText with account mapping for debit", func() {
		It("maps account name from config using debit account number", func() {
			accountMappings := map[string]string{
				"98765432101": "HLB Savings",
			}
			provider := hlbprov.New(nil, nil, nil, accountMappings)
			text := `A/C No / No Akaun
: 98765432101
Statement Period /
: 01/06/26 - 30/06/26
Tempoh Penyataan

Date
Tarikh
15-06-2026

Transaction Description
Deskripsi Transaksi

Simpanan

Withdrawal
Pengeluaran

Balance
Baki

Grocery Store
2500.00`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Account).To(Equal("HLB Savings"))
		})

		It("falls back to HLB Debit Account when no account number in PDF", func() {
			provider := hlbprov.New(nil, nil, nil, nil)
			text := `No Akaun
Statement Period /
Tempoh Penyataan

Date
Tarikh
15-06-2026

Transaction Description
Deskripsi Transaksi

Simpanan

Withdrawal
Pengeluaran

Balance
Baki

Grocery Store
2500.00`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Account).To(Equal("HLB Debit Account"))
		})
	})

	Describe("ParsePDFText auto-detects debit format", func() {
		It("routes to debit parser for debit statement text", func() {
			provider := hlbprov.New(nil, nil, nil, nil)
			text := `A/C No / No Akaun
: 98765432101
Statement Period /
: 01/06/26 - 30/06/26
Tempoh Penyataan

Date
Tarikh
15-06-2026

Transaction Description
Deskripsi Transaksi

Simpanan

Withdrawal
Pengeluaran

Balance
Baki

ATM Withdrawal
150.00`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Notes).To(Equal("ATM Withdrawal"))
			Expect(reports[0].Amount).To(Equal("-150.00"))
		})
	})

	Describe("ParsePDFText with filtering", func() {
		It("skips rows matching exclude keywords", func() {
			provider := hlbprov.New([]string{"STORE"}, nil, nil, nil)
			text := `Tarikh Penyata                    14 JUL 2026
  15 JUN          16 JUN      STORE-ABC          KOTA LAMA                                                                     25.00
  20 JUN          22 JUN      FUEL STATION       SEGAMBUT                                                                       75.00`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Notes).To(Equal("FUEL STATION SEGAMBUT"))
		})
	})
})
