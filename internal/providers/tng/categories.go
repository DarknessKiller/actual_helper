package tng

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type rule struct {
	Keyword  string `json:"keyword"`
	Group    string `json:"group"`
	Category string `json:"category"`
}

func loadRules(path string) ([]rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rules file: %w", err)
	}

	var rules []rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("parse rules file: %w", err)
	}

	for i := range rules {
		rules[i].Keyword = strings.ToLower(rules[i].Keyword)
	}

	return rules, nil
}

func match(rules []rule, description string) (string, string) {
	lower := strings.ToLower(description)
	for _, rule := range rules {
		if strings.Contains(lower, rule.Keyword) {
			return rule.Group, rule.Category
		}
	}
	return "", ""
}

func (provider *TNGProvider) ensureRulesLoaded() {
	if provider.categoriesPath == "" {
		return
	}

	info, err := os.Stat(provider.categoriesPath)
	if err != nil {
		slog.Warn("cannot stat categories file", "path", provider.categoriesPath, "error", err)
		return
	}

	if !info.ModTime().After(provider.lastRuleLoad) && provider.rules != nil {
		return
	}

	rules, err := loadRules(provider.categoriesPath)
	if err != nil {
		slog.Warn("cannot load categories file", "path", provider.categoriesPath, "error", err)
		return
	}

	provider.rules = rules
	provider.lastRuleLoad = info.ModTime()
	slog.Info("categories rules loaded", "path", provider.categoriesPath, "count", len(rules))
}
