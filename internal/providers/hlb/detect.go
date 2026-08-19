package hlb

import "strings"

// DetectFormat classifies a statement as credit or debit. Text comes from
// browser pdf.js/Tesseract extraction where marker case and spacing vary, so
// match case-insensitively after collapsing whitespace runs to single spaces.
func DetectFormat(text string) string {
	norm := collapseWS(strings.ToLower(text))
	if strings.Contains(norm, "credit card number") || strings.Contains(norm, "hlb credit card") {
		return "credit"
	}
	if strings.Contains(norm, "a/c no") || strings.Contains(norm, "no akaun") {
		return "debit"
	}
	if strings.Contains(norm, "tarikh penyata") || strings.Contains(norm, "statement date") {
		return "credit"
	}
	if strings.Contains(norm, "deposit") && strings.Contains(norm, "withdrawal") {
		return "debit"
	}
	return "unknown"
}

func collapseWS(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
