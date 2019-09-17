package utils_test

import (
	"github.com/rancher/rdns-server/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("random", func() {
	var (
		length int
	)

	BeforeEach(func() {
		length = 24
	})

	Describe("compare the token", func() {
		Context("with small letters", func() {
			It("generate token should correctly", func() {
				Expect(len(utils.RandStringWithSmall(length))).To(Equal(24))
			})
		})
		Context("with all letters", func() {
			It("generate token should correctly", func() {
				Expect(len(utils.RandStringWithAll(length))).To(Equal(24))
			})
		})
	})
})
