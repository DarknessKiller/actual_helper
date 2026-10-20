package config_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"actual_helper/internal/config"
	"actual_helper/internal/models"
)

var _ = Describe("Loader", func() {
	Describe("ProviderConfig", func() {
		It("returns empty when nothing is loaded", func() {
			loader := config.NewLoader()
			pc := loader.ProviderConfig("tng")

			Expect(pc.ExcludeKeywords).To(BeEmpty())
			Expect(pc.IncludeKeywords).To(BeEmpty())
			Expect(pc.Categories).To(BeEmpty())
			Expect(pc.AccountMappings).To(BeEmpty())
		})

		It("returns merged global and provider config after ApplyConfig", func() {
			loader := config.NewLoader()
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
						"categories": [{"keyword": "grab", "group": "Food", "category": "Delivery"}],
						"account_mappings": {"": "TNG"}
					}
				}
			}`

			Expect(loader.ApplyConfig([]byte(content))).To(Succeed())

			pc := loader.ProviderConfig("tng")
			Expect(pc.ExcludeKeywords).To(ConsistOf("Global Noise", "TNG Fee"))
			Expect(pc.IncludeKeywords).To(ConsistOf("Global Include", "TNG Include"))
			Expect(pc.Categories).To(HaveLen(2))
			Expect(pc.AccountMappings).To(Equal(map[string]string{"": "TNG"}))
		})

		It("uses only global when no provider section exists", func() {
			loader := config.NewLoader()
			Expect(loader.ApplyConfig([]byte(`{"global":{"exclude_keywords":["Global Only"]}}`))).To(Succeed())

			pc := loader.ProviderConfig("tng")
			Expect(pc.ExcludeKeywords).To(ConsistOf("Global Only"))
		})

		It("uses only provider when no global section exists", func() {
			loader := config.NewLoader()
			Expect(loader.ApplyConfig([]byte(`{"providers":{"tng":{"exclude_keywords":["Provider Only"]}}}`))).To(Succeed())

			pc := loader.ProviderConfig("tng")
			Expect(pc.ExcludeKeywords).To(ConsistOf("Provider Only"))
		})

		It("returns empty for an unknown provider while global is still merged", func() {
			loader := config.NewLoader()
			Expect(loader.ApplyConfig([]byte(`{"global":{"exclude_keywords":["Global Only"]}}`))).To(Succeed())

			pc := loader.ProviderConfig("unknown")
			Expect(pc.ExcludeKeywords).To(ConsistOf("Global Only"))
		})

		It("rejects invalid JSON and leaves the loader empty", func() {
			loader := config.NewLoader()
			err := loader.ApplyConfig([]byte("{invalid"))
			Expect(err).To(HaveOccurred())

			pc := loader.ProviderConfig("tng")
			Expect(pc.ExcludeKeywords).To(BeEmpty())
		})

		It("returns empty again after ClearConfig", func() {
			loader := config.NewLoader()
			Expect(loader.ApplyConfig([]byte(`{"global":{"exclude_keywords":["Noise"]}}`))).To(Succeed())
			Expect(loader.ProviderConfig("tng").ExcludeKeywords).To(ConsistOf("Noise"))

			loader.ClearConfig()

			pc := loader.ProviderConfig("tng")
			Expect(pc.ExcludeKeywords).To(BeEmpty())
			Expect(pc.IncludeKeywords).To(BeEmpty())
			Expect(pc.Categories).To(BeEmpty())
		})
	})

	Describe("SessionConfig", func() {
		var sessionBody = []byte(`{"global":{"exclude_keywords":["SI Noise"]},"providers":{"tng":{"exclude_keywords":["SI TNG Noise"]}}}`)

		It("returns empty tuning for an unknown session", func() {
			loader := config.NewLoader()

			pc := loader.SessionProviderConfig("no-such-sid", "tng")

			Expect(pc.ExcludeKeywords).To(BeEmpty())
		})

		It("returns session rules after ApplySessionConfig", func() {
			loader := config.NewLoader()
			Expect(loader.ApplySessionConfig("sid-a", sessionBody)).To(Succeed())

			pc := loader.SessionProviderConfig("sid-a", "tng")
			Expect(pc.ExcludeKeywords).To(ConsistOf("SI Noise", "SI TNG Noise"))
		})

		It("keeps sessions isolated", func() {
			loader := config.NewLoader()
			Expect(loader.ApplySessionConfig("sid-a", sessionBody)).To(Succeed())

			pc := loader.SessionProviderConfig("sid-b", "tng")
			Expect(pc.ExcludeKeywords).To(BeEmpty())
		})

		It("expires sessions lazily with no goroutine", func() {
			loader := config.NewLoader()
			loader.SetSessionTTL(-1 * time.Second)
			Expect(loader.ApplySessionConfig("sid-a", sessionBody)).To(Succeed())

			pc := loader.SessionProviderConfig("sid-a", "tng")
			Expect(pc.ExcludeKeywords).To(BeEmpty())
		})

		It("rejects new sessions when the store is full", func() {
			loader := config.NewLoader()
			loader.SessionLimit(2)

			Expect(loader.ApplySessionConfig("s1", sessionBody)).To(Succeed())
			Expect(loader.ApplySessionConfig("s2", sessionBody)).To(Succeed())
			err := loader.ApplySessionConfig("s3", sessionBody)
			Expect(err).To(MatchError(config.ErrTooManySessions))
		})

		It("prunes expired sessions on the next write", func() {
			loader := config.NewLoader()
			loader.SetSessionTTL(-1 * time.Second)
			loader.SessionLimit(1)
			Expect(loader.ApplySessionConfig("stale", sessionBody)).To(Succeed())

			Expect(loader.ApplySessionConfig("fresh", sessionBody)).To(Succeed())
		})

		It("clears one session only on ClearSession", func() {
			loader := config.NewLoader()
			Expect(loader.ApplySessionConfig("sid-a", sessionBody)).To(Succeed())
			Expect(loader.ApplySessionConfig("sid-b", sessionBody)).To(Succeed())

			loader.ClearSession("sid-a")

			Expect(loader.SessionProviderConfig("sid-a", "tng").ExcludeKeywords).To(BeEmpty())
			Expect(loader.SessionProviderConfig("sid-b", "tng").ExcludeKeywords).To(ConsistOf("SI Noise", "SI TNG Noise"))
		})

		It("keeps a previous config on invalid JSON", func() {
			loader := config.NewLoader()
			Expect(loader.ApplySessionConfig("sid-a", sessionBody)).To(Succeed())

			Expect(loader.ApplySessionConfig("sid-a", []byte(`not json`))).To(HaveOccurred())
			Expect(loader.SessionProviderConfig("sid-a", "tng").ExcludeKeywords).To(ConsistOf("SI Noise", "SI TNG Noise"))
		})
	})

	Describe("SampleConfig", func() {
		It("returns an error when path is empty", func() {
			_, err := config.SampleConfig("")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error for a missing file", func() {
			_, err := config.SampleConfig("/no/such/path/config.json")
			Expect(err).To(HaveOccurred())
		})

		It("returns the file bytes when readable", func() {
			path := writeTempConfig(`{"global":{"exclude_keywords":["Sample"]}}`)
			data, err := config.SampleConfig(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(MatchJSON(`{"global":{"exclude_keywords":["Sample"]}}`))
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
