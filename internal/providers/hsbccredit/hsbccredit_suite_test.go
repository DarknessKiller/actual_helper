package hsbccredit_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHSBCCredit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HSBCCredit Suite")
}
