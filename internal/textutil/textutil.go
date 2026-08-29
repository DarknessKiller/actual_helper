// Package textutil holds small text helpers shared across providers.
package textutil

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Whitespace collapses runs of whitespace into a single space.
var Whitespace = regexp.MustCompile(`\s+`)

// Collapse returns s with all runs of whitespace replaced by a single space,
// and surrounding spaces trimmed. Safe on empty input.
func Collapse(s string) string {
	return strings.TrimSpace(Whitespace.ReplaceAllString(s, " "))
}

// ParseAmount strips thousands separators and an optional currency prefix,
// then parses the result as a float64.
func ParseAmount(s string, stripPrefixes ...string) (float64, error) {
	for _, p := range stripPrefixes {
		s = strings.ReplaceAll(s, p, "")
	}
	s = strings.ReplaceAll(s, ",", "")
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}

// ParseDateMulti tries each format in order and returns the first match.
func ParseDateMulti(s string, formats ...string) (time.Time, error) {
	var lastErr error
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = errors.New("no matching date format")
	}
	return time.Time{}, lastErr
}
