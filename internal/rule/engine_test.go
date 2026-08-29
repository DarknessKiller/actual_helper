package rule_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"actual_helper/internal/models"
	"actual_helper/internal/rule"
)

var _ = Describe("Engine", func() {
	Describe("ShouldSkip", func() {
		It("returns true when exclude keyword matches", func() {
			e := rule.NewEngine([]string{"Quick Reload"}, nil, nil, nil)
			Expect(e.ShouldSkip("Quick Reload Payment")).To(BeTrue())
		})

		It("returns false when no keyword matches", func() {
			e := rule.NewEngine([]string{"Quick Reload"}, nil, nil, nil)
			Expect(e.ShouldSkip("GrabFood Order")).To(BeFalse())
		})

		It("include keyword whitelist keeps matching row despite matching exclude", func() {
			e := rule.NewEngine(
				[]string{"Daily Interest"},
				[]string{"Daily Interest"},
				nil, nil,
			)
			Expect(e.ShouldSkip("Daily Interest earned")).To(BeFalse())
		})

		It("matches case-insensitively", func() {
			e := rule.NewEngine([]string{"quick reload"}, nil, nil, nil)
			Expect(e.ShouldSkip("QUICK RELOAD PAYMENT")).To(BeTrue())
		})

		It("returns false for nil keywords", func() {
			e := rule.NewEngine(nil, nil, nil, nil)
			Expect(e.ShouldSkip("anything")).To(BeFalse())
		})

		It("include keyword whitelist keeps matching rows", func() {
			e := rule.NewEngine(nil, []string{"Grab"}, nil, nil)
			Expect(e.ShouldSkip("GrabFood Order")).To(BeFalse())
		})

		It("include keyword whitelist skips non-matching rows", func() {
			e := rule.NewEngine(nil, []string{"Grab"}, nil, nil)
			Expect(e.ShouldSkip("Shopee Order")).To(BeTrue())
		})

		It("include keyword overrides exclude in whitelist mode", func() {
			e := rule.NewEngine(
				[]string{"Grab"},
				[]string{"Grab"},
				nil, nil,
			)
			Expect(e.ShouldSkip("GrabFood Order")).To(BeFalse())
		})

		It("include keyword skips non-matching even when exclude would match", func() {
			e := rule.NewEngine(
				[]string{"Shopee"},
				[]string{"Grab"},
				nil, nil,
			)
			Expect(e.ShouldSkip("Shopee Order")).To(BeTrue())
		})

		It("empty include slice falls back to exclude logic", func() {
			e := rule.NewEngine([]string{"Grab"}, []string{}, nil, nil)
			Expect(e.ShouldSkip("GrabFood Order")).To(BeTrue())
			Expect(e.ShouldSkip("Shopee Order")).To(BeFalse())
		})
	})

	Describe("MatchCategory", func() {
		It("returns group and category on match", func() {
			e := rule.NewEngine(nil, nil, []models.CategoryRule{
				{Keyword: "grab", Group: "Food", Category: "Delivery"},
			}, nil)
			grp, cat := e.MatchCategory("GrabFood Order")
			Expect(grp).To(Equal("Food"))
			Expect(cat).To(Equal("Delivery"))
		})

		It("returns empty on no match", func() {
			e := rule.NewEngine(nil, nil, nil, nil)
			grp, cat := e.MatchCategory("Unknown")
			Expect(grp).To(BeEmpty())
			Expect(cat).To(BeEmpty())
		})

		It("first match wins", func() {
			e := rule.NewEngine(nil, nil, []models.CategoryRule{
				{Keyword: "grab", Group: "Food", Category: "Delivery"},
				{Keyword: "grab", Group: "Override", Category: "ShouldNotReach"},
			}, nil)
			grp, cat := e.MatchCategory("GrabFood")
			Expect(grp).To(Equal("Food"))
			Expect(cat).To(Equal("Delivery"))
		})

		It("matches case-insensitively", func() {
			e := rule.NewEngine(nil, nil, []models.CategoryRule{
				{Keyword: "GRAB", Group: "Food", Category: "Delivery"},
			}, nil)
			grp, cat := e.MatchCategory("grabfood")
			Expect(grp).To(Equal("Food"))
			Expect(cat).To(Equal("Delivery"))
		})
	})

	Describe("MapAccount", func() {
		It("returns mapped name when key exists", func() {
			e := rule.NewEngine(nil, nil, nil, map[string]string{"raw": "mapped"})
			Expect(e.MapAccount("raw")).To(Equal("mapped"))
		})

		It("returns original when key missing", func() {
			e := rule.NewEngine(nil, nil, nil, map[string]string{"raw": "mapped"})
			Expect(e.MapAccount("other")).To(Equal("other"))
		})

		It("returns original when mapping is nil", func() {
			e := rule.NewEngine(nil, nil, nil, nil)
			Expect(e.MapAccount("raw")).To(Equal("raw"))
		})
	})

	Describe("Reload", func() {
		It("replaces keywords and categories", func() {
			e := rule.NewEngine([]string{"old"}, nil, nil, nil)
			Expect(e.ShouldSkip("old")).To(BeTrue())

			e.Reload([]string{"new"}, nil, nil, nil)
			Expect(e.ShouldSkip("old")).To(BeFalse())
			Expect(e.ShouldSkip("new")).To(BeTrue())
		})

		It("replaces account mapping", func() {
			e := rule.NewEngine(nil, nil, nil, map[string]string{"a": "A"})
			Expect(e.MapAccount("a")).To(Equal("A"))

			e.Reload(nil, nil, nil, map[string]string{"a": "AA"})
			Expect(e.MapAccount("a")).To(Equal("AA"))
		})
	})
})
