package global_test

import (
	"github.com/rancher/rdns-server/commands/global"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("global", func() {
	var (
		cases []struct {
			expect []string
		}
	)

	BeforeEach(func() {
		cases = []struct {
			expect []string
		}{
			{
				expect: []string{
					"LEVEL",
					"PORT",
					"DOMAIN",
					"EXPIRE",
					"ROTATE",
				},
			},
		}
	})

	Describe("get global flags", func() {
		It("get global flags should correctly", func() {
			for _, c := range cases {
				flags := make([]string, 0)
				for flag := range global.GetFlags() {
					flags = append(flags, flag)
				}
				Expect(flags).Should(ConsistOf(c.expect))
			}
		})
	})
})
