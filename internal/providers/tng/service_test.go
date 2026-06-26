package tng_test

import (
	"context"
	"strings"

	"actual_helper/internal/providers/tng"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TNGProvider", func() {
	var (
		provider = tng.New(nil, nil, nil, nil)
		ctx      = context.Background()
	)

	Describe("ParseCSV", func() {
		It("returns error because tng only supports PDF", func() {
			_, err := provider.ParseCSV(ctx, strings.NewReader("a,b,c\n1,2,3"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("CSV not supported"))
		})
	})

	Describe("Name", func() {
		It("returns tng", func() {
			Expect(provider.Name()).To(Equal("tng"))
		})
	})
})
