package sample_test

import (
	"context"
	"strings"

	"actual_helper/internal/models"
	"actual_helper/internal/providers"
	"actual_helper/internal/providers/sample"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SampleProvider", func() {
	var (
		ctx      = context.Background()
		provider providers.Provider
	)

	BeforeEach(func() {
		provider = sample.New(nil, nil, nil, nil)
	})

	Describe("Name", func() {
		It("returns sample", func() {
			Expect(provider.Name()).To(Equal("sample"))
		})
	})

	Describe("ParseCSV", func() {
		It("returns deterministic sample rows echoing the header", func() {
			reports, err := provider.ParseCSV(ctx, strings.NewReader("Description,Amount\nGrocery,12.34\n"))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(2))
			Expect(reports[0].Account).To(Equal("Sample Account"))
			Expect(reports[0].Date).To(Equal("2026-01-01"))
			Expect(reports[0].Notes).To(Equal("Description Amount"))
			Expect(reports[0].Amount).To(Equal("-10.00"))
			Expect(reports[1].Date).To(Equal("2026-01-02"))
			Expect(reports[1].Notes).To(Equal("Description Amount (credit)"))
			Expect(reports[1].Amount).To(Equal("5.50"))
		})

		It("returns default sample rows for empty input", func() {
			reports, err := provider.ParseCSV(ctx, strings.NewReader(""))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(2))
			Expect(reports[0].Notes).To(Equal("Sample Merchant"))
		})
	})

	Describe("ParsePDFText", func() {
		It("uses the first non-empty line as the description", func() {
			reports, err := provider.ParsePDFText(ctx, "\n\nCoffee Shop\nmore text")
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(2))
			Expect(reports[0].Notes).To(Equal("Coffee Shop"))
			Expect(reports[1].Notes).To(Equal("Coffee Shop (credit)"))
		})

		It("returns default sample rows for empty text", func() {
			reports, err := provider.ParsePDFText(ctx, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(2))
			Expect(reports[0].Notes).To(Equal("Sample Merchant"))
		})
	})

	Describe("ExtractionMethod", func() {
		It("returns a non-OCR method", func() {
			Expect(string(provider.ExtractionMethod())).To(Equal("digital"))
		})
	})

	Describe("ConfigurableProvider", func() {
		It("implements the interface", func() {
			_, ok := provider.(providers.ConfigurableProvider)
			Expect(ok).To(BeTrue())
		})

		It("does not panic on Reload", func() {
			cp := provider.(providers.ConfigurableProvider)
			Expect(func() {
				cp.Reload([]string{"skip"}, nil, []models.CategoryRule{
					{Keyword: "shop", Group: "Shopping", Category: "Online"},
				}, map[string]string{"Sample Account": "Mapped Account"})
			}).NotTo(Panic())
		})

		It("applies exclude keywords after Reload", func() {
			cp := provider.(providers.ConfigurableProvider)
			cp.Reload([]string{"coffee"}, nil, nil, nil)
			reports, err := provider.ParsePDFText(ctx, "Coffee Shop")
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(BeEmpty())
		})

		It("applies category rules after Reload", func() {
			cp := provider.(providers.ConfigurableProvider)
			cp.Reload(nil, nil, []models.CategoryRule{
				{Keyword: "coffee", Group: "Food & Dining", Category: "Cafe"},
			}, nil)
			reports, err := provider.ParsePDFText(ctx, "Coffee Shop")
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(2))
			Expect(reports[0].CategoryGroup).To(Equal("Food & Dining"))
			Expect(reports[0].Category).To(Equal("Cafe"))
		})

		It("applies account mappings after Reload", func() {
			cp := provider.(providers.ConfigurableProvider)
			cp.Reload(nil, nil, nil, map[string]string{"Sample Account": "Mapped Account"})
			reports, err := provider.ParsePDFText(ctx, "Coffee Shop")
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(2))
			Expect(reports[0].Account).To(Equal("Mapped Account"))
		})
	})

	Describe("io.Closer", func() {
		It("closes without error", func() {
			closer, ok := provider.(interface{ Close() error })
			Expect(ok).To(BeTrue())
			Expect(closer.Close()).To(Succeed())
		})
	})
})
