package hlb

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var debitDateRe = regexp.MustCompile(`^\s*(\d{2}-\d{2}-\d{4})\s`)
var debitAmountRe = regexp.MustCompile(`([\d,]+\.\d{2})`)

func parseDebitTransactions(text string) ([]HLBReport, error) {
	lines := strings.Split(text, "\n")

	// Detect layout format: header has "Deposit" and "Withdrawal" on same line
	withdrawalCol := -1
	balanceCol := -1
	for _, line := range lines {
		if strings.Contains(line, "Deposit") && strings.Contains(line, "Withdrawal") {
			withdrawalCol = strings.Index(line, "Withdrawal")
			balanceCol = strings.Index(line, "Balance")
			break
		}
	}

	if withdrawalCol >= 0 {
		return parseDebitLayout(lines, withdrawalCol, balanceCol)
	}
	return parseDebitColumnar(lines)
}

func parseDebitLayout(lines []string, withdrawalCol, balanceCol int) ([]HLBReport, error) {
	txStart := 0
	headerFound := false
	for i, line := range lines {
		if strings.Contains(line, "Deposit") && strings.Contains(line, "Withdrawal") {
			headerFound = true
			continue
		}
		if headerFound {
			// First date line after header
			dateMatch := debitDateRe.FindStringSubmatch(line)
			if dateMatch != nil {
				txStart = i
				break
			}
			// Also check for "Balance from previous statement" as first transaction line
			if strings.Contains(line, "Balance from previous statement") {
				txStart = i
				break
			}
		}
	}

	var reports []HLBReport
	var curDate string
	var curDesc []string
	var curDeposit, curWithdrawal string
	blankCount := 0

	flush := func() {
		if curDate == "" {
			return
		}
		desc := strings.TrimSpace(strings.Join(curDesc, " "))
		if desc == "" || strings.Contains(desc, "Balance from previous statement") {
			curDate = ""
			curDesc = nil
			curDeposit = ""
			curWithdrawal = ""
			return
		}

		amount := ""
		isCredit := false
		if curDeposit != "" {
			amount = curDeposit
			isCredit = true
		} else if curWithdrawal != "" {
			amount = curWithdrawal
			isCredit = false
		}

		if amount != "" {
			reports = append(reports, HLBReport{
				TransDate:   curDate,
				Description: desc,
				Amount:      amount,
				IsCredit:    isCredit,
			})
		}

		curDate = ""
		curDesc = nil
		curDeposit = ""
		curWithdrawal = ""
	}

	for i := txStart; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blankCount++
			// If we had a date and now hit 3+ blank lines, we're past the transaction section
			if blankCount >= 3 && curDate != "" {
				flush()
				break
			}
			continue
		}
		blankCount = 0

		// Stop at summary/footer
		if strings.Contains(trimmed, "Total Withdrawals") || strings.Contains(trimmed, "Total Deposits") ||
			strings.Contains(trimmed, "Closing Balance") || strings.Contains(trimmed, "Baki Akhir") {
			flush()
			break
		}

		dateMatch := debitDateRe.FindStringSubmatch(line)
		if dateMatch != nil {
			flush()
			parts := strings.Split(dateMatch[1], "-")
			curDate = parts[2] + "-" + parts[1] + "-" + parts[0]

			desc, dep, wit := extractLayoutAmounts(line, len(dateMatch[0]), withdrawalCol, balanceCol)
			curDesc = []string{desc}
			curDeposit = dep
			curWithdrawal = wit
		} else {
			desc, dep, wit := extractLayoutAmounts(line, 0, withdrawalCol, balanceCol)
			if desc != "" {
				curDesc = append(curDesc, desc)
			}
			if dep != "" {
				curDeposit = dep
			}
			if wit != "" {
				curWithdrawal = wit
			}
		}
	}
	flush()

	if len(reports) == 0 {
		return nil, errors.New("no transactions found")
	}
	return reports, nil
}

func extractLayoutAmounts(line string, start int, withdrawalCol, balanceCol int) (desc, deposit, withdrawal string) {
	content := line[start:]
	matches := debitAmountRe.FindAllStringIndex(content, -1)
	if matches == nil {
		return strings.TrimSpace(content), "", ""
	}

	lastMatch := matches[len(matches)-1]
	lastNum := content[lastMatch[0]:lastMatch[1]]
	lastNumEnd := start + lastMatch[1]
	rest := content[:lastMatch[0]]

	prevMatches := debitAmountRe.FindAllStringIndex(rest, -1)
	if prevMatches == nil {
		if balanceCol >= 0 && lastNumEnd >= balanceCol {
			return strings.TrimSpace(rest), "", ""
		} else if lastNumEnd >= withdrawalCol {
			return strings.TrimSpace(rest), "", lastNum
		}
		return strings.TrimSpace(rest), lastNum, ""
	}

	prevMatch := prevMatches[len(prevMatches)-1]
	firstNum := rest[prevMatch[0]:prevMatch[1]]
	firstNumEnd := start + prevMatch[1]
	desc = strings.TrimSpace(rest[:prevMatch[0]])

	if firstNumEnd >= withdrawalCol {
		return desc, "", firstNum
	}
	return desc, firstNum, ""
}

func parseDebitColumnar(lines []string) ([]HLBReport, error) {
	var dates []string
	var descriptions []string
	var amounts []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if len(trimmed) == 10 && trimmed[2] == '-' && trimmed[5] == '-' {
			parts := strings.Split(trimmed, "-")
			if len(parts) == 3 {
				dates = append(dates, parts[2]+"-"+parts[1]+"-"+parts[0])
			}
			continue
		}

		if trimmed == "Total" || trimmed == "Date / Tarikh" ||
			trimmed == "A/C No / No Akaun" || trimmed == "No Akaun" ||
			trimmed == "Statement Period /" || trimmed == "Tempoh Penyataan" ||
			trimmed == "Balance" || trimmed == "Baki" ||
			trimmed == "Simpanan" ||
			trimmed == "Withdrawal" || trimmed == "Pengeluaran" ||
			trimmed == "Transaction Description" || trimmed == "Deskripsi Transaksi" ||
			trimmed == "Date" || trimmed == "Tarikh" ||
			trimmed == "Branch / Cawangan" || trimmed == "Tel No / No Tel" ||
			strings.HasPrefix(trimmed, ":") {
			continue
		}

		amountStr := strings.ReplaceAll(trimmed, ",", "")
		if amount, err := strconv.ParseFloat(amountStr, 64); err == nil && amount > 0 {
			amounts = append(amounts, trimmed)
			continue
		}

		if trimmed == "Balance from previous statement" {
			continue
		}

		descriptions = append(descriptions, trimmed)
	}

	if len(dates) == 0 {
		return nil, errors.New("no dates found")
	}

	var reports []HLBReport
	for i, date := range dates {
		desc := ""
		amount := ""

		if i < len(descriptions) {
			desc = descriptions[i]
		}
		if i < len(amounts) {
			amount = amounts[i]
		}

		if desc != "" && amount != "" {
			reports = append(reports, HLBReport{
				TransDate:   date,
				Description: desc,
				Amount:      amount,
				IsCredit:    desc == "Deposit",
			})
		}
	}

	return reports, nil
}
