package tng_test

import (
	"context"

	"actual-helper/internal/providers/tng"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParsePDFText", func() {
	var (
		provider = tng.New(nil, nil, nil, nil)
		ctx      = context.Background()
	)

	It("parses a payment as debit (negative amount)", func() {
		text := `TNG WALLET TRANSACTION
Date
Status
Transaction Type
Reference
Description
Details
Amount (RM)
Wallet Balance
1/5/2026
Success
Payment
111111
Merchant A
222222
RM34.00
RM5.10`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Payee).To(BeEmpty())
		Expect(reports[0].Amount).To(Equal("-34.00"))
	})

	It("parses a reload as credit (positive amount)", func() {
		text := `TNG WALLET TRANSACTION
Date
Status
Transaction Type
Reference
Description
Details
Amount (RM)
Wallet Balance
1/5/2026
Success
Reload
111111
Top Up from Bank

RM100.00
RM150.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Payee).To(BeEmpty())
		Expect(reports[0].Amount).To(Equal("100.00"))
	})

	It("parses DUITNOW_RECEIVEFROM as credit", func() {
		text := `TNG WALLET TRANSACTION
Date
Status
Transaction Type
Reference
Description
Details
Amount (RM)
Wallet Balance
3/5/2026
Success
DUITNOW_RECEIVEFROM
111111
Bob

RM100.00
RM105.10`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("100.00"))
	})

	It("parses multiple transactions", func() {
		text := `TNG WALLET TRANSACTION
Date
Status
Transaction Type
Reference
Description
Details
Amount (RM)
Wallet Balance
1/5/2026
Success
Payment
111111
Merchant A

RM34.00
RM50.00
2/5/2026
Success
Reload
222222
Top Up

RM100.00
RM150.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(2))
		Expect(reports[0].Amount).To(Equal("-34.00"))
		Expect(reports[1].Amount).To(Equal("100.00"))
	})

	It("returns error for text without transaction section", func() {
		_, err := provider.ParsePDFText(ctx, "random text")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no transactions section found"))
	})

	It("returns empty for text with header but no transactions", func() {
		text := `TNG WALLET TRANSACTION`
		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(BeEmpty())
	})

	It("handles date with single-digit day", func() {
		text := `TNG WALLET TRANSACTION
Date
Status
Transaction Type
Reference
Description
Details
Amount (RM)
Wallet Balance
1/5/2026
Success
Payment
111111
Test Merchant

RM25.50
RM100.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Date).To(Equal("2026-05-01"))
	})

	It("handles date with double-digit day and month", func() {
		text := `TNG WALLET TRANSACTION
Date
Status
Transaction Type
Reference
Description
Details
Amount (RM)
Wallet Balance
01/12/2026
Success
Reload
111111
Salary

RM1000.00
RM2000.00`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Date).To(Equal("2026-12-01"))
	})

	It("extracts description without trailing reference noise", func() {
		text := `TNG WALLET TRANSACTION
Date
Status
Transaction Type
Reference
Description
Details
Amount (RM)
Wallet Balance
1/5/2026
Success
Payment
111111222222
Ninja Cat Cafe 11111111111111111111
33333
RM34.00
RM5.10`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Payee).To(BeEmpty())
	})

	It("parses DuitNow QR transaction type", func() {
		text := `TNG WALLET TRANSACTION
Date
Status
Transaction Type
Reference
Description
Details
Amount (RM)
Wallet Balance
1/5/2026
Success
DuitNow QR TNGD
202605011111111111111111111111111111
Merchant A

RM34.00
RM5.10`

		reports, err := provider.ParsePDFText(ctx, text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("-34.00"))
	})
})
