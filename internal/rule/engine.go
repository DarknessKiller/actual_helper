package rule

import (
	"strings"
	"sync"

	"actual_helper/internal/models"
)

type compiledCategoryRule struct {
	lowerKeyword string
	Group        string
	Category     string
}

type Engine struct {
	excludeKeywords []string
	includeKeywords []string
	categories      []compiledCategoryRule
	mu              sync.RWMutex
}

func NewEngine(excludeKeywords, includeKeywords []string, categories []models.CategoryRule) *Engine {
	compiled := make([]compiledCategoryRule, len(categories))
	for i, c := range categories {
		compiled[i] = compiledCategoryRule{lowerKeyword: strings.ToLower(c.Keyword), Group: c.Group, Category: c.Category}
	}
	return &Engine{
		excludeKeywords: lowerSlice(excludeKeywords),
		includeKeywords: lowerSlice(includeKeywords),
		categories:      compiled,
	}
}

func (e *Engine) ShouldSkip(description string) bool {
	skip, _, _ := e.Evaluate(description)
	return skip
}

func (e *Engine) MatchCategory(description string) (string, string) {
	_, group, category := e.Evaluate(description)
	return group, category
}

// Evaluate performs skip check and category match in a single lock acquisition.
func (e *Engine) Evaluate(description string) (skip bool, group, category string) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	lower := strings.ToLower(description)

	if len(e.includeKeywords) > 0 {
		found := false
		for _, kw := range e.includeKeywords {
			if strings.Contains(lower, kw) {
				found = true
				break
			}
		}
		if !found {
			return true, "", ""
		}
	} else {
		for _, kw := range e.excludeKeywords {
			if strings.Contains(lower, kw) {
				return true, "", ""
			}
		}
	}

	for _, r := range e.categories {
		if strings.Contains(lower, r.lowerKeyword) {
			return false, r.Group, r.Category
		}
	}
	return false, "", ""
}

func lowerSlice(s []string) []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	for i, v := range s {
		out[i] = strings.ToLower(v)
	}
	return out
}
