package main

import (
	"context"
	"encoding/json"
	"syscall/js"

	"actual_helper/internal/models"
	"actual_helper/internal/providers"
	"actual_helper/internal/providers/gxbank"
	"actual_helper/internal/providers/hlb"
	"actual_helper/internal/providers/hsbccredit"
	"actual_helper/internal/providers/ryt"
	"actual_helper/internal/providers/tng"
	"actual_helper/internal/providers/uobcredit"
	"actual_helper/internal/rule"
)

var registry *providers.Registry

func init() {
	registry = providers.NewRegistry()
	registry.Register(tng.New(nil, nil, nil, nil))
	registry.Register(ryt.New(nil, nil, nil, nil))
	registry.Register(hlb.New(nil, nil, nil, nil))
	registry.Register(hsbccredit.New(nil, nil, nil, nil))
	registry.Register(uobcredit.New(nil, nil, nil, nil))
	registry.Register(gxbank.New(nil, nil, nil, nil))
}

func main() {
	js.Global().Set("goParse", js.FuncOf(parseText))
	js.Global().Set("goProviders", js.FuncOf(listProviders))
	<-make(chan struct{})
}

func listProviders(_ js.Value, _ []js.Value) interface{} {
	return []interface{}{"tng", "ryt", "hlb", "hsbccredit", "uobcredit", "gxbank"}
}

func parseText(_ js.Value, args []js.Value) interface{} {
	if len(args) < 3 {
		return errorResult("usage: goParse(providerName, text, configJSON)")
	}

	name := args[0].String()
	text := args[1].String()
	configJSON := args[2].String()

	prov, ok := registry.Get(name)
	if !ok {
		return errorResult("unknown provider: " + name)
	}

	ctx := context.Background()
	reports, err := prov.ParsePDFText(ctx, text)
	if err != nil {
		return errorResult(err.Error())
	}

	// Apply rule engine if config provided
	if configJSON != "" && configJSON != "{}" {
		cfg, err := parseConfigJSON(configJSON)
		if err == nil {
			merged := mergeConfig(cfg, name)
			engine := rule.NewEngine(merged.ExcludeKeywords, merged.IncludeKeywords, merged.Categories)
			reports = applyRules(reports, engine)
		}
	}

	return js.ValueOf(map[string]interface{}{
		"ok":      true,
		"count":   len(reports),
		"reports": reportsToJS(reports),
	})
}

type configJSON struct {
	Global    providerConfig            `json:"global"`
	Providers map[string]providerConfig `json:"providers"`
}

type providerConfig struct {
	ExcludeKeywords []string              `json:"exclude_keywords"`
	IncludeKeywords []string              `json:"include_keywords"`
	Categories      []models.CategoryRule `json:"categories"`
	AccountMappings map[string]string     `json:"account_mappings"`
}

func parseConfigJSON(s string) (*configJSON, error) {
	var cfg configJSON
	if err := json.Unmarshal([]byte(s), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func mergeConfig(cfg *configJSON, providerName string) providerConfig {
	merged := cfg.Global
	if merged.AccountMappings == nil {
		merged.AccountMappings = make(map[string]string)
	}
	if pc, ok := cfg.Providers[providerName]; ok {
		merged.ExcludeKeywords = append(merged.ExcludeKeywords, pc.ExcludeKeywords...)
		merged.IncludeKeywords = append(merged.IncludeKeywords, pc.IncludeKeywords...)
		merged.Categories = append(merged.Categories, pc.Categories...)
		for k, v := range pc.AccountMappings {
			merged.AccountMappings[k] = v
		}
	}
	return merged
}

func applyRules(reports []models.ActualBudgetReport, engine *rule.Engine) []models.ActualBudgetReport {
	var filtered []models.ActualBudgetReport
	for _, r := range reports {
		if engine.ShouldSkip(r.Notes) {
			continue
		}
		group, cat := engine.MatchCategory(r.Notes)
		if group != "" {
			r.CategoryGroup = group
			r.Category = cat
		}
		filtered = append(filtered, r)
	}
	return filtered
}

func reportsToJS(reports []models.ActualBudgetReport) []interface{} {
	result := make([]interface{}, len(reports))
	for i, r := range reports {
		result[i] = map[string]interface{}{
			"Account":        r.Account,
			"Date":           r.Date,
			"Payee":          r.Payee,
			"Notes":          r.Notes,
			"Category_Group": r.CategoryGroup,
			"Category":       r.Category,
			"Amount":         r.Amount,
			"Split_Amount":   r.SplitAmount,
			"Cleared":        r.Cleared,
		}
	}
	return result
}

func errorResult(msg string) interface{} {
	return js.ValueOf(map[string]interface{}{
		"ok":    false,
		"error": msg,
	})
}
