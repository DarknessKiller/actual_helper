package config_test

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"actual-helper/internal/config"
	"actual-helper/internal/models"
)

var _ = Describe("Loader", func() {
	Describe("ProviderConfig", func() {
		It("returns merged global and provider config", func() {
			tmpDir := GinkgoT().TempDir()
			path := filepath.Join(tmpDir, "config.json")
			content := `{
				"global": {
					"exclude_keywords": ["Global Noise"],
					"include_keywords": ["Global Include"],
					"categories": [{"keyword": "shopee", "group": "Shopping", "category": "Online"}]
				},
				"providers": {
					"tng": {
						"exclude_keywords": ["TNG Fee"],
						"include_keywords": ["TNG Include"],
						"categories": [{"keyword": "grab", "group": "Food", "category": "Delivery"}]
					}
				}
			}`
			Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())

			loader := config.NewLoader(path)
			pc := loader.ProviderConfig("tng")

			Expect(pc.ExcludeKeywords).To(ConsistOf("Global Noise", "TNG Fee"))
			Expect(pc.IncludeKeywords).To(ConsistOf("Global Include", "TNG Include"))
			Expect(pc.Categories).To(HaveLen(2))
		})

		It("uses only global when no provider section exists", func() {
			tmpDir := GinkgoT().TempDir()
			path := filepath.Join(tmpDir, "config.json")
			content := `{"global":{"exclude_keywords":["Global Only"]}}`
			Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())

			loader := config.NewLoader(path)
			pc := loader.ProviderConfig("tng")

			Expect(pc.ExcludeKeywords).To(ConsistOf("Global Only"))
		})

		It("uses only provider when no global section exists", func() {
			tmpDir := GinkgoT().TempDir()
			path := filepath.Join(tmpDir, "config.json")
			content := `{"providers":{"tng":{"exclude_keywords":["Provider Only"]}}}`
			Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())

			loader := config.NewLoader(path)
			pc := loader.ProviderConfig("tng")

			Expect(pc.ExcludeKeywords).To(ConsistOf("Provider Only"))
		})

		It("returns empty ProviderConfig when file is missing", func() {
			tmpDir := GinkgoT().TempDir()
			loader := config.NewLoader(filepath.Join(tmpDir, "nonexistent.json"))
			pc := loader.ProviderConfig("tng")

			Expect(pc.ExcludeKeywords).To(BeEmpty())
			Expect(pc.IncludeKeywords).To(BeEmpty())
			Expect(pc.Categories).To(BeEmpty())
		})

		It("returns empty ProviderConfig when path is empty", func() {
			loader := config.NewLoader("")
			pc := loader.ProviderConfig("tng")

			Expect(pc.ExcludeKeywords).To(BeEmpty())
			Expect(pc.IncludeKeywords).To(BeEmpty())
			Expect(pc.Categories).To(BeEmpty())
		})

		It("returns empty ProviderConfig on invalid JSON", func() {
			tmpDir := GinkgoT().TempDir()
			path := filepath.Join(tmpDir, "config.json")
			Expect(os.WriteFile(path, []byte("{invalid}"), 0644)).To(Succeed())

			loader := config.NewLoader(path)
			pc := loader.ProviderConfig("tng")

			Expect(pc.ExcludeKeywords).To(BeEmpty())
		})

		It("reloads when file mtime changes", func() {
			tmpDir := GinkgoT().TempDir()
			path := filepath.Join(tmpDir, "config.json")

			initial := `{"global":{"exclude_keywords":["Old"]}}`
			Expect(os.WriteFile(path, []byte(initial), 0644)).To(Succeed())

			loader := config.NewLoader(path)

			pc := loader.ProviderConfig("tng")
			Expect(pc.ExcludeKeywords).To(ConsistOf("Old"))

			updated := `{"global":{"exclude_keywords":["New"]}}`
			Expect(os.WriteFile(path, []byte(updated), 0644)).To(Succeed())

			// Ensure mtime differs
			time.Sleep(100 * time.Millisecond)

			pc = loader.ProviderConfig("tng")
			Expect(pc.ExcludeKeywords).To(ConsistOf("New"))
		})
	})

	Describe("CategoryRule model", func() {
		It("uses models.CategoryRule", func() {
			rule := models.CategoryRule{
				Keyword:  "grab",
				Group:    "Food",
				Category: "Delivery",
			}
			Expect(rule.Keyword).To(Equal("grab"))
			Expect(rule.Group).To(Equal("Food"))
			Expect(rule.Category).To(Equal("Delivery"))
		})
	})
})
