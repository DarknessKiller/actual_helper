package dateutil_test

import (
	"testing"
	"time"

	"actual_helper/internal/dateutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDateutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dateutil Suite")
}

var _ = Describe("FormatDate", func() {
	It("formats same month correctly", func() {
		stmtDate := time.Date(2026, time.July, 14, 0, 0, 0, 0, time.UTC)
		result := dateutil.FormatDate("15 JUN", stmtDate)
		Expect(result).To(Equal("2026-06-15"))
	})

	It("formats previous year for December on January statement", func() {
		stmtDate := time.Date(2027, time.January, 14, 0, 0, 0, 0, time.UTC)
		result := dateutil.FormatDate("25 DEC", stmtDate)
		Expect(result).To(Equal("2026-12-25"))
	})

	It("formats same month for January on January statement", func() {
		stmtDate := time.Date(2027, time.January, 14, 0, 0, 0, 0, time.UTC)
		result := dateutil.FormatDate("05 JAN", stmtDate)
		Expect(result).To(Equal("2027-01-05"))
	})

	It("returns original string for invalid format", func() {
		stmtDate := time.Date(2026, time.July, 14, 0, 0, 0, 0, time.UTC)
		result := dateutil.FormatDate("invalid", stmtDate)
		Expect(result).To(Equal("invalid"))
	})

	It("returns original string for invalid day", func() {
		stmtDate := time.Date(2026, time.July, 14, 0, 0, 0, 0, time.UTC)
		result := dateutil.FormatDate("XX JUL", stmtDate)
		Expect(result).To(Equal("XX JUL"))
	})

	It("returns original string for invalid month", func() {
		stmtDate := time.Date(2026, time.July, 14, 0, 0, 0, 0, time.UTC)
		result := dateutil.FormatDate("15 XXX", stmtDate)
		Expect(result).To(Equal("15 XXX"))
	})
})

var _ = Describe("Truncate", func() {
	It("returns original string if shorter than limit", func() {
		result := dateutil.Truncate("hello", 10)
		Expect(result).To(Equal("hello"))
	})

	It("returns original string if equal to limit", func() {
		result := dateutil.Truncate("hello", 5)
		Expect(result).To(Equal("hello"))
	})

	It("truncates and adds ellipsis if longer than limit", func() {
		result := dateutil.Truncate("hello world", 5)
		Expect(result).To(Equal("hello..."))
	})

	It("handles empty string", func() {
		result := dateutil.Truncate("", 5)
		Expect(result).To(Equal(""))
	})
})
