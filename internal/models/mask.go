package models

// MaskAccountNumber keeps the last 4 digits of an account or card number and
// replaces every earlier digit with '*'. Separators and non-digit characters
// are preserved. Values with 4 digits or fewer are returned unchanged, so
// account names and fallbacks pass through untouched. Idempotent.
//
// Used to log which account was detected without writing the full number.
// CSV output is intentionally left unmasked.
func MaskAccountNumber(s string) string {
	lastFourStart := -1
	digits := 0
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] < '0' || s[i] > '9' {
			continue
		}
		digits++
		if digits == 4 {
			lastFourStart = i
			break
		}
	}
	if lastFourStart < 0 {
		return s
	}

	masked := []byte(s)
	for i := 0; i < lastFourStart; i++ {
		if masked[i] >= '0' && masked[i] <= '9' {
			masked[i] = '*'
		}
	}
	return string(masked)
}
