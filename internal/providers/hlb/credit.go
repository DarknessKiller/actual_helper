package hlb

import (
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"actual_helper/internal/dateutil"
)

var (
	statementDateRe = regexp.MustCompile(`(?:Tarikh Penyata|Statement Date)\s+(\d{2} \w{3} \d{4})`)

	transactionLineRe  = regexp.MustCompile(`^\s*(\d{2} \w{3})\s+(\d{2} \w{3})\s+(.+?)\s{2,}([\d,.]+)\s*(CR)?$`)
	creditSkipPatterns = []string{
		"PREVIOUS BALANCE FROM LAST STATEMENT",
		"NEW TRANSACTION / CHARGES",
		"SUB TOTAL",
		"TOTAL BALANCE",
		"PAYMENT RECEIVED - THANK YOU",
	}
)

func parseCreditTransactions(text string) ([]HLBReport, error) {
	lines := strings.Split(text, "\n")

	var stmtDateStr string
	for _, line := range lines {
		if matches := statementDateRe.FindStringSubmatch(line); matches != nil {
			stmtDateStr = matches[1]
			break
		}
	}
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
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		skip := false
		for _, pattern := range creditSkipPatterns {
			if strings.Contains(lower, strings.ToLower(pattern)) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		matches := transactionLineRe.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		transDate, err := dateutil.FormatDate(matches[1], stmtDate)
		if err != nil {
			continue
		}
		postDate, err := dateutil.FormatDate(matches[2], stmtDate)
		if err != nil {
			continue
		}

		reports = append(reports, HLBReport{
			TransDate:   transDate,
			PostDate:    postDate,
			Description: strings.TrimSpace(matches[3]),
			Amount:      matches[4],
			IsCredit:    matches[5] == "CR",
		})
	}

	return reports, nil
}
