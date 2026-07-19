package gxbank_test

import (
	"actual_helper/internal/providers/gxbank"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParsePDFBlocks", func() {
	It("returns error when no marker found", func() {
		_, err := gxbank.ParsePDFBlocks("random text without marker")
		Expect(err).To(MatchError("no transactions section found"))
	})

	It("returns error when marker exists but no year in header", func() {
		text := `Statements of Accounts
GX Savings Account
Closing balance (RM)
Baki penutup
`
		_, err := gxbank.ParsePDFBlocks(text)
		Expect(err).To(MatchError("no statement year found in document header"))
	})

	It("parses single interest earned transaction as credit", func() {
		text := `May 2026
Closing balance (RM)
Baki penutup
1 Jun 2026
12:00 AM
Opening balance
10,006.05
1 Jun
11:59 PM
Interest earned
+0.55
10,006.60`
		reports, err := gxbank.ParsePDFBlocks(text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Date).To(Equal("1 Jun 2026"))
		Expect(reports[0].Description).To(Equal("Interest earned"))
		Expect(reports[0].Amount).To(Equal("+0.55"))
		Expect(reports[0].IsCredit).To(BeTrue())
	})

	It("joins multi-line description", func() {
		text := `May 2026
Closing balance (RM)
Baki penutup
21 May 2026
12:00 AM
Opening balance
0.00
21 May
12:09 AM
Pocket
Withdraw from Pocket
+10,097.90
10,097.90`
		reports, err := gxbank.ParsePDFBlocks(text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Description).To(Equal("Pocket Withdraw from Pocket"))
		Expect(reports[0].Amount).To(Equal("+10,097.90"))
		Expect(reports[0].IsCredit).To(BeTrue())
	})

	It("handles amount with commas", func() {
		text := `May 2026
Closing balance (RM)
Baki penutup
5 Jun 2026
12:00 AM
Opening balance
0.00
5 Jun
9:00 AM
Salary
+10,097.90
10,097.90`
		reports, err := gxbank.ParsePDFBlocks(text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Amount).To(Equal("+10,097.90"))
		Expect(reports[0].IsCredit).To(BeTrue())
	})

	It("skips opening balance and parses multiple transactions", func() {
		text := `May 2026
Closing balance (RM)
Baki penutup
1 Jun 2026
12:00 AM
Opening balance
100.00
5 Jun
3:00 PM
Lunch
-25.50
80.50
10 Jun
12:00 AM
Daily Interest Earned
+0.55
81.05`
		reports, err := gxbank.ParsePDFBlocks(text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(2))
		Expect(reports[0].Description).To(Equal("Lunch"))
		Expect(reports[0].Amount).To(Equal("-25.50"))
		Expect(reports[0].IsCredit).To(BeFalse())
		Expect(reports[1].Description).To(Equal("Daily Interest Earned"))
		Expect(reports[1].Amount).To(Equal("+0.55"))
		Expect(reports[1].IsCredit).To(BeTrue())
	})

	It("skips time lines in description", func() {
		text := `May 2026
Closing balance (RM)
Baki penutup
3 Jun 2026
12:00 AM
Opening balance
0.00
3 Jun
12:00 PM
Transfer In
+500.00
500.00`
		reports, err := gxbank.ParsePDFBlocks(text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(1))
		Expect(reports[0].Description).To(Equal("Transfer In"))
	})

	It("skips block with no amount", func() {
		text := `May 2026
Closing balance (RM)
Baki penutup
4 Jun 2026
12:00 AM
Opening balance
0.00
4 Jun
Some description only`
		reports, err := gxbank.ParsePDFBlocks(text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(BeEmpty())
	})

	It("skips page break noise between transactions", func() {
		text := `May 2026
Closing balance (RM)
Baki penutup
11 Jun
11:59 PM
Interest earned
+0.55
10,012.10
GX Bank Berhad
Reg No. 202101014409 (1414709-A)

Date
Tarikh
Transaction description
Butir urusniaga
Money in (RM)
Duit masuk
Money out (RM)
Duit keluar
Interest earned (RM)
Faedah diperolehi
Closing balance (RM)
Baki penutup
12 Jun
11:59 PM
Interest earned
+0.55
10,012.65`
		reports, err := gxbank.ParsePDFBlocks(text)
		Expect(err).NotTo(HaveOccurred())
		Expect(reports).To(HaveLen(2))
		Expect(reports[0].Date).To(Equal("11 Jun 2026"))
		Expect(reports[1].Date).To(Equal("12 Jun 2026"))
	})
})

var _ = Describe("ExtractAccountName", func() {
	It("extracts account name after Statements of Accounts", func() {
		text := `Statements of Accounts
GX Savings Account
Account number
8888--5`
		Expect(gxbank.ExtractAccountName(text)).To(Equal("GX Savings Account"))
	})

	It("extracts account name without Account number line", func() {
		text := `Statements of Accounts
Secret stash Bonus Pocket
May 2026`
		Expect(gxbank.ExtractAccountName(text)).To(Equal("Secret stash Bonus Pocket"))
	})

	It("returns GX Bank when no Statements of Accounts found", func() {
		text := `Some other header
Random content`
		Expect(gxbank.ExtractAccountName(text)).To(Equal("GX Bank"))
	})

	It("returns GX Bank for empty text", func() {
		Expect(gxbank.ExtractAccountName("")).To(Equal("GX Bank"))
	})
})

var _ = Describe("ExtractStatementYear", func() {
	It("extracts year from month year header", func() {
		text := `Statements of Accounts
GX Savings Account
May 2026
Closing balance (RM)`
		Expect(gxbank.ExtractStatementYear(text)).To(Equal("2026"))
	})

	It("handles different months", func() {
		Expect(gxbank.ExtractStatementYear("January 2025")).To(Equal("2025"))
		Expect(gxbank.ExtractStatementYear("December 2024")).To(Equal("2024"))
	})

	It("returns empty when no year found", func() {
		Expect(gxbank.ExtractStatementYear("no dates here")).To(BeEmpty())
	})

	It("returns empty for empty text", func() {
		Expect(gxbank.ExtractStatementYear("")).To(BeEmpty())
	})

	It("returns first year found in cross-month statement", func() {
		// Dec 2025 → Jan 2026 statement: ExtractStatementYear returns first match (2025)
		text := `December 2025
Closing balance (RM)
Baki penutup
31 Dec 2025
12:00 AM
Opening balance
5,000.00
31 Dec
11:59 PM
Interest earned
+1.50
5,001.50
1 Jan 2026
12:00 AM
Opening balance
5,001.50
1 Jan
9:00 AM
Transfer In
+100.00
5,101.50`
		// ExtractStatementYear returns "2025" (first match), which is correct for Dec but wrong for Jan
		// This is a known limitation for cross-year statements
		Expect(gxbank.ExtractStatementYear(text)).To(Equal("2025"))
	})
})
