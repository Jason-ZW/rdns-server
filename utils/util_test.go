package utils_test

import (
	"time"

	"github.com/rancher/rdns-server/keepers"
	"github.com/rancher/rdns-server/keepers/fake"
	"github.com/rancher/rdns-server/types"
	"github.com/rancher/rdns-server/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("util", func() {
	var (
		domain  string
		token   string
		create  time.Time
		payload types.Payload
		keeper  keepers.Keeper
	)

	BeforeEach(func() {
		domain = "lf2bl9.rancher.example"
		token = "IXKM8zsHrBxVBix0yx5hTFwE7m7iuQEH"
		create, _ = time.Parse("2006-01-02 15:04:05", "2017-12-03 22:01:02")
		keeper = fake.NewFaker(token)
		payload = types.Payload{
			Fqdn:     domain,
			Type:     types.RecordTypeA,
			Hosts:    []string{"1.1.1.1"},
			Wildcard: false,
		}
	})

	JustBeforeEach(func() {
		keepers.SetKeeper(keeper)
	})

	Describe("compare the token", func() {
		It("compare the token should correctly", func() {
			wrapToken, err := utils.WrapToken(domain, token)
			if err != nil {
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(utils.CompareToken(wrapToken, payload)).To(BeTrue())
		})
	})

	Describe("convert expire time", func() {
		It("convert expire time should correctly", func() {
			expire, err := time.ParseDuration("240h")
			if err != nil {
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(utils.ConvertExpire(create, int64(expire.Seconds())).UnixNano()).To(Equal(int64(1513202462000000000)))
		})
	})
})
