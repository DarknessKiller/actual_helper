package hlb_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHLB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HLB Suite")
}
