package hsbccredit_test

import (
	"context"
	"strings"

	hsbccreditprov "actual_helper/internal/providers/hsbccredit"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HSBCProvider", func() {
	Describe("Name", func() {
		It("returns hsbccredit", func() {
			provider := hsbccreditprov.New(nil, nil, nil, nil)
			Expect(provider.Name()).To(Equal("hsbccredit"))
		})
	})

	Describe("ParseCSV", func() {
		It("returns error because hsbccredit only supports PDF", func() {
			provider := hsbccreditprov.New(nil, nil, nil, nil)
			_, err := provider.ParseCSV(context.Background(), strings.NewReader("a,b,c\n1,2,3"))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ParsePDFText with account mapping", func() {
		It("maps account name from config using card number", func() {
			accountMappings := map[string]string{
				"1234 5678 9012 3456": "Current",
			}
			provider := hsbccreditprov.New(nil, nil, nil, accountMappings)
			text := `Card Number : 1234 5678 9012 3456
Statement Date 04 Jun 2026
Post date | Transaction date | Transaction details | Amount (RM)
17 MAY 17 MAY PAYMENT - THANK YOU 259.72CR
Your charge(s) for this month RM259.72`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Account).To(Equal("Current"))
		})

		It("falls back to HSBC Credit Card when no card number in PDF", func() {
			provider := hsbccreditprov.New(nil, nil, nil, nil)
			text := `Statement Date 04 Jun 2026
Post date | Transaction date | Transaction details | Amount (RM)
17 MAY 17 MAY PAYMENT - THANK YOU 259.72CR
Your charge(s) for this month RM259.72`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Account).To(Equal("HSBC Credit Card"))
		})
	})

	Describe("ParsePDFText with filtering", func() {
		It("skips rows matching exclude keywords", func() {
			provider := hsbccreditprov.New([]string{"Grab"}, nil, nil, nil)
			text := `Statement Date 04 Jun 2026
Post date | Transaction date | Transaction details | Amount (RM)
06 MAY 05 MAY Grab Car Ride 8.50
10 MAY 07 MAY Shopee Purchase 4.00`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Notes).To(Equal("Shopee Purchase"))
		})
	})
})
