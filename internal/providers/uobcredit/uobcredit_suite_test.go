package uobcredit_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestUOBCredit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "UOB Credit Suite")
}
