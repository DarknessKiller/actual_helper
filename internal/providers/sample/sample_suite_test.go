package sample_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSampleProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sample Provider Suite")
}
