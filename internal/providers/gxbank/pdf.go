package gxbank

import (
	"errors"
	"regexp"
	"strings"
)

var (
	monthYearRe = regexp.MustCompile(`(?i)^(?:January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{4})$`)
	dateRe      = regexp.MustCompile(`(?i)^\d{1,2}\s+(?:Jan(?:uary)?|Feb(?:ruary)?|Mar(?:ch)?|Apr(?:il)?|May|Jun(?:e)?|Jul(?:y)?|Aug(?:ust)?|Sep(?:tember)?|Oct(?:ober)?|Nov(?:ember)?|Dec(?:ember)?)\b(?:\s+\d{4})?$`)
	timeRe      = regexp.MustCompile(`^\d{1,2}:\d{2}\s*(?:AM|PM)$`)
	amountRe    = regexp.MustCompile(`^[+-][\d,]+\.\d{2}$`)
	balanceRe   = regexp.MustCompile(`^[\d,]+\.\d{2}$`)
)

// ParsePDFBlocks parses digital-extracted text where each field is on its own line.
// Transaction structure: date → time → description(s) → amount → balance
func ParsePDFBlocks(text string) ([]GXReport, error) {
	const marker = "Closing balance (RM)"
	idx := strings.Index(text, marker)
	if idx == -1 {
		return nil, errors.New("no transactions section found")
	}

	body := text[idx+len(marker):]
	lines := strings.Split(body, "\n")
	year := ExtractStatementYear(text)
	if year == "" {
		return nil, errors.New("no statement year found in document header")
	}

	// Skip to first date line
	startIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if dateRe.MatchString(trimmed) {
			startIdx = i
			break
		}
	}
	if startIdx == -1 {
		return nil, errors.New("no transaction blocks found in pdf body")
	}

	var reports []GXReport

	// State machine: expect date → time → description(s) → amount → balance
	type state int
	const (
		stIdle state = iota
		stAfterDate
		stAfterTime
		stInDesc
		stAfterAmount
	)

	st := stIdle
	var curDate string
	var curDesc []string
	var curAmount string
	var curIsCredit bool

	flush := func() {
		if curDate != "" && curAmount != "" && len(curDesc) > 0 {
			desc := strings.TrimSpace(strings.Join(curDesc, " "))
			if desc != "" && strings.ToLower(desc) != "opening balance" {
				reports = append(reports, GXReport{
					Date:        curDate,
					Description: desc,
					Amount:      curAmount,
					IsCredit:    curIsCredit,
				})
			}
		}
		curDate = ""
		curDesc = nil
		curAmount = ""
		curIsCredit = false
	}

	for _, line := range lines[startIdx:] {
		trimmed := strings.TrimSpace(line)

		// New date line always starts a new transaction (flush previous)
		if dateRe.MatchString(trimmed) {
			flush()
			fields := strings.Fields(trimmed)
			if len(fields) == 2 && year != "" {
				curDate = trimmed + " " + year
			} else {
				curDate = trimmed
			}
			st = stAfterDate
			continue
		}

		switch st {
		case stAfterDate:
			if timeRe.MatchString(trimmed) {
				st = stAfterTime
			}

		case stAfterTime:
			if amountRe.MatchString(trimmed) {
				// Opening balance: date → time → amount (no description)
				st = stIdle
			} else if strings.ToLower(trimmed) == "opening balance" {
				// Skip opening balance block entirely
				curDate = ""
				curDesc = nil
				st = stIdle
			} else {
				curDesc = append(curDesc, trimmed)
				st = stInDesc
			}

		case stInDesc:
			if amountRe.MatchString(trimmed) {
				curAmount = trimmed
				curIsCredit = strings.HasPrefix(trimmed, "+")
				st = stAfterAmount
			} else {
				curDesc = append(curDesc, trimmed)
			}

		case stAfterAmount:
			if balanceRe.MatchString(trimmed) {
				st = stIdle
			}
		}
	}
	flush()

	return reports, nil
}

func ExtractAccountName(text string) string {
	const header = "Statements of Accounts"
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if !strings.Contains(line, header) {
			continue
		}
		// Digital mode: account name on next non-empty line
		for j := i + 1; j < len(lines); j++ {
			name := strings.TrimSpace(lines[j])
			if name == "" {
				continue
			}
			if monthYearRe.MatchString(name) {
				continue // skip month/year line
			}
			if strings.HasPrefix(name, "Account number") {
				continue // skip account number line
			}
			return name
		}
	}

	return "GX Bank"
}

func ExtractStatementYear(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if m := monthYearRe.FindStringSubmatch(trimmed); m != nil {
			return m[1]
		}
	}
	return ""
}
