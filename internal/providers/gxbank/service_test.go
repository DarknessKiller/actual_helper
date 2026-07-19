package gxbank_test

import (
	"context"
	"strings"

	"actual_helper/internal/pdfutil"
	"actual_helper/internal/providers"
	gxbankprov "actual_helper/internal/providers/gxbank"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// digitalText builds a minimal digital-extracted text block for tests.
func digitalText(account, txLines string) string {
	return `Statements of Accounts
` + account + `

January 2026
Closing balance (RM)
Baki penutup
` + txLines + `
Note/Perhatian
1. Some note`
}

var _ = Describe("GXBankProvider", func() {
	Describe("Name", func() {
		It("returns gxbank", func() {
			provider := gxbankprov.New(nil, nil, nil, nil)
			Expect(provider.Name()).To(Equal("gxbank"))
		})
	})

	Describe("ExtractionMethod", func() {
		It("returns digital", func() {
			provider := gxbankprov.New(nil, nil, nil, nil)
			Expect(provider.ExtractionMethod()).To(Equal(pdfutil.ExtractionMethodDigital))
		})
	})

	Describe("ParseCSV", func() {
		It("returns error because gxbank only supports PDF", func() {
			provider := gxbankprov.New(nil, nil, nil, nil)
			_, err := provider.ParseCSV(context.Background(), strings.NewReader("Date,Description,Amount\n1 Jan 2026,Test,-10.00"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not supported"))
		})
	})

	Describe("ParsePDFText full flow", func() {
		It("parses transactions into ActualBudgetReport", func() {
			provider := gxbankprov.New(nil, nil, nil, nil)
			text := digitalText("GX Savings Account", `1 January 2026
12:00 AM
Opening balance
0.00
1 January 2026
12:00 PM
Interest Earned
+0.55
0.55
5 January 2026
3:00 PM
Payment to Merchant
-25.00
-24.45
10 January 2026
12:00 PM
Receive from Friend
+100.00
75.55`)

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(3))

			Expect(reports[0].Account).To(Equal("GX Savings Account"))
			Expect(reports[0].Date).To(Equal("2026-01-01"))
			Expect(reports[0].Notes).To(Equal("Interest Earned"))
			Expect(reports[0].Amount).To(Equal("0.55"))

			Expect(reports[1].Date).To(Equal("2026-01-05"))
			Expect(reports[1].Notes).To(Equal("Payment to Merchant"))
			Expect(reports[1].Amount).To(Equal("-25.00"))

			Expect(reports[2].Date).To(Equal("2026-01-10"))
			Expect(reports[2].Notes).To(Equal("Receive from Friend"))
			Expect(reports[2].Amount).To(Equal("100.00"))
		})

		It("skips opening balance entries", func() {
			provider := gxbankprov.New(nil, nil, nil, nil)
			text := digitalText("GX Savings Account", `1 January 2026
12:00 AM
Opening Balance
1000.00
5 January 2026
12:00 PM
Interest Earned
+0.55
1000.55`)

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Notes).To(Equal("Interest Earned"))
		})

		It("handles amounts with commas", func() {
			provider := gxbankprov.New(nil, nil, nil, nil)
			text := digitalText("GX Savings Account", `1 January 2026
12:00 AM
Opening balance
0.00
1 January 2026
12:00 PM
Large Deposit
+10,097.90
10,097.90`)

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Amount).To(Equal("10097.90"))
		})

		It("joins multi-line descriptions", func() {
			provider := gxbankprov.New(nil, nil, nil, nil)
			text := digitalText("GX Savings Account", `1 January 2026
12:00 AM
Opening balance
0.00
1 January 2026
12:00 PM
Pocket
Withdraw from Pocket
-50.00
-50.00`)

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Notes).To(Equal("Pocket Withdraw from Pocket"))
		})

		It("skips time lines", func() {
			provider := gxbankprov.New(nil, nil, nil, nil)
			text := digitalText("GX Savings Account", `1 January 2026
12:00 AM
Opening balance
0.00
1 January 2026
12:00 PM
Interest Earned
+0.55
0.55`)

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Notes).To(Equal("Interest Earned"))
		})
	})

	Describe("ParsePDFText with account mapping", func() {
		It("maps account name from config", func() {
			accountMappings := map[string]string{
				"GX Savings Account": "My GX Savings",
			}
			provider := gxbankprov.New(nil, nil, nil, accountMappings)
			text := digitalText("GX Savings Account", `1 January 2026
12:00 AM
Opening balance
0.00
1 January 2026
12:00 PM
Interest Earned
+0.55
0.55`)

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Account).To(Equal("My GX Savings"))
		})

		It("falls back to extracted name when no mapping match", func() {
			accountMappings := map[string]string{
				"Other Account": "Mapped Name",
			}
			provider := gxbankprov.New(nil, nil, nil, accountMappings)
			text := digitalText("GX Savings Account", `1 January 2026
12:00 AM
Opening balance
0.00
1 January 2026
12:00 PM
Interest Earned
+0.55
0.55`)

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Account).To(Equal("GX Savings Account"))
		})

		It("falls back to GX Bank when no Statements of Accounts header", func() {
			provider := gxbankprov.New(nil, nil, nil, nil)
			text := `January 2026
Closing balance (RM)
Baki penutup
1 January 2026
12:00 PM
Interest Earned
+0.55
0.55
Note/Perhatian
1. Some note`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Account).To(Equal("GX Bank"))
		})
	})

	Describe("ParsePDFText with filtering", func() {
		It("skips rows matching exclude keywords", func() {
			provider := gxbankprov.New([]string{"Interest"}, nil, nil, nil)
			text := digitalText("GX Savings Account", `1 January 2026
12:00 AM
Opening balance
0.00
1 January 2026
12:00 PM
Interest Earned
+0.55
0.55
5 January 2026
3:00 PM
Payment to Merchant
-25.00
-24.45`)

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Notes).To(Equal("Payment to Merchant"))
		})
	})

	Describe("Reload", func() {
		It("updates account mapping", func() {
			prov := gxbankprov.New(nil, nil, nil, nil)
			configurable, ok := prov.(providers.ConfigurableProvider)
			Expect(ok).To(BeTrue())

			text := digitalText("GX Savings Account", `1 January 2026
12:00 AM
Opening balance
0.00
1 January 2026
12:00 PM
Interest Earned
+0.55
0.55`)

			reports, err := prov.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports[0].Account).To(Equal("GX Savings Account"))

			configurable.Reload(nil, nil, nil, map[string]string{
				"GX Savings Account": "Updated Name",
			})

			reports, err = prov.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports[0].Account).To(Equal("Updated Name"))
		})

		It("updates exclude keywords", func() {
			prov := gxbankprov.New(nil, nil, nil, nil)
			configurable, ok := prov.(providers.ConfigurableProvider)
			Expect(ok).To(BeTrue())

			text := digitalText("GX Savings Account", `1 January 2026
12:00 AM
Opening balance
0.00
1 January 2026
12:00 PM
Interest Earned
+0.55
0.55
5 January 2026
3:00 PM
Payment to Merchant
-25.00
-24.45`)

			reports, err := prov.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(2))

			configurable.Reload([]string{"Interest"}, nil, nil, nil)

			reports, err = prov.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Notes).To(Equal("Payment to Merchant"))
		})
	})
})
