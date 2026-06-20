package ryt_test

import (
	"context"

	rytprov "actual-helper/internal/providers/ryt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParsePDFText", func() {
	var (
		provider = rytprov.New(nil, nil, nil, nil)
		ctx      = context.Background()
	)

	It("parses a credit transaction", func() {
		text := `Account Transactions / Transaksi Akaun
Main Account / Akaun Utama
Date
Tarikh
Description
Butiran
(MYR)
Amount
Amaun
(MYR)
Balance
Baki
1 May 2026
From Alice Tan
Transfer
Sent from Online
Ref. ID: F20260501ABCDEF1
+783.88
784.14`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("783.88"))
		Expect(reports[0].Date).To(Equal("2026-05-01"))
		Expect(reports[0].Payee).To(BeEmpty())
		Expect(reports[0].Notes).To(ContainSubstring("From Alice Tan"))
		Expect(reports[0].Notes).To(ContainSubstring("Transfer"))
		Expect(reports[0].Notes).To(ContainSubstring("Ref. ID: F20260501ABCDEF1"))
	})

	It("parses a debit transaction", func() {
		text := `Account Transactions / Transaksi Akaun
Main Account / Akaun Utama
Date
Tarikh
Description
Butiran
(MYR)
Amount
Amaun
(MYR)
Balance
Baki
1 May 2026
To Savings Goal
Money movement
Ref. ID: F20260501GHIJKL2
-784.14
0.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("-784.14"))
		Expect(reports[0].Date).To(Equal("2026-05-01"))
	})

	It("skips opening balance row", func() {
		text := `Account Transactions / Transaksi Akaun
Main Account / Akaun Utama
Date
Tarikh
Description
Butiran
(MYR)
Amount
Amaun
(MYR)
Balance
Baki
1 May 2026
Opening balance
0.26
1 May 2026
From Alice Tan
Transfer
Sent from Online
Ref. ID: F20260501ABCDEF1
+783.88
784.14`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
	})

	It("parses multiple transactions", func() {
		text := `Account Transactions / Transaksi Akaun
Main Account / Akaun Utama
Date
Tarikh
Description
Butiran
(MYR)
Amount
Amaun
(MYR)
Balance
Baki
1 May 2026
From Alice Tan
Transfer
Sent from Online
Ref. ID: F20260501ABCDEF1
+783.88
784.14
2 May 2026
From Daily Wallet
Money movement
Ref. ID: F20260502MNOPQR3
+10.00
10.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(2))
		Expect(reports[0].Amount).To(Equal("783.88"))
		Expect(reports[1].Amount).To(Equal("10.00"))
	})

	It("returns error for text without account transactions section", func() {
		_, err := provider.ParsePDFText(ctx, "random text")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no account transactions section found"))
	})

	It("returns empty for text with header but no transactions", func() {
		text := `Account Transactions / Transaksi Akaun
Main Account / Akaun Utama
Date
Tarikh
Description
Butiran
(MYR)
Amount
Amaun
(MYR)
Balance
Baki`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(BeEmpty())
	})
})
