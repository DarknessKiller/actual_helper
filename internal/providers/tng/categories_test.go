package tng_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"actual-helper/internal/providers/tng"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var ctx = context.Background()

var _ = Describe("Categories", func() {
	Describe("with a rules file", func() {
		It("applies category from matching keyword", func() {
			tmpDir := GinkgoT().TempDir()
			tmpFile := filepath.Join(tmpDir, "categories.json")
			os.WriteFile(tmpFile, []byte(`[
				{"keyword":"grab","group":"Food & Dining","category":"Delivery"}
			]`), 0644)

			GinkgoT().Setenv("TNG_CATEGORIES_PATH", tmpFile)
			categorizingProvider := tng.New()

			csv := "F,Status,Transaction Type,Reference,Description,Details,Amount(RM)\n" +
				"13/6/2026,Success,Purchase,TXN001,GrabFood Order,Test,25.50\n"

			reports, err := categorizingProvider.ParseCSV(ctx, strings.NewReader(csv))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].CategoryGroup).To(Equal("Food & Dining"))
			Expect(reports[0].Category).To(Equal("Delivery"))
		})

		It("matches case-insensitively", func() {
			tmpDir := GinkgoT().TempDir()
			tmpFile := filepath.Join(tmpDir, "categories.json")
			os.WriteFile(tmpFile, []byte(`[
				{"keyword":"grab","group":"Food","category":"Delivery"}
			]`), 0644)

			GinkgoT().Setenv("TNG_CATEGORIES_PATH", tmpFile)
			categorizingProvider := tng.New()

			csv := "F,Status,Transaction Type,Reference,Description,Details,Amount(RM)\n" +
				"13/6/2026,Success,Purchase,TXN001,GRABFOOD ORDER,Test,25.50\n"

			reports, err := categorizingProvider.ParseCSV(ctx, strings.NewReader(csv))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports[0].CategoryGroup).To(Equal("Food"))
		})

		It("first match wins", func() {
			tmpDir := GinkgoT().TempDir()
			tmpFile := filepath.Join(tmpDir, "categories.json")
			os.WriteFile(tmpFile, []byte(`[
				{"keyword":"grab","group":"Food","category":"Delivery"},
				{"keyword":"grab","group":"Transport","category":"Ride"}
			]`), 0644)

			GinkgoT().Setenv("TNG_CATEGORIES_PATH", tmpFile)
			categorizingProvider := tng.New()

			csv := "F,Status,Transaction Type,Reference,Description,Details,Amount(RM)\n" +
				"13/6/2026,Success,Purchase,TXN001,GrabCar Ride,Test,25.50\n"

			reports, err := categorizingProvider.ParseCSV(ctx, strings.NewReader(csv))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports[0].CategoryGroup).To(Equal("Food"))
		})

		It("leaves category empty when no rule matches", func() {
			tmpDir := GinkgoT().TempDir()
			tmpFile := filepath.Join(tmpDir, "categories.json")
			os.WriteFile(tmpFile, []byte(`[
				{"keyword":"grab","group":"Food","category":"Delivery"}
			]`), 0644)

			GinkgoT().Setenv("TNG_CATEGORIES_PATH", tmpFile)
			categorizingProvider := tng.New()

			csv := "F,Status,Transaction Type,Reference,Description,Details,Amount(RM)\n" +
				"13/6/2026,Success,Purchase,TXN001,7-Eleven KLIA,Test,12.80\n"

			reports, err := categorizingProvider.ParseCSV(ctx, strings.NewReader(csv))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports[0].CategoryGroup).To(BeEmpty())
			Expect(reports[0].Category).To(BeEmpty())
		})

		It("leaves category empty when description is empty", func() {
			tmpDir := GinkgoT().TempDir()
			tmpFile := filepath.Join(tmpDir, "categories.json")
			os.WriteFile(tmpFile, []byte(`[
				{"keyword":"grab","group":"Food","category":"Delivery"}
			]`), 0644)

			GinkgoT().Setenv("TNG_CATEGORIES_PATH", tmpFile)
			categorizingProvider := tng.New()

			csv := "F,Status,Transaction Type,Reference,Description,Details,Amount(RM)\n" +
				"13/6/2026,Success,Purchase,TXN001,,Test,\n"

			reports, err := categorizingProvider.ParseCSV(ctx, strings.NewReader(csv))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(BeEmpty())
		})

		It("loads rules from valid JSON file", func() {
			tmpDir := GinkgoT().TempDir()
			tmpFile := filepath.Join(tmpDir, "categories.json")
			os.WriteFile(tmpFile, []byte(`[
				{"keyword":"grab","group":"Food","category":"Delivery"},
				{"keyword":"shopee","group":"Shopping","category":"Online"}
			]`), 0644)

			GinkgoT().Setenv("TNG_CATEGORIES_PATH", tmpFile)
			categorizingProvider := tng.New()

			csv := "F,Status,Transaction Type,Reference,Description,Details,Amount(RM)\n" +
				"13/6/2026,Success,Purchase,TXN001,GrabFood,Test,25.50\n" +
				"14/6/2026,Success,Purchase,TXN002,Shopee Mall,Test,100.00\n"

			reports, err := categorizingProvider.ParseCSV(ctx, strings.NewReader(csv))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(2))
		})

		It("works when no env var is set", func() {
			defaultProvider := tng.New()
			csv := "F,Status,Transaction Type,Reference,Description,Details,Amount(RM)\n" +
				"13/6/2026,Success,Purchase,TXN001,GrabFood,Test,25.50\n"

			reports, err := defaultProvider.ParseCSV(ctx, strings.NewReader(csv))
			Expect(err).NotTo(HaveOccurred())
			Expect(reports).To(HaveLen(1))
			Expect(reports[0].CategoryGroup).To(BeEmpty())
		})
	})
})
