package gxbank_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGXBankProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GX Bank Provider Suite")
}
