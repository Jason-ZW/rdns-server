package utils_test

import (
	"github.com/rancher/rdns-server/types"
	"github.com/rancher/rdns-server/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("dns", func() {
	var (
		domains        []string
		wildcards      []string
		supportTypes   []string
		unSupportTypes []string
		hosts          []string
		texts          []string
	)

	BeforeEach(func() {
		domains = []string{
			"lf2bl9.rancher.example",
			"lf2bl9.rancher.example.",
			"LF2BL9.RANCHER.EXAMPLE",
			"LF2BL9.RANCHER.EXAMPLE.",
		}
		wildcards = []string{
			"*.lf2bl9.rancher.example",
			"*.lf2bl9.rancher.example.",
			"*.LF2BL9.RANCHER.EXAMPLE",
			"*.LF2BL9.RANCHER.EXAMPLE.",
			"\\052.lf2bl9.rancher.example",
			"\\052.lf2bl9.rancher.example.",
			"\\052.LF2BL9.RANCHER.EXAMPLE",
			"\\052.LF2BL9.RANCHER.EXAMPLE.",
		}
		supportTypes = []string{
			types.RecordTypeA,
			types.RecordTypeAAAA,
			types.RecordTypeTXT,
			types.RecordTypeCNAME,
		}
		unSupportTypes = []string{
			types.RecordTypeSRV,
			types.RecordTypeCAA,
			types.RecordTypeMX,
			types.RecordTypeNS,
			types.RecordTypePTR,
		}
		hosts = []string{
			"192.168.1.1",
			"0:0:0:0:0:ffff:c0a8:101",
			"cname.rancher.example",
		}
		texts = []string{
			"this is example text",
		}
	})

	Describe("type to int", func() {
		Context("support types", func() {
			It("type to int should correctly", func() {
				for _, t := range supportTypes {
					if t == types.RecordTypeAAAA || t == types.RecordTypeA {
						Expect(utils.TypeToInt(t, false)).To(Equal(1))
						Expect(utils.TypeToInt(t, true)).To(Equal(2))
					} else if t == types.RecordTypeTXT {
						Expect(utils.TypeToInt(t, false)).To(Equal(0))
					} else {
						Expect(utils.TypeToInt(t, false)).To(Equal(3))
					}
				}
			})
		})
		Context("not support types", func() {
			It("type to int should correctly", func() {
				for _, t := range unSupportTypes {
					Expect(utils.TypeToInt(t, false)).To(Equal(-1))
					Expect(utils.TypeToInt(t, true)).To(Equal(-1))
				}
			})
		})
	})

	Describe("host to type", func() {
		It("host to type should correctly", func() {
			for i, h := range hosts {
				switch i {
				case 0:
					Expect(utils.HostType(h)).To(Equal(types.RecordTypeA))
				case 1:
					Expect(utils.HostType(h)).To(Equal(types.RecordTypeAAAA))
				case 2:
					Expect(utils.HostType(h)).To(Equal(types.RecordTypeCNAME))
				}
			}
		})
	})

	Describe("dns supported type", func() {
		It("dns supported type should correctly", func() {
			for _, t := range supportTypes {
				Expect(utils.IsSupportedType(t)).To(BeTrue())
			}
			for _, t := range unSupportTypes {
				Expect(utils.IsSupportedType(t)).To(BeFalse())
			}
		})
	})

	Describe("has sub domain", func() {
		It("has sub domain should correctly", func() {
			for _, t := range supportTypes {
				switch t {
				case types.RecordTypeA, types.RecordTypeAAAA:
					Expect(utils.HasSubDomain(t)).To(BeTrue())
				default:
					Expect(utils.HasSubDomain(t)).To(BeFalse())
				}
			}
		})
	})

	Describe("get dns name", func() {
		It("get dns name should correctly", func() {
			for _, d := range domains {
				Expect(utils.GetDNSName(d)).To(Equal("lf2bl9.rancher.example"))
			}
			for _, w := range wildcards {
				Expect(utils.GetDNSName(w)).To(Equal("*.lf2bl9.rancher.example"))
			}
		})
	})

	Describe("get dns prefix", func() {
		It("get dns prefix should correctly", func() {
			for _, d := range domains {
				Expect(utils.GetDNSPrefix(d, false)).To(Equal("lf2bl9"))
			}
			for _, w := range wildcards {
				Expect(utils.GetDNSPrefix(w, true)).To(Equal("lf2bl9"))
			}
		})
	})

	Describe("ensure trailing dot", func() {
		It("ensure trailing dot should correctly", func() {
			for _, d := range domains {
				Expect(utils.EnsureTrailingDot(utils.GetDNSName(d))).To(Equal("lf2bl9.rancher.example."))
			}
		})
	})

	Describe("wildcard escape", func() {
		It("wildcard escape should correctly", func() {
			for _, d := range wildcards {
				Expect(utils.WildcardEscape(utils.GetDNSName(d))).To(Equal("\\052.lf2bl9.rancher.example"))
			}
		})
	})

	Describe("text with quotes", func() {
		It("text with quotes should correctly", func() {
			for _, t := range texts {
				Expect(utils.TextWithQuotes(t)).To(Equal("\"this is example text\""))
			}
		})
	})

	Describe("text remove quotes", func() {
		It("text remove quotes should correctly", func() {
			for _, t := range texts {
				Expect(utils.TextWithQuotes(t)).To(Equal("\"this is example text\""))
				Expect(utils.TextRemoveQuotes(utils.TextWithQuotes(t))).To(Equal("this is example text"))
			}
		})
	})
})
