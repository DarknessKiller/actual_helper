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
		result, err := dateutil.FormatDate("15 JUN", stmtDate)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal("2026-06-15"))
	})

	It("formats previous year for December on January statement", func() {
		stmtDate := time.Date(2027, time.January, 14, 0, 0, 0, 0, time.UTC)
		result, err := dateutil.FormatDate("25 DEC", stmtDate)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal("2026-12-25"))
	})

	It("formats same month for January on January statement", func() {
		stmtDate := time.Date(2027, time.January, 14, 0, 0, 0, 0, time.UTC)
		result, err := dateutil.FormatDate("05 JAN", stmtDate)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal("2027-01-05"))
	})

	It("returns error for invalid format", func() {
		stmtDate := time.Date(2026, time.July, 14, 0, 0, 0, 0, time.UTC)
		_, err := dateutil.FormatDate("invalid", stmtDate)
		Expect(err).To(HaveOccurred())
	})

	It("returns error for invalid day", func() {
		stmtDate := time.Date(2026, time.July, 14, 0, 0, 0, 0, time.UTC)
		_, err := dateutil.FormatDate("XX JUL", stmtDate)
		Expect(err).To(HaveOccurred())
	})

	It("returns error for invalid month", func() {
		stmtDate := time.Date(2026, time.July, 14, 0, 0, 0, 0, time.UTC)
		_, err := dateutil.FormatDate("15 XXX", stmtDate)
		Expect(err).To(HaveOccurred())
	})
})
