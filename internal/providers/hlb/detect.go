package hlb

import "strings"

func DetectFormat(text string) string {
	if strings.Contains(text, "Credit Card Number") || strings.Contains(text, "HLB Credit Card") {
		return "credit"
	}
	if strings.Contains(text, "A/C No") || strings.Contains(text, "No Akaun") {
		return "debit"
	}
	return "unknown"
}