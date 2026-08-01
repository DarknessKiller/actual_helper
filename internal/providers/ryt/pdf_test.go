package ryt_test

import (
	"context"

	rytprov "actual_helper/internal/providers/ryt"

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
1 Mar 2026
From Alice Tan
Transfer
Sent from Online
Ref. ID: REF20260301ABCDEF1
+123.45
123.45`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("123.45"))
		Expect(reports[0].Date).To(Equal("2026-03-01"))
		Expect(reports[0].Payee).To(BeEmpty())
		Expect(reports[0].Notes).To(ContainSubstring("From Alice Tan"))
		Expect(reports[0].Notes).To(ContainSubstring("Transfer"))
		Expect(reports[0].Notes).To(ContainSubstring("Ref. ID: REF20260301ABCDEF1"))
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
2 Mar 2026
To Savings Goal
Money movement
Ref. ID: REF20260302GHIJKL2
-456.78
0.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("-456.78"))
		Expect(reports[0].Date).To(Equal("2026-03-02"))
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
1 Mar 2026
Opening balance
0.26
1 Mar 2026
From Alice Tan
Transfer
Sent from Online
Ref. ID: REF20260301ABCDEF1
+123.45
123.45`

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
1 Mar 2026
From Alice Tan
Transfer
Sent from Online
Ref. ID: REF20260301ABCDEF1
+123.45
123.45
2 Mar 2026
From Daily Wallet
Money movement
Ref. ID: REF20260302MNOPQR3
+10.00
10.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(2))
		Expect(reports[0].Amount).To(Equal("123.45"))
		Expect(reports[1].Amount).To(Equal("10.00"))
	})

	It("returns error for text without account transactions section", func() {
		_, err := provider.ParsePDFText(ctx, "random text")
		Expect(err).To(HaveOccurred())
	})

	It("returns error for text with header but no transactions", func() {
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

		_, err := provider.ParsePDFText(ctx, text)
		Expect(err).To(HaveOccurred())
	})

	It("parses transactions across multiple pages", func() {
		text := `Savings Account Statement
/ Penyata Akaun Simpanan
From 1 Jul 2026 to 31 Jul 2026
Savings Account No.
 : 
/ Nombor Akaun Simpanan
12 3456 7890

Account Transactions
/ Transaksi Akaun
Main Account
/ Akaun Utama
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
1 Jul 2026
Opening balance
0.00
4 Jul 2026
From Daily Wallet
Money movement
Ref. ID: REF20260704AC938221
+181.42
181.42
Savings Account Statement
/ Penyata Akaun Simpanan
From 1 Jul 2026 to 31 Jul 2026
Savings Account No.
 : 
/ Nombor Akaun Simpanan
12 3456 7890

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
6 Jul 2026
From Daily Wallet
Money movement
Ref. ID: REF20260706FE218E5H
+15.80
15.80
23 Jul 2026
From Alice Tan
Transfer
Fake Transfer Reason
Ref. ID: REF202607232020D27
+14.67
14.67
END OF STATEMENT / PENYATA TAMAT`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(reports)).To(BeNumerically(">=", 2))
		Expect(reports[0].Account).To(Equal("Main Account"))
		Expect(reports[0].Amount).To(Equal("181.42"))
		Expect(reports[0].Date).To(Equal("2026-07-04"))
		Expect(reports[1].Account).To(Equal("Main Account"))
		Expect(reports[1].Amount).To(Equal("15.80"))
		Expect(reports[1].Date).To(Equal("2026-07-06"))
	})
})
