package rule

import (
	"strings"
	"sync"

	"actual_helper/internal/models"
)

type Engine struct {
	excludeKeywords []string
	includeKeywords []string
	categories      []models.CategoryRule
	mu              sync.RWMutex
}

func NewEngine(excludeKeywords, includeKeywords []string, categories []models.CategoryRule) *Engine {
	return &Engine{
		excludeKeywords: copySlice(excludeKeywords),
		includeKeywords: copySlice(includeKeywords),
		categories:      copyCategories(categories),
	}
}

func (e *Engine) Reload(excludeKeywords, includeKeywords []string, categories []models.CategoryRule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.excludeKeywords = copySlice(excludeKeywords)
	e.includeKeywords = copySlice(includeKeywords)
	e.categories = copyCategories(categories)
}

func (e *Engine) ShouldSkip(description string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	lower := strings.ToLower(description)

	if len(e.includeKeywords) > 0 {
		for _, kw := range e.includeKeywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				return false
			}
		}
		return true
	}

	for _, kw := range e.excludeKeywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

func (e *Engine) MatchCategory(description string) (string, string) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	lower := strings.ToLower(description)
	for _, r := range e.categories {
		if strings.Contains(lower, strings.ToLower(r.Keyword)) {
			return r.Group, r.Category
		}
	}
	return "", ""
}

func copySlice(s []string) []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)
	return out
}

func copyCategories(c []models.CategoryRule) []models.CategoryRule {
	if c == nil {
		return nil
	}
	out := make([]models.CategoryRule, len(c))
	copy(out, c)
	return out
}
