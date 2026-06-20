package tng_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTNGProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TNG Provider Suite")
}
