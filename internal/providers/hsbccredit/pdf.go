package hsbccredit

import (
	"errors"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var summaryPrefixes = []string{
	"Your Previous Statement Balance",
	"Credit limit used last statement",
	"Your Credit Limit:",
	"Your charge(s) for this month",
	"Total credit limit used",
	"Your statement balance",
}

var (
	statementDateRe = regexp.MustCompile(`Statement Date\s+(\d{2} \w{3} \d{4})`)
	cardNumberRe    = regexp.MustCompile(`(\d{4}[\s-]*\d{4}[\s-]*\d{4}[\s-]*\d{4})`)
	postHeaderRe    = regexp.MustCompile(`(?i)Post date.*Transaction details.*Amount`)
	transactionRe   = regexp.MustCompile(`^(\d{2} \w{3})\s+(\d{2} \w{3})\s+(.+)\s+([\d,]+\.?\d*)(CR)?\s*$`)
)

func parseTransactions(text string) ([]HSBCReport, error) {
	lines := strings.Split(text, "\n")

	stmtDateStr := extractStatementDate(lines)
	if stmtDateStr == "" {
		slog.Warn("statement date not found in text",
			"text_preview", truncate(text, 400),
		)
		return nil, errors.New("statement date not found")
	}

	stmtDate, err := time.Parse("02 Jan 2006", stmtDateStr)
	if err != nil {
		slog.Warn("invalid statement date format", "raw", stmtDateStr)
		return nil, errors.New("invalid statement date")
	}

	dataStart := findTransactionStart(lines)
	if dataStart == -1 {
		slog.Info("no transaction section found in text",
			"text_preview", truncate(text, 400),
		)
		return nil, nil
	}

	var reports []HSBCReport
	for i := dataStart; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		line = strings.ReplaceAll(line, "|", "")
		line = strings.ReplaceAll(line, "[", "")
		line = strings.ReplaceAll(line, "]", "")

		if line == "" {
			continue
		}

		if isSummaryLine(line) {
			continue
		}

		report, err := parseTransactionLine(line, stmtDate)
		if err != nil {
			continue
		}
		reports = append(reports, report)
	}

	return reports, nil
}

func extractAccountName(text string) string {
	idx := strings.Index(text, "Card Number")
	if idx == -1 {
		slog.Debug("card number marker not found", "preview", truncate(text, 600))
		return "HSBC Credit Card"
	}

	after := text[idx+len("Card Number"):]
	after = strings.ReplaceAll(after, "\n", " ")
	after = strings.ReplaceAll(after, "-", " ")

	if matches := cardNumberRe.FindString(after); matches != "" {
		return matches
	}

	slog.Debug("card number not found after 'Card Number'", "preview", truncate(after, 600))
	return "HSBC Credit Card"
}

func extractStatementDate(lines []string) string {
	for _, line := range lines {
		matches := statementDateRe.FindStringSubmatch(line)
		if matches != nil {
			return matches[1]
		}
	}
	return ""
}

func findTransactionStart(lines []string) int {
	for i, line := range lines {
		if postHeaderRe.MatchString(line) {
			for j := i + 1; j < len(lines); j++ {
				candidate := strings.TrimSpace(lines[j])
				candidate = strings.ReplaceAll(candidate, "|", "")
				candidate = strings.ReplaceAll(candidate, "[", "")
				candidate = strings.ReplaceAll(candidate, "]", "")
				if candidate == "" {
					continue
				}
				if transactionRe.MatchString(candidate) {
					return j
				}
			}
			return i + 1
		}
	}
	return -1
}

func isSummaryLine(line string) bool {
	lower := strings.ToLower(line)
	for _, prefix := range summaryPrefixes {
		if strings.HasPrefix(lower, strings.ToLower(prefix)) {
			return true
		}
	}
	return false
}

func parseTransactionLine(line string, stmtDate time.Time) (HSBCReport, error) {
	matches := transactionRe.FindStringSubmatch(line)
	if matches == nil {
		return HSBCReport{}, errors.New("no match")
	}

	postDateStr := matches[1]
	description := strings.TrimSpace(matches[3])
	amountStr := matches[4]
	isCredit := matches[5] == "CR"

	postDate := formatDate(postDateStr, stmtDate)

	return HSBCReport{
		PostDate:    postDate,
		Description: description,
		Amount:      amountStr,
		IsCredit:    isCredit,
	}, nil
}

func formatDate(ddmmm string, stmtDate time.Time) string {
	parts := strings.SplitN(ddmmm, " ", 2)
	if len(parts) != 2 {
		return ddmmm
	}

	day, err := strconv.Atoi(parts[0])
	if err != nil {
		return ddmmm
	}

	monthNum, ok := monthNames[strings.ToUpper(parts[1])]
	if !ok {
		return ddmmm
	}

	stmtMonth := stmtDate.Month()
	year := stmtDate.Year()

	if monthNum > stmtMonth {
		year--
	}

	t := time.Date(year, monthNum, day, 0, 0, 0, 0, time.UTC)
	return t.Format("2006-01-02")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
