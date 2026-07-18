package hlbcredit_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHLBCredit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HLB Credit Suite")
}
