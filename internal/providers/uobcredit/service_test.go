package uobcredit_test

import (
	"context"
	"strings"

	uobcreditprov "actual_helper/internal/providers/uobcredit"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UOBProvider", func() {
	Describe("Name", func() {
		It("returns uobcredit", func() {
			provider := uobcreditprov.New(nil, nil, nil, nil)
			Expect(provider.Name()).To(Equal("uobcredit"))
		})
	})

	Describe("ParseCSV", func() {
		It("returns error because uobcredit only supports PDF", func() {
			provider := uobcreditprov.New(nil, nil, nil, nil)
			_, err := provider.ParseCSV(context.Background(), strings.NewReader("a,b,c\n1,2,3"))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ParsePDFText with account mapping", func() {
		It("maps account name from config using card number", func() {
			accountMappings := map[string]string{
				"1234 5678 9012 3456": "UOB Credit",
			}
			provider := uobcreditprov.New(nil, nil, nil, accountMappings)
			text := `WORLD MASTERCARD              1234-5678-9012-3456
Statement Date    16 JUL 2026
          15 JUL              PURCHASE AT MERCHANT ABC                                                                                45.90`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Account).To(Equal("UOB Credit"))
		})

		It("falls back to UOB Credit Card when no card number in PDF", func() {
			provider := uobcreditprov.New(nil, nil, nil, nil)
			text := `Statement Date    16 JUL 2026
          15 JUL              PURCHASE AT MERCHANT ABC                                                                                45.90`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Account).To(Equal("UOB Credit Card"))
		})
	})

	Describe("ParsePDFText with filtering", func() {
		It("skips rows matching exclude keywords", func() {
			provider := uobcreditprov.New([]string{"SUBSCRIPTION"}, nil, nil, nil)
			text := `Statement Date    16 JUL 2026
          15 JUL              PURCHASE AT MERCHANT ABC                                                                                45.90
          16 JUL              ONLINE SUBSCRIPTION                                                                29.90`

			reports, err := provider.ParsePDFText(context.Background(), text)
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].Notes).To(Equal("PURCHASE AT MERCHANT ABC"))
		})
	})
})
