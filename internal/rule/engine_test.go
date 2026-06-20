package rule_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"actual-helper/internal/models"
	"actual-helper/internal/rule"
)

var _ = Describe("Engine", func() {
	Describe("ShouldSkip", func() {
		It("returns true when exclude keyword matches", func() {
			e := rule.NewEngine([]string{"Quick Reload"}, nil, nil)
			Expect(e.ShouldSkip("Quick Reload Payment")).To(BeTrue())
		})

		It("returns false when no keyword matches", func() {
			e := rule.NewEngine([]string{"Quick Reload"}, nil, nil)
			Expect(e.ShouldSkip("GrabFood Order")).To(BeFalse())
		})

		It("include keyword overrides exclude", func() {
			e := rule.NewEngine(
				[]string{"Daily Interest"},
				[]string{"Daily Interest"},
				nil,
			)
			Expect(e.ShouldSkip("Daily Interest earned")).To(BeFalse())
		})

		It("matches case-insensitively", func() {
			e := rule.NewEngine([]string{"quick reload"}, nil, nil)
			Expect(e.ShouldSkip("QUICK RELOAD PAYMENT")).To(BeTrue())
		})

		It("returns false for nil keywords", func() {
			e := rule.NewEngine(nil, nil, nil)
			Expect(e.ShouldSkip("anything")).To(BeFalse())
		})
	})

	Describe("MatchCategory", func() {
		It("returns group and category on match", func() {
			e := rule.NewEngine(nil, nil, []models.CategoryRule{
				{Keyword: "grab", Group: "Food", Category: "Delivery"},
			})
			grp, cat := e.MatchCategory("GrabFood Order")
			Expect(grp).To(Equal("Food"))
			Expect(cat).To(Equal("Delivery"))
		})

		It("returns empty on no match", func() {
			e := rule.NewEngine(nil, nil, nil)
			grp, cat := e.MatchCategory("Unknown")
			Expect(grp).To(BeEmpty())
			Expect(cat).To(BeEmpty())
		})

		It("first match wins", func() {
			e := rule.NewEngine(nil, nil, []models.CategoryRule{
				{Keyword: "grab", Group: "Food", Category: "Delivery"},
				{Keyword: "grab", Group: "Override", Category: "ShouldNotReach"},
			})
			grp, cat := e.MatchCategory("GrabFood")
			Expect(grp).To(Equal("Food"))
			Expect(cat).To(Equal("Delivery"))
		})

		It("matches case-insensitively", func() {
			e := rule.NewEngine(nil, nil, []models.CategoryRule{
				{Keyword: "GRAB", Group: "Food", Category: "Delivery"},
			})
			grp, cat := e.MatchCategory("grabfood")
			Expect(grp).To(Equal("Food"))
			Expect(cat).To(Equal("Delivery"))
		})
	})

	Describe("Reload", func() {
		It("replaces keywords and categories", func() {
			e := rule.NewEngine([]string{"old"}, nil, nil)
			Expect(e.ShouldSkip("old")).To(BeTrue())

			e.Reload([]string{"new"}, nil, nil)
			Expect(e.ShouldSkip("old")).To(BeFalse())
			Expect(e.ShouldSkip("new")).To(BeTrue())
		})
	})
})
