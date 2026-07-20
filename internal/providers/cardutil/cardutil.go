package cardutil

import (
	"log/slog"
	"regexp"
	"strings"

	"actual_helper/internal/dateutil"
)

// CardNumberRe matches card numbers: 4 groups of 4 digits separated by spaces or dashes.
var CardNumberRe = regexp.MustCompile(`(\d{4}[\s-]*\d{4}[\s-]*\d{4}[\s-]*\d{4})`)

// ExtractAfterMarker finds marker in text, extracts card number after it.
// Returns fallback if marker or card number not found.
func ExtractAfterMarker(text, marker, fallback string) string {
	idx := strings.Index(text, marker)
	if idx == -1 {
		slog.Debug("card number marker not found", "marker", marker, "preview", dateutil.Truncate(text, 600))
		return fallback
	}

	after := text[idx+len(marker):]
	after = strings.ReplaceAll(after, "\n", " ")
	after = strings.ReplaceAll(after, "-", " ")

	if matches := CardNumberRe.FindString(after); matches != "" {
		return matches
	}

	slog.Debug("card number not found after marker", "marker", marker, "preview", dateutil.Truncate(after, 600))
	return fallback
}

// ExtractNearCardType finds a card type indicator, searches nearby area for card number.
// Returns fallback if not found.
func ExtractNearCardType(text string, cardTypes []string, fallback string) string {
	for _, ct := range cardTypes {
		idx := strings.Index(text, ct)
		if idx == -1 {
			continue
		}

		start := idx
		if start > 50 {
			start -= 50
		}
		end := idx + 200
		if end > len(text) {
			end = len(text)
		}
		area := text[start:end]

		if matches := CardNumberRe.FindString(area); matches != "" {
			return matches
		}
	}

	slog.Debug("card number not found near card type indicators", "card_types", cardTypes)
	return fallback
}

// ApplyMapping looks up account name in mapping, returns original if not found.
func ApplyMapping(mapping map[string]string, name string) string {
	if mapping == nil {
		return name
	}
	if mapped, ok := mapping[name]; ok {
		return mapped
	}
	return name
}
