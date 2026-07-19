package dateutil

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// monthNames maps 3-letter month abbreviations to time.Month.
var monthNames = map[string]time.Month{
	"JAN": time.January, "FEB": time.February, "MAR": time.March,
	"APR": time.April, "MAY": time.May, "JUN": time.June,
	"JUL": time.July, "AUG": time.August, "SEP": time.September,
	"OCT": time.October, "NOV": time.November, "DEC": time.December,
}

// FormatDate converts "DD MMM" to "YYYY-MM-DD", inferring year from stmtDate.
func FormatDate(ddmmm string, stmtDate time.Time) (string, error) {
	parts := strings.SplitN(ddmmm, " ", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid date format: " + ddmmm)
	}

	day, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", errors.New("invalid day in date: " + ddmmm)
	}

	monthNum, ok := monthNames[strings.ToUpper(parts[1])]
	if !ok {
		return "", errors.New("unknown month in date: " + ddmmm)
	}

	stmtMonth := stmtDate.Month()
	year := stmtDate.Year()

	if monthNum > stmtMonth {
		year--
	}

	t := time.Date(year, monthNum, day, 0, 0, 0, 0, time.UTC)
	return t.Format("2006-01-02"), nil
}

// Truncate truncates string s to n characters, appending "..." if truncated.
func Truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
