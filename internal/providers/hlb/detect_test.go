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
: 12345678901
Statement Period /
: 01/06/26 - 30/06/26`
		Expect(hlbprov.DetectFormat(text)).To(Equal("debit"))
	})

	It("returns unknown for unrecognized format", func() {
		text := `Random text without markers`
		Expect(hlbprov.DetectFormat(text)).To(Equal("unknown"))
	})

	It("detects credit when marker is uppercase with a colon (OCR output)", func() {
		text := `TARIKH PENYATA: 14 JUL 2026`
		Expect(hlbprov.DetectFormat(text)).To(Equal("credit"))
	})

	It("detects credit when marker has variable spacing", func() {
		text := `Tarikh   Penyata    14 JUL 2026`
		Expect(hlbprov.DetectFormat(text)).To(Equal("credit"))
	})

	It("detects credit via the English Statement Date marker", func() {
		text := `Statement Date  14 JUL 2026`
		Expect(hlbprov.DetectFormat(text)).To(Equal("credit"))
	})
})
