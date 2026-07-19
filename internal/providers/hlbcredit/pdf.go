package hlbcredit

import (
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"actual_helper/internal/dateutil"
)

var (
	statementDateRe   = regexp.MustCompile(`(?:Tarikh Penyata|Statement Date)\s+(\d{2} \w{3} \d{4})`)
	cardNumberRe      = regexp.MustCompile(`(\d{4}[\s-]*\d{4}[\s-]*\d{4}[\s-]*\d{4})`)
	transactionLineRe = regexp.MustCompile(`^\s*(\d{2} \w{3})\s+(\d{2} \w{3})\s+(.+?)\s{2,}([\d,.]+)\s*(CR)?$`)
	skipPatterns      = []string{
		"PREVIOUS BALANCE FROM LAST STATEMENT",
		"NEW TRANSACTION / CHARGES",
		"SUB TOTAL",
		"TOTAL BALANCE",
		"PAYMENT RECEIVED - THANK YOU",
	}
)

func parseTransactions(text string) ([]HLBReport, error) {
	lines := strings.Split(text, "\n")

	stmtDateStr := extractStatementDate(lines)
	if stmtDateStr == "" {
		slog.Warn("statement date not found in HLB text",
			"text_preview", dateutil.Truncate(text, 400),
		)
		return nil, errors.New("statement date not found")
	}

	stmtDate, err := time.Parse("02 Jan 2006", stmtDateStr)
	if err != nil {
		slog.Warn("invalid statement date format", "raw", stmtDateStr)
		return nil, errors.New("invalid statement date")
	}

	var reports []HLBReport
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

func extractStatementDate(lines []string) string {
	for _, line := range lines {
		matches := statementDateRe.FindStringSubmatch(line)
		if matches != nil {
			return matches[1]
		}
	}
	return ""
}

func extractAccountName(text string) string {
	idx := strings.Index(text, "Credit Card Number")
	if idx == -1 {
		slog.Debug("card number marker not found in HLB text", "preview", dateutil.Truncate(text, 600))
		return "HLB Credit Card"
	}

	after := text[idx+len("Credit Card Number"):]
	after = strings.ReplaceAll(after, "\n", " ")
	after = strings.ReplaceAll(after, "-", " ")

	if matches := cardNumberRe.FindString(after); matches != "" {
		return matches
	}

	slog.Debug("card number not found after 'Credit Card Number'", "preview", dateutil.Truncate(after, 600))
	return "HLB Credit Card"
}

func shouldSkipLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	lower := strings.ToLower(trimmed)
	for _, pattern := range skipPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func parseTransactionLine(line string, stmtDate time.Time) (HLBReport, error) {
	matches := transactionLineRe.FindStringSubmatch(line)
	if matches == nil {
		return HLBReport{}, errors.New("no match")
	}

	transDateStr := matches[1]
	postDateStr := matches[2]
	description := strings.TrimSpace(matches[3])
	amountStr := matches[4]
	isCredit := matches[5] == "CR"

	transDate, err := dateutil.FormatDate(transDateStr, stmtDate)
	if err != nil {
		return HLBReport{}, err
	}
	postDate, err := dateutil.FormatDate(postDateStr, stmtDate)
	if err != nil {
		return HLBReport{}, err
	}

	return HLBReport{
		TransDate:   transDate,
		PostDate:    postDate,
		Description: description,
		Amount:      amountStr,
		IsCredit:    isCredit,
	}, nil
}
