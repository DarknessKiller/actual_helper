//go:build js && wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"syscall/js"

	"actual_helper/internal/models"
	"actual_helper/internal/providers"
	gxbank "actual_helper/internal/providers/gxbank"
	hlb "actual_helper/internal/providers/hlb"
	hsbccredit "actual_helper/internal/providers/hsbccredit"
	ryt "actual_helper/internal/providers/ryt"
	tng "actual_helper/internal/providers/tng"
	uobcredit "actual_helper/internal/providers/uobcredit"
	"actual_helper/internal/services"
)

type wasmConfig struct {
	Global    wasmProviderConfig            `json:"global"`
	Providers map[string]wasmProviderConfig `json:"providers"`
}
type wasmProviderConfig struct {
	ExcludeKeywords []string              `json:"exclude_keywords"`
	IncludeKeywords []string              `json:"include_keywords"`
	Categories      []models.CategoryRule `json:"categories"`
	AccountMappings map[string]string     `json:"account_mappings"`
}

var currentConfig wasmConfig
var cachedRegistry map[string]providers.Provider
var configVersion int

func main() {
	js.Global().Set("actualHelperConvert", js.FuncOf(convert))
	js.Global().Set("actualHelperParsePDFText", js.FuncOf(parsePDFText))
	js.Global().Set("actualHelperSetConfig", js.FuncOf(setConfig))
	js.Global().Set("actualHelperResetConfig", js.FuncOf(resetConfig))
	select {}
}

func setConfig(_ js.Value, args []js.Value) any {
	if len(args) != 1 {
		return errorJSON("expected provider config JSON")
	}
	var cfg wasmConfig
	if err := json.Unmarshal([]byte(args[0].String()), &cfg); err != nil {
		return errorJSON(fmt.Sprintf("invalid provider config: %v", err))
	}
	currentConfig = cfg
	configVersion++
	cachedRegistry = nil
	return `{"ok":true}`
}

func resetConfig(_ js.Value, _ []js.Value) any {
	currentConfig = wasmConfig{}
	configVersion++
	cachedRegistry = nil
	return `{"ok":true}`
}

func getRegistry() map[string]providers.Provider {
	if cachedRegistry != nil {
		return cachedRegistry
	}
	cfg := currentConfig
	options := func(name string) wasmProviderConfig {
		pc := cfg.Global
		if p, ok := cfg.Providers[name]; ok {
			pc.ExcludeKeywords = append(pc.ExcludeKeywords, p.ExcludeKeywords...)
			pc.IncludeKeywords = append(pc.IncludeKeywords, p.IncludeKeywords...)
			pc.Categories = append(pc.Categories, p.Categories...)
			pc.AccountMappings = p.AccountMappings
		}
		return pc
	}
	factories := map[string]func([]string, []string, []models.CategoryRule, map[string]string) providers.Provider{
		"tng": tng.New, "ryt": ryt.New, "hsbccredit": hsbccredit.New, "hlb": hlb.New, "gxbank": gxbank.New, "uobcredit": uobcredit.New,
	}
	result := make(map[string]providers.Provider, len(factories))
	for name, factory := range factories {
		pc := options(name)
		result[name] = factory(pc.ExcludeKeywords, pc.IncludeKeywords, pc.Categories, pc.AccountMappings)
	}
	cachedRegistry = result
	return result
}

func parsePDFText(_ js.Value, args []js.Value) any {
	if len(args) != 2 {
		return errorJSON("expected provider and extracted PDF text")
	}
	provider, ok := getRegistry()[args[0].String()]
	if !ok {
		return errorJSON(fmt.Sprintf("provider %q not found", args[0].String()))
	}
	reports, err := provider.ParsePDFText(context.Background(), args[1].String())
	if err != nil {
		return errorJSON(err.Error())
	}
	data, err := services.ToActualCSV(reports)
	if err != nil {
		return errorJSON(err.Error())
	}
	return string(data)
}

func convert(_ js.Value, args []js.Value) any {
	if len(args) != 2 {
		return errorJSON("expected provider and CSV text")
	}
	provider, ok := getRegistry()[args[0].String()]
	if !ok {
		return errorJSON(fmt.Sprintf("provider %q not found", args[0].String()))
	}
	reports, err := provider.ParseCSV(context.Background(), stringReader(args[1].String()))
	if err != nil {
		return errorJSON(err.Error())
	}
	data, err := services.ToActualCSV(reports)
	if err != nil {
		return errorJSON(err.Error())
	}
	return string(data)
}

func stringReader(s string) io.Reader { return &reader{s: s} }

type reader struct{ s string }

func (r *reader) Read(p []byte) (int, error) {
	if len(r.s) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.s)
	r.s = r.s[n:]
	return n, nil
}
func errorJSON(message string) string {
	b, _ := json.Marshal(map[string]string{"error": message})
	return string(b)
}
