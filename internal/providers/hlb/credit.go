package hlb

import (
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"actual_helper/internal/dateutil"
	"actual_helper/internal/providers/cardutil"
)

var (
	statementDateRe = regexp.MustCompile(`(?:Tarikh Penyata|Statement Date)\s+(\d{2} \w{3} \d{4})`)

	transactionLineRe = regexp.MustCompile(`^\s*(\d{2} \w{3})\s+(\d{2} \w{3})\s+(.+?)\s{2,}([\d,.]+)\s*(CR)?$`)
	creditSkipPatterns = []string{
		"PREVIOUS BALANCE FROM LAST STATEMENT",
		"NEW TRANSACTION / CHARGES",
		"SUB TOTAL",
		"TOTAL BALANCE",
		"PAYMENT RECEIVED - THANK YOU",
	}
)

func extractCreditAccountName(text string) string {
	return cardutil.ExtractAfterMarker(text, "Credit Card Number", "HLB Credit Card")
}

func parseCreditTransactions(text string) ([]HLBReport, error) {
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
		if shouldSkipCreditLine(line) {
			continue
		}

		report, err := parseCreditTransactionLine(line, stmtDate)
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

func shouldSkipCreditLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	lower := strings.ToLower(trimmed)
	for _, pattern := range creditSkipPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func parseCreditTransactionLine(line string, stmtDate time.Time) (HLBReport, error) {
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
