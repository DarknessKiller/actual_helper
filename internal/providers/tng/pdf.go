package tng

import (
	"errors"
	"log/slog"
	"regexp"
	"strings"
)

func parsePDFBlocks(text string) ([]TNGReport, error) {
	const marker = "TNG WALLET TRANSACTION"
	idx := strings.LastIndex(text, marker)
	if idx == -1 {
		slog.Debug("marker not found in text", "text_preview", truncate(text, 200))
		return nil, errors.New("no transactions section found")
	}

	body := text[idx+len(marker):]
	slog.Debug("pdf body preview", "body", truncate(body, 500))

	dateRe := regexp.MustCompile(`(?m)^(\d{1,2}/\d{1,2}/\d{4})\s+(Success|Failed)\s+`)
	splits := dateRe.FindAllStringSubmatchIndex(body, -1)
	if len(splits) == 0 {
		slog.Info("no transaction blocks found in pdf body",
			"body", truncate(body, 600),
		)
		return nil, nil
	}

	var reports []TNGReport
	for i, split := range splits {
		blockStart := split[0]
		var blockEnd int
		if i+1 < len(splits) {
			blockEnd = splits[i+1][0]
		} else {
			blockEnd = len(body)
		}

		block := strings.TrimSpace(body[blockStart:blockEnd])
		if block == "" {
			continue
		}

		report, err := parseBlock(block)
		if err != nil {
			slog.Debug("pdf block skipped", "reason", err.Error())
			continue
		}
		reports = append(reports, report)
	}

	return reports, nil
}

func parseBlock(block string) (TNGReport, error) {
	lines := strings.Split(block, "\n")
	if len(lines) < 7 {
		return TNGReport{}, errors.New("block too short")
	}

	date := strings.TrimSpace(lines[0])
	status := strings.TrimSpace(lines[1])
	if date == "" || status == "" {
		return TNGReport{}, errors.New("empty date or status")
	}

	transType := strings.TrimSpace(lines[2])
	ref := strings.TrimSpace(lines[3])

	desc := ""
	if len(lines) > 4 {
		desc = trimAtReference(strings.TrimSpace(lines[4]))
	}

	amountRe := regexp.MustCompile(`RM(\d+[.,]?\d*\.\d{2})`)
	amount := ""
	for i := 5; i < len(lines); i++ {
		if m := amountRe.FindStringSubmatch(strings.TrimSpace(lines[i])); m != nil {
			amount = m[1]
			break
		}
	}
	if amount == "" {
		return TNGReport{}, errors.New("no amount found")
	}

	return TNGReport{
		Date:            date,
		Status:          status,
		TransactionType: transType,
		Reference:       ref,
		Description:     desc,
		Amount:          amount,
	}, nil
}



func trimAtReference(text string) string {
	tokens := strings.Fields(text)
	var result []string
	for _, tok := range tokens {
		if isReferenceToken(tok) {
			break
		}
		result = append(result, tok)
	}
	return strings.Join(result, " ")
}

func isReferenceToken(tok string) bool {
	if len(tok) == 0 {
		return false
	}

	allDigits := true
	hasLetter := false
	for _, ch := range tok {
		if ch < '0' || ch > '9' {
			allDigits = false
		}
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			hasLetter = true
		}
	}
	if allDigits && len(tok) >= 10 {
		return true
	}

	// Long alphanumeric starting with >=8 leading digits (YYYYMMDD prefix)
	if len(tok) >= 14 && hasLetter {
		digitCount := 0
		for _, ch := range tok {
			if ch >= '0' && ch <= '9' {
				digitCount++
			} else {
				break
			}
		}
		if digitCount >= 8 {
			return true
		}
	}

	// Letter prefix followed by only digits (e.g. "ABC123")
	if len(tok) >= 4 {
		firstDigit := -1
		for i, ch := range tok {
			if ch >= '0' && ch <= '9' {
				firstDigit = i
				break
			}
		}
		if firstDigit > 0 && firstDigit < len(tok)-1 {
			allDigitsAfter := true
			for i := firstDigit; i < len(tok); i++ {
				if tok[i] < '0' || tok[i] > '9' {
					allDigitsAfter = false
					break
				}
			}
			if allDigitsAfter {
				return true
			}
		}
	}

	prefixes := []string{"TNGD", "TNGQR", "TNGOW"}
	for _, p := range prefixes {
		if strings.HasPrefix(tok, p) && len(tok) > len(p) {
			return true
		}
	}

	return false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}


