package hlb_test

import (
	hlbprov "actual_helper/internal/providers/hlb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("detectFormat", func() {
	It("detects credit format", func() {
		text := `Credit Card Number    1234 5678 9012 3456
Tarikh Penyata                    14 JUL 2026`
		Expect(hlbprov.DetectFormat(text)).To(Equal("credit"))
	})

	It("detects debit format", func() {
		text := `A/C No / No Akaun
: 31200037164
Statement Period /
: 06/06/26 - 05/07/26`
		Expect(hlbprov.DetectFormat(text)).To(Equal("debit"))
	})

	It("returns unknown for unrecognized format", func() {
		text := `Random text without markers`
		Expect(hlbprov.DetectFormat(text)).To(Equal("unknown"))
	})
})