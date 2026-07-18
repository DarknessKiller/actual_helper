package uobcredit

import (
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"actual_helper/internal/dateutil"
	"actual_helper/internal/providers"
)

var (
	// Statement date: "Statement Date    16 JUL 26" or "Statement Date\nTarikh Penyata\n\n16 JUL 26"
	statementDateRe = regexp.MustCompile(`Statement Date\s+(\d{2} \w{3} \d{2,4})`)
	// Transaction line (pdftotext -layout): "04 JUL    PAYMENT REC'D...    326.76 CR"
	transactionLineRe = regexp.MustCompile(`^\s*(\d{2} \w{3})\s+(.+?)\s{2,}([\d,.]+)\s*(CR)?\s*$`)
	// Card number: "1234-5678-9012-3456"
	cardNumberRe = regexp.MustCompile(`\d{4}[\s-]*\d{4}[\s-]*\d{4}[\s-]*\d{4}`)
)

var skipPatterns = []string{
	"sub-total",
	"minimum payment due",
	"** end of statement**",
	"credit limit",
	"previous bal",
	"page no",
}

func parseTransactions(text string) ([]UOBReport, error) {
	stmtDateStr := extractStatementDate(text)
	if stmtDateStr == "" {
		slog.Warn("statement date not found in UOB text",
			"text_preview", dateutil.Truncate(text, 400),
		)
		return nil, errors.New("statement date not found")
	}

	stmtDate, err := time.Parse("02 Jan 2006", stmtDateStr)
	if err != nil {
		// Try 2-digit year fallback
		stmtDate, err = time.Parse("02 Jan 06", stmtDateStr)
		if err != nil {
			slog.Warn("invalid statement date format", "raw", stmtDateStr)
			return nil, errors.New("invalid statement date")
		}
	}

	lines := strings.Split(text, "\n")
	var reports []UOBReport

	for _, line := range lines {
		if shouldSkipLine(line) {
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

func extractStatementDate(text string) string {
	matches := statementDateRe.FindStringSubmatch(text)
	if matches == nil {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func shouldSkipLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	lower := strings.ToLower(trimmed)
	for _, pattern := range skipPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

func parseTransactionLine(line string, stmtDate time.Time) (UOBReport, error) {
	matches := transactionLineRe.FindStringSubmatch(line)
	if matches == nil {
		return UOBReport{}, errors.New("no match")
	}

	dateStr := strings.TrimSpace(matches[1])
	description := strings.TrimSpace(matches[2])
	amount := strings.ReplaceAll(strings.TrimSpace(matches[3]), ",", "")
	isCredit := matches[4] == "CR"

	transDate := dateutil.FormatDate(dateStr, stmtDate)

	return UOBReport{
		TransDate:   transDate,
		Description: description,
		Amount:      amount,
		IsCredit:    isCredit,
	}, nil
}

func extractAccountName(text string) string {
	// Look for card number near card type indicator (WORLD MASTERCARD, VISA, etc.)
	cardTypeIdx := strings.Index(text, "WORLD MASTERCARD")
	if cardTypeIdx == -1 {
		cardTypeIdx = strings.Index(text, "MASTERCARD")
	}
	if cardTypeIdx == -1 {
		cardTypeIdx = strings.Index(text, "VISA")
	}

	if cardTypeIdx != -1 {
		// Look in the area around the card type indicator
		start := cardTypeIdx
		if start > 50 {
			start -= 50
		}
		end := cardTypeIdx + 200
		if end > len(text) {
			end = len(text)
		}
		area := text[start:end]

		// Try full card number first
		if matches := cardNumberRe.FindString(area); matches != "" {
			return matches
		}
	}

	slog.Debug("card number not found in UOB text")
	return "UOB Credit Card"
}

// Compile-time check: UOBProvider implements Provider.
var _ providers.Provider = (*UOBProvider)(nil)
