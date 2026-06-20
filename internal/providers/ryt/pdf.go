package ryt

import (
	"errors"
	"log/slog"
	"regexp"
	"strings"
)

func extractAccountName(text string) string {
	const marker = "Account Transactions"
	idx := strings.Index(text, marker)
	if idx == -1 {
		return ""
	}
	after := text[idx+len(marker):]
	lines := strings.Split(after, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "/") {
			continue
		}
		if parts := strings.SplitN(line, " / ", 2); len(parts) == 2 && strings.TrimSpace(parts[0]) != "" {
			return strings.TrimSpace(parts[0])
		}
	}
	// Fallback: if no " / " found, use first non-empty non-slash line
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "/") {
			return line
		}
	}
	return ""
}

func parseBlocks(text string) ([]RytReport, error) {
	const marker = "Account Transactions"
	idx := strings.Index(text, marker)
	if idx == -1 {
		slog.Warn("marker not found in text", "text_preview", truncate(text, 400))
		return nil, errors.New("no account transactions section found")
	}

	body := text[idx+len(marker):]

	// Find last column header (Baki = balance) to know where data starts
	balanceHeaderIdx := findBalanceHeader(body)
	if balanceHeaderIdx == -1 {
		slog.Info("no column headers found in pdf body",
			"body_preview", truncate(body, 400),
		)
		return nil, nil
	}

	dataStart := balanceHeaderIdx + len("Baki")
	for dataStart < len(body) && (body[dataStart] == '\n' || body[dataStart] == '\r') {
		dataStart++
	}
	data := strings.TrimSpace(body[dataStart:])
	if data == "" {
		slog.Info("empty data section after balance header",
			"body_around_header", truncate(body[max(0, balanceHeaderIdx-50):min(len(body), balanceHeaderIdx+100)], 150),
		)
		return nil, nil
	}

	dateRe := regexp.MustCompile(`(?m)^\s*(\d{1,2} [A-Za-z]+ \d{4})\b`)
	splits := dateRe.FindAllStringSubmatchIndex(data, -1)
	if len(splits) == 0 {
		slog.Info("no transaction blocks found in pdf data",
			"data", truncate(data, 600),
		)
		return nil, nil
	}

	var reports []RytReport
	for i, split := range splits {
		blockStart := split[0]
		var blockEnd int
		if i+1 < len(splits) {
			blockEnd = splits[i+1][0]
		} else {
			blockEnd = len(data)
		}

		block := strings.TrimSpace(data[blockStart:blockEnd])
		if block == "" {
			continue
		}

		report, err := parseBlock(block)
		if err != nil {
			slog.Info("pdf block skipped", "reason", err.Error(), "block", truncate(block, 200))
			continue
		}
		reports = append(reports, report)
	}

	return reports, nil
}

func parseBlock(block string) (RytReport, error) {
	lines := strings.Split(block, "\n")
	if len(lines) < 2 {
		return RytReport{}, errors.New("block too short")
	}

	firstLine := strings.TrimSpace(lines[0])
	dateRe := regexp.MustCompile(`^(\d{1,2} [A-Za-z]+ \d{4})\s*(.*)`)
	matches := dateRe.FindStringSubmatch(firstLine)
	if matches == nil {
		return RytReport{}, errors.New("no date found in block")
	}

	date := matches[1]

	// Find amount: scan from bottom for [+-] prefix (signed transaction)
	amountLine := -1
	signedRe := regexp.MustCompile(`^[+-]\d+[.,]?\d*\.?\d*$`)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if signedRe.MatchString(line) {
			amountLine = i
			break
		}
	}

	if amountLine == -1 {
		// Fallback: last non-empty line (opening balance, no sign)
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if line != "" {
				amountRe := regexp.MustCompile(`^(-?\d+[.,]?\d*\.?\d*)$`)
				if amountRe.MatchString(line) {
					amountLine = i
					break
				}
			}
		}
	}

	if amountLine == -1 {
		return RytReport{}, errors.New("no amount found in block")
	}

	amount := strings.TrimSpace(lines[amountLine])

	// Description: text after date on first line + lines between date and amount
	var descParts []string
	if text := strings.TrimSpace(matches[2]); text != "" {
		descParts = append(descParts, text)
	}
	for i := 1; i < amountLine; i++ {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			descParts = append(descParts, line)
		}
	}

	description := strings.Join(descParts, " / ")

	return RytReport{
		Date:        date,
		Description: description,
		Amount:      amount,
	}, nil
}

func findBalanceHeader(body string) int {
	// Try several patterns in order of specificity
	patterns := []string{
		"Balance\nBaki",    // separate lines (English then Malay)
		"Balance / Baki",   // on same line with slash
		"\nBaki\n",         // standalone "Baki" on its own line
	}
	for _, p := range patterns {
		if idx := strings.Index(body, p); idx != -1 {
			// For "\nBaki\n", the actual Baki starts at idx+1
			if p == "\nBaki\n" {
				return idx + 1
			}
			// For "Balance\nBaki" or "Balance / Baki", find the "Baki" part
			bakiIdx := strings.LastIndex(body[:idx+len(p)], "Baki")
			return bakiIdx
		}
	}
	// Last resort: find any "Baki" in the first 300 chars (header area)
	headerArea := body
	if len(headerArea) > 300 {
		headerArea = headerArea[:300]
	}
	if idx := strings.Index(headerArea, "Baki"); idx != -1 {
		return idx
	}
	return -1
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
