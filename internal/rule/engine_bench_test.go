package rule_test

import (
	"testing"

	"actual_helper/internal/models"
	"actual_helper/internal/rule"
)

func BenchmarkShouldSkip(b *testing.B) {
	e := rule.NewEngine(
		[]string{"Quick Reload", "Daily Interest", "Shopee", "Grab", "TNG"},
		nil,
		nil,
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.ShouldSkip("Quick Reload Payment via TNG Wallet")
	}
}

func BenchmarkMatchCategory(b *testing.B) {
	categories := []models.CategoryRule{
		{Keyword: "shopee", Group: "Shopping", Category: "Online"},
		{Keyword: "grab", Group: "Food", Category: "Delivery"},
		{Keyword: "tng", Group: "Transport", Category: "E-Wallet"},
		{Keyword: "steam", Group: "Entertainment", Category: "Gaming"},
		{Keyword: "netflix", Group: "Entertainment", Category: "Streaming"},
	}
	e := rule.NewEngine(nil, nil, categories)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.MatchCategory("Shopee Order - 12345")
	}
}

func BenchmarkShouldSkipAndMatchCategory(b *testing.B) {
	e := rule.NewEngine(
		[]string{"Daily Interest"},
		nil,
		[]models.CategoryRule{
			{Keyword: "shopee", Group: "Shopping", Category: "Online"},
			{Keyword: "grab", Group: "Food", Category: "Delivery"},
		},
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.ShouldSkip("GrabFood Order #12345")
		e.MatchCategory("GrabFood Order #12345")
	}
}
