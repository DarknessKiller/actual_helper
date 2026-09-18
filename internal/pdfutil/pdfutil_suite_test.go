package pdfutil

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPdfutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pdfutil Suite")
}
