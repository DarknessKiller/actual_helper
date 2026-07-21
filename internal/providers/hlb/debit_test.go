package hlb_test

import (
	"context"

	hlbprov "actual_helper/internal/providers/hlb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("parseDebitPDF", func() {
	var (
		provider = hlbprov.New(nil, nil, nil, nil)
		ctx      = context.Background()
	)

	It("parses a debit transaction", func() {
		text := `A/C No / No Akaun
: 12345678901
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
	500.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Account).To(Equal("12345678901"))
		Expect(reports[0].Date).To(Equal("2026-06-15"))
		Expect(reports[0].Amount).To(Equal("-500.00"))
	})

	It("parses a credit transaction", func() {
		text := `A/C No / No Akaun
: 12345678901
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

	Salary Deposit
	5200.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("-5200.00"))
	})

	It("extracts account name from text", func() {
		text := `A/C No / No Akaun
: 12345678901
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
	500.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports[0].Account).To(Equal("12345678901"))
	})

	It("falls back to HLB Debit Account when no account number found", func() {
		text := `Statement Period /
: 01/06/26 - 30/06/26`

		_, err := provider.ParsePDFText(ctx, text)
		Expect(err).To(HaveOccurred())
	})

	It("parses multiple transactions", func() {
		text := `A/C No / No Akaun
: 12345678901
Statement Period /
: 01/06/26 - 30/06/26
Tempoh Penyataan

Date
Tarikh
15-06-2026
20-06-2026

Transaction Description
Deskripsi Transaksi

Simpanan

Withdrawal
Pengeluaran

Balance
Baki

ATM Withdrawal
500.00

Salary Deposit
5200.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(2))
		Expect(reports[0].Amount).To(Equal("-500.00"))
		Expect(reports[1].Amount).To(Equal("-5200.00"))
	})

	It("skips opening balance line", func() {
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
2500.00

Balance from previous statement
150.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Notes).To(Equal("Grocery Store"))
		Expect(reports[0].Amount).To(Equal("-2500.00"))
	})

	It("handles multiple transactions with dates", func() {
		text := `A/C No / No Akaun
: 98765432101
Statement Period /
: 01/06/26 - 30/06/26
Tempoh Penyataan

Date
Tarikh
10-06-2026
15-06-2026
20-06-2026

Transaction Description
Deskripsi Transaksi

Simpanan

Withdrawal
Pengeluaran

Balance
Baki

ATM Withdrawal
150.00

Grocery Store
2500.00

Utility Payment
88.50`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(3))
		Expect(reports[0].Date).To(Equal("2026-06-10"))
		Expect(reports[0].Notes).To(Equal("ATM Withdrawal"))
		Expect(reports[0].Amount).To(Equal("-150.00"))
		Expect(reports[1].Date).To(Equal("2026-06-15"))
		Expect(reports[1].Notes).To(Equal("Grocery Store"))
		Expect(reports[1].Amount).To(Equal("-2500.00"))
		Expect(reports[2].Date).To(Equal("2026-06-20"))
		Expect(reports[2].Notes).To(Equal("Utility Payment"))
		Expect(reports[2].Amount).To(Equal("-88.50"))
	})

	It("skips header lines correctly", func() {
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

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Notes).To(Equal("Grocery Store"))
	})

	It("detects credit when description is Deposit", func() {
		text := `A/C No / No Akaun
: 12345678901
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

Deposit
150.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("150.00"))
	})

	It("handles amounts with commas", func() {
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
1,234.56`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("-1234.56"))
	})

	It("parses layout format with single withdrawal", func() {
		text := `Date           Transaction Description                                                           Deposit                 Withdrawal                   Balance
   Tarikh          Deskripsi Transaksi                                                             Simpanan                  Pengeluaran                   Baki

                Balance from previous statement                                                                                                                      20.00
 26-06-2026     HLConnect DuitNow-previously Inst                                                                6,384.60
                Fund transfer
                Salary
                CHEN KAEL VIN
                20260626HLBBMYKL010ORM23935561`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Date).To(Equal("2026-06-26"))
		Expect(reports[0].Amount).To(Equal("-6384.60"))
		Expect(reports[0].Notes).To(ContainSubstring("HLConnect DuitNow-previously Inst"))
	})

	It("parses layout format with deposit and balance", func() {
		text := `Date           Transaction Description                                                           Deposit                 Withdrawal                   Balance
   Tarikh          Deskripsi Transaksi                                                             Simpanan                  Pengeluaran                   Baki

                Balance from previous statement                                                                                                                      20.00
 26-06-2026     Cr Adv-Interbank GIRO at KLM                                                                 6,384.60                                                20.00
                L2606262709321
                Outward ACH
                STRATEQ BUSINESSHUB SDN. BHD.`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Date).To(Equal("2026-06-26"))
		Expect(reports[0].Amount).To(Equal("6384.60"))
		Expect(reports[0].Notes).To(ContainSubstring("Cr Adv-Interbank GIRO at KLM"))
	})

	It("parses layout format with multiple transactions", func() {
		text := `Date           Transaction Description                                                           Deposit                 Withdrawal                   Balance
   Tarikh          Deskripsi Transaksi                                                             Simpanan                  Pengeluaran                   Baki

                Balance from previous statement                                                                                                                      20.00
 26-06-2026     HLConnect DuitNow-previously Inst                                                                6,384.60
                Fund transfer
                Salary
                CHEN KAEL VIN
                20260626HLBBMYKL010ORM23935561
 26-06-2026     Cr Adv-Interbank GIRO at KLM                                                                 6,384.60                                                20.00
                L2606262709321
                Outward ACH
                STRATEQ BUSINESSHUB SDN. BHD.
 30-06-2026     Service Charge                                                                                                                5.00                   15.00
 01-07-2026     Interest                                                                                         0.28                                                15.28`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(4))
		Expect(reports[0].Date).To(Equal("2026-06-26"))
		Expect(reports[0].Amount).To(Equal("-6384.60"))
		Expect(reports[1].Date).To(Equal("2026-06-26"))
		Expect(reports[1].Amount).To(Equal("6384.60"))
		Expect(reports[2].Date).To(Equal("2026-06-30"))
		Expect(reports[2].Amount).To(Equal("-5.00"))
		Expect(reports[3].Date).To(Equal("2026-07-01"))
		Expect(reports[3].Amount).To(Equal("0.28"))
	})

	It("extracts account from layout format", func() {
		text := `A/C No / No Akaun      : 31200037164         MYR
Date           Transaction Description                                                           Deposit                 Withdrawal                   Balance
   Tarikh          Deskripsi Transaksi                                                             Simpanan                  Pengeluaran                   Baki

                Balance from previous statement                                                                20.00
 26-06-2026     HLConnect DuitNow-previously Inst                                                                6,384.60`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Account).To(Equal("31200037164"))
	})

	It("skips opening balance in layout format", func() {
		text := `Date           Transaction Description                                                           Deposit                 Withdrawal                   Balance
   Tarikh          Deskripsi Transaksi                                                             Simpanan                  Pengeluaran                   Baki

                Balance from previous statement                                                                                                                      20.00
 30-06-2026     Service Charge                                                                                                                5.00                   15.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Notes).To(Equal("Service Charge"))
		Expect(reports[0].Amount).To(Equal("-5.00"))
	})

	It("handles layout format with interest credit", func() {
		text := `Date           Transaction Description                                                           Deposit                 Withdrawal                   Balance
   Tarikh          Deskripsi Transaksi                                                             Simpanan                  Pengeluaran                   Baki

                Balance from previous statement                                                                                                                      15.00
 01-07-2026     Interest                                                                                         0.28                                                15.28`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Date).To(Equal("2026-07-01"))
		Expect(reports[0].Amount).To(Equal("0.28"))
		Expect(reports[0].Notes).To(Equal("Interest"))
	})
})
