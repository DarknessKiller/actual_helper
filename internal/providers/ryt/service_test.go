package ryt_test

import (
	"context"
	"strings"

	rytprov "actual_helper/internal/providers/ryt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RytProvider", func() {
	Describe("Name", func() {
		It("returns ryt", func() {
			provider := rytprov.New(nil, nil, nil, nil)
			Expect(provider.Name()).To(Equal("ryt"))
		})
	})

	Describe("ParseCSV", func() {
		It("returns error because ryt only supports PDF", func() {
			provider := rytprov.New(nil, nil, nil, nil)
			_, err := provider.ParseCSV(context.Background(), strings.NewReader("a,b,c\n1,2,3"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("CSV not supported"))
		})
	})

	Describe("ParsePDFText with account mapping", func() {
		It("maps account name from config", func() {
			accountMappings := map[string]string{
				"Main Account": "Ryt Bank Checking",
			}
			provider := rytprov.New(nil, nil, nil, accountMappings)
			text := `Main Account Statement

Account Transactions / Transaksi Akaun
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
Ref. ID: F20260501ABCDEF1
+783.88
784.14`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Account).To(Equal("Ryt Bank Checking"))
		})

		It("falls back to extracted account name when no mapping exists", func() {
			provider := rytprov.New(nil, nil, nil, nil)
			text := `Main Account Statement

Account Transactions / Transaksi Akaun
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
Ref. ID: F20260501ABCDEF1
+783.88
784.14`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Account).To(Equal("Main Account"))
		})
	})
})
