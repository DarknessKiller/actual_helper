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

func main() {
	js.Global().Set("actualHelperConvert", js.FuncOf(convert))
	js.Global().Set("actualHelperParsePDFText", js.FuncOf(parsePDFText))
	select {}
}

func parsePDFText(_ js.Value, args []js.Value) any {
	if len(args) != 2 {
		return errorJSON("expected provider and extracted PDF text")
	}
	provider, ok := registry()[args[0].String()]
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
	provider, ok := registry()[args[0].String()]
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

func registry() map[string]providers.Provider {
	var empty []string
	var categories []models.CategoryRule
	var mappings map[string]string
	return map[string]providers.Provider{
		"tng": tng.New(empty, empty, categories, mappings), "ryt": ryt.New(empty, empty, categories, mappings),
		"hsbccredit": hsbccredit.New(empty, empty, categories, mappings), "hlb": hlb.New(empty, empty, categories, mappings),
		"gxbank": gxbank.New(empty, empty, categories, mappings), "uobcredit": uobcredit.New(empty, empty, categories, mappings),
	}
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
