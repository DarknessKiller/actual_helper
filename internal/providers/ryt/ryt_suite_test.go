package ryt_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRytProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ryt Provider Suite")
}
