package e2e_test

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rancher/rdns-server/types"
	"github.com/rancher/rdns-server/utils"

	"github.com/miekg/dns"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	dnsTimeout    = 10 * time.Second
	lookupTimeout = 30 * time.Second
	envEndpoint   = "ENV_ENDPOINT"
	resolvConf    = "/etc/resolv.conf"
)

type e2ePayload struct {
	Method string
	URI    string
	Auth   string
	Body   string
	Type   string
}

type e2eAPIExpect struct {
	ShouldStatusOK       bool
	ShouldTokenEmpty     bool
	ShouldHasSubDomain   bool
	ShouldRootValueEmpty bool
	ShouldWildcard       bool
}

type e2eDNSExpect struct {
	RootAddresses []string
	SubAddresses  map[string][]string
	CNAME         string
	TXT           string
}

var _ = Describe("e2e", func() {
	var (
		endpoint string
		cases    []struct {
			payload   e2ePayload
			apiExpect e2eAPIExpect
			dnsExpect e2eDNSExpect
		}
	)

	BeforeEach(func() {
		endpoint = os.Getenv(envEndpoint)
		Expect(endpoint).NotTo(BeEmpty())
		cases = []struct {
			payload   e2ePayload
			apiExpect e2eAPIExpect
			dnsExpect e2eDNSExpect
		}{
			{
				payload: e2ePayload{
					Method: http.MethodPost,
					URI:    "/v1/domain",
					Body:   `{"hosts": ["192.168.1.1"]}`,
					Type:   types.RecordTypeA,
				},
				apiExpect: e2eAPIExpect{
					ShouldStatusOK:       true,
					ShouldTokenEmpty:     false,
					ShouldRootValueEmpty: false,
					ShouldHasSubDomain:   false,
					ShouldWildcard:       false,
				},
				dnsExpect: e2eDNSExpect{
					RootAddresses: []string{"192.168.1.1"},
				},
			},
			{
				payload: e2ePayload{
					Method: http.MethodPost,
					URI:    "/v1/domain",
					Body:   `{"hosts": ["0:0:0:0:0:ffff:c0a8:101"]}`,
					Type:   types.RecordTypeAAAA,
				},
				apiExpect: e2eAPIExpect{
					ShouldStatusOK:       true,
					ShouldTokenEmpty:     false,
					ShouldRootValueEmpty: false,
					ShouldHasSubDomain:   false,
					ShouldWildcard:       false,
				},
				dnsExpect: e2eDNSExpect{
					RootAddresses: []string{"192.168.1.1"},
				},
			},
			{
				payload: e2ePayload{
					Method: http.MethodPost,
					URI:    "/v1/domain",
					Body:   `{"hosts": ["192.168.1.1", "192.168.1.2"]}`,
					Type:   types.RecordTypeA,
				},
				apiExpect: e2eAPIExpect{
					ShouldStatusOK:       true,
					ShouldTokenEmpty:     false,
					ShouldRootValueEmpty: false,
					ShouldHasSubDomain:   false,
					ShouldWildcard:       false,
				},
				dnsExpect: e2eDNSExpect{
					RootAddresses: []string{"192.168.1.1", "192.168.1.2"},
				},
			},
			{
				payload: e2ePayload{
					Method: http.MethodPost,
					URI:    "/v1/domain",
					Body:   `{"hosts": ["192.168.1.1", "192.168.1.2"], "subdomain": {"test1": ["192.168.1.3", "192.168.1.4"]}}`,
					Type:   types.RecordTypeA,
				},
				apiExpect: e2eAPIExpect{
					ShouldStatusOK:       true,
					ShouldTokenEmpty:     false,
					ShouldRootValueEmpty: false,
					ShouldHasSubDomain:   true,
					ShouldWildcard:       false,
				},
				dnsExpect: e2eDNSExpect{
					RootAddresses: []string{"192.168.1.1", "192.168.1.2"},
					SubAddresses: map[string][]string{
						"test1": {"192.168.1.3", "192.168.1.4"},
					},
				},
			},
			{
				payload: e2ePayload{
					Method: http.MethodPost,
					URI:    "/v1/domain",
					Body:   `{"hosts": ["192.168.1.1", "192.168.1.2"], "subdomain": {"test1": ["192.168.1.3", "192.168.1.4"], "test2": ["192.168.1.5"]}}`,
					Type:   types.RecordTypeA,
				},
				apiExpect: e2eAPIExpect{
					ShouldStatusOK:       true,
					ShouldTokenEmpty:     false,
					ShouldRootValueEmpty: false,
					ShouldHasSubDomain:   true,
					ShouldWildcard:       false,
				},
				dnsExpect: e2eDNSExpect{
					RootAddresses: []string{"192.168.1.1", "192.168.1.2"},
					SubAddresses: map[string][]string{
						"test1": {"192.168.1.3", "192.168.1.4"},
						"test2": {"192.168.1.5"},
					},
				},
			},
			{
				payload: e2ePayload{
					Method: http.MethodPost,
					URI:    "/v1/domain",
					Body:   `{"subdomain": {"test1": ["192.168.1.3", "192.168.1.4"], "test2": ["192.168.1.5"]}}`,
					Type:   types.RecordTypeA,
				},
				apiExpect: e2eAPIExpect{
					ShouldStatusOK:       true,
					ShouldTokenEmpty:     false,
					ShouldRootValueEmpty: true,
					ShouldHasSubDomain:   true,
					ShouldWildcard:       false,
				},
				dnsExpect: e2eDNSExpect{
					SubAddresses: map[string][]string{
						"test1": {"192.168.1.3", "192.168.1.4"},
						"test2": {"192.168.1.5"},
					},
				},
			},
			{
				payload: e2ePayload{
					Method: http.MethodPost,
					URI:    "/v1/domain",
					Body:   `{"fqdn":"test12345.rancher-cn-test.com", "hosts": ["192.168.1.1"]}`,
					Type:   types.RecordTypeA,
				},
				apiExpect: e2eAPIExpect{
					ShouldStatusOK:       true,
					ShouldTokenEmpty:     false,
					ShouldRootValueEmpty: false,
					ShouldHasSubDomain:   false,
					ShouldWildcard:       false,
				},
				dnsExpect: e2eDNSExpect{
					RootAddresses: []string{"192.168.1.1"},
				},
			},
			{
				payload: e2ePayload{
					Method: http.MethodPost,
					URI:    "/v1/domain",
					Body:   `{"fqdn":"*.test23456.rancher-cn-test.com", "hosts": ["192.168.1.1"]}`,
					Type:   types.RecordTypeA,
				},
				apiExpect: e2eAPIExpect{
					ShouldStatusOK:       true,
					ShouldTokenEmpty:     false,
					ShouldRootValueEmpty: false,
					ShouldHasSubDomain:   false,
					ShouldWildcard:       true,
				},
				dnsExpect: e2eDNSExpect{
					RootAddresses: []string{"192.168.1.1"},
				},
			},
			{
				payload: e2ePayload{
					Method: http.MethodPost,
					URI:    "/v1/domain/cname",
					Body:   `{"cname": "test1.rancher-cn-test.com"}`,
					Type:   types.RecordTypeCNAME,
				},
				apiExpect: e2eAPIExpect{
					ShouldStatusOK:       true,
					ShouldTokenEmpty:     false,
					ShouldRootValueEmpty: false,
					ShouldHasSubDomain:   false,
					ShouldWildcard:       false,
				},
				dnsExpect: e2eDNSExpect{
					CNAME: "test1.rancher-cn-test.com",
				},
			},
			{
				payload: e2ePayload{
					Method: http.MethodPost,
					URI:    "/v1/domain/cname",
					Body:   `{"fqdn": "test45678.rancher-cn-test.com", "cname": "test2.rancher-cn-test.com"}`,
					Type:   types.RecordTypeCNAME,
				},
				apiExpect: e2eAPIExpect{
					ShouldStatusOK:       true,
					ShouldTokenEmpty:     false,
					ShouldRootValueEmpty: false,
					ShouldHasSubDomain:   false,
					ShouldWildcard:       false,
				},
				dnsExpect: e2eDNSExpect{
					CNAME: "test2.rancher-cn-test.com",
				},
			},
			{
				payload: e2ePayload{
					Method: http.MethodPost,
					URI:    "/v1/domain/cname",
					Body:   `{"fqdn": "*.test56789.rancher-cn-test.com", "cname": "test3.rancher-cn-test.com"}`,
					Type:   types.RecordTypeCNAME,
				},
				apiExpect: e2eAPIExpect{
					ShouldStatusOK:       true,
					ShouldTokenEmpty:     false,
					ShouldRootValueEmpty: false,
					ShouldHasSubDomain:   false,
					ShouldWildcard:       true,
				},
				dnsExpect: e2eDNSExpect{
					CNAME: "test3.rancher-cn-test.com",
				},
			},
			{
				payload: e2ePayload{
					Method: http.MethodPost,
					URI:    "/v1/domain/txt",
					Body:   `{"text": "this is text msg"}`,
					Type:   types.RecordTypeTXT,
				},
				apiExpect: e2eAPIExpect{
					ShouldStatusOK:       true,
					ShouldTokenEmpty:     false,
					ShouldRootValueEmpty: false,
					ShouldHasSubDomain:   false,
					ShouldWildcard:       false,
				},
				dnsExpect: e2eDNSExpect{
					TXT: "this is text msg",
				},
			},
			{
				payload: e2ePayload{
					Method: http.MethodPost,
					URI:    "/v1/domain/txt",
					Body:   `{"fqdn": "test7890.rancher-cn-test.com", "text": "this is text msg 2"}`,
					Type:   types.RecordTypeTXT,
				},
				apiExpect: e2eAPIExpect{
					ShouldStatusOK:       true,
					ShouldTokenEmpty:     false,
					ShouldRootValueEmpty: false,
					ShouldHasSubDomain:   false,
					ShouldWildcard:       false,
				},
				dnsExpect: e2eDNSExpect{
					TXT: "this is text msg 2",
				},
			},
			{
				payload: e2ePayload{
					Method: http.MethodPost,
					URI:    "/v1/domain/txt",
					Body:   `{"fqdn": "*.test8901.rancher-cn-test.com", "text": "this is text msg 3"}`,
					Type:   types.RecordTypeTXT,
				},
				apiExpect: e2eAPIExpect{
					ShouldStatusOK:       true,
					ShouldTokenEmpty:     false,
					ShouldRootValueEmpty: false,
					ShouldHasSubDomain:   false,
					ShouldWildcard:       true,
				},
				dnsExpect: e2eDNSExpect{
					TXT: "this is text msg 3",
				},
			},
			{
				payload: e2ePayload{
					Method: http.MethodPost,
					URI:    "/v1/domain/txt",
					Auth:   domains[types.RecordTypeA]["test12345.rancher-cn-test.com"],
					Body:   `{"fqdn":"_acme-challenge.test12345.rancher-cn-test.com", "text": "acme challenge text msg"}`,
					Type:   types.RecordTypeTXT,
				},
				apiExpect: e2eAPIExpect{
					ShouldStatusOK:       true,
					ShouldTokenEmpty:     false,
					ShouldRootValueEmpty: false,
					ShouldHasSubDomain:   false,
					ShouldWildcard:       false,
				},
				dnsExpect: e2eDNSExpect{
					TXT: "acme challenge text msg",
				},
			},
		}
	})

	JustBeforeEach(func() {
		domains = make(map[string]map[string]string)
		client.SetHostURL(endpoint)
	})

	Describe("check insecure endpoints", func() {
		Context("check insecure /ping endpoint", func() {
			It("/ping endpoint should correctly", func() {
				resp, err := client.R().Get("/ping")
				if err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(resp.StatusCode()).To(Equal(http.StatusOK))
			})
		})
		Context("check insecure /healthz endpoint", func() {
			It("/healthz endpoint should correctly", func() {
				resp, err := client.R().Get("/healthz")
				if err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(resp.StatusCode()).To(Equal(http.StatusOK))
			})
		})
		Context("check insecure /metrics endpoint", func() {
			It("/metrics endpoint should correctly", func() {
				resp, err := client.R().Get("/metrics")
				if err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(resp.StatusCode()).To(Equal(http.StatusOK))
			})
		})
	})

	Describe("check secure domain apis", func() {
		It("check secure domain apis should correctly", func() {
			for _, c := range cases {
				switch c.payload.Method {
				case http.MethodPost:
					resp, err := client.R().SetBody(c.payload.Body).SetAuthToken(c.payload.Auth).Post(c.payload.URI)
					if err != nil {
						Expect(err).NotTo(HaveOccurred())
					}

					if c.apiExpect.ShouldStatusOK {
						Expect(resp.StatusCode()).To(Equal(http.StatusOK))
					} else {
						Expect(resp.StatusCode()).NotTo(Equal(http.StatusOK))
					}

					// in order to make dns take effect, maybe lead to slow test.
					time.Sleep(lookupTimeout)

					response := &types.Response{}
					if err := json.Unmarshal(resp.Body(), response); err != nil {
						Expect(err).NotTo(HaveOccurred())
					}

					if len(domains[c.payload.Type]) <= 0 {
						domains[c.payload.Type] = make(map[string]string)
					}
					domains[c.payload.Type][response.Data.Fqdn] = response.Data.Token

					logrus.Infof("success post %s domain request: %s: %s\n",
						c.payload.Type, response.Data.Fqdn, response.Data.Token)

					if c.apiExpect.ShouldTokenEmpty {
						Expect(response.Token).To(BeEmpty())
					} else {
						Expect(response.Token).NotTo(BeEmpty())
					}

					payload := &types.Payload{
						Fqdn: response.Data.Fqdn,
						Type: c.payload.Type,
					}
					if err := json.Unmarshal([]byte(c.payload.Body), payload); err != nil {
						Expect(err).NotTo(HaveOccurred())
					}

					if c.apiExpect.ShouldWildcard {
						Expect(response.Data.Fqdn).To(ContainSubstring("*"))
					} else {
						Expect(response.Data.Fqdn).NotTo(ContainSubstring("*"))
					}

					if c.apiExpect.ShouldRootValueEmpty {
						Expect(response.Data.Hosts).To(BeEmpty())
					} else {
						if payload.Type == types.RecordTypeA || payload.Type == types.RecordTypeAAAA {
							Expect(response.Data.Hosts).NotTo(BeEmpty())
							Expect(response.Data.Hosts).Should(ConsistOf(payload.Hosts))
						} else if payload.Type == types.RecordTypeCNAME {
							Expect(utils.TrimTrailingDot(response.Data.CNAME)).NotTo(BeEmpty())
							Expect(utils.TrimTrailingDot(response.Data.CNAME)).To(Equal(payload.CNAME))
						} else if payload.Type == types.RecordTypeTXT {
							Expect(utils.TextRemoveQuotes(utils.TrimTrailingDot(response.Data.Text))).NotTo(BeEmpty())
							Expect(utils.TextRemoveQuotes(utils.TrimTrailingDot(response.Data.Text))).To(Equal(payload.Text))
						}

						rr, err := DNSLookup(payload.Fqdn, payload.Type)
						if err != nil {
							Expect(err).NotTo(HaveOccurred())
						}
						ExpectDNSByType(payload.Type, c.dnsExpect, rr, true, "")

						if !c.apiExpect.ShouldWildcard && payload.Type != types.RecordTypeTXT && payload.Type != types.RecordTypeCNAME {
							rr, err = DNSLookup("*."+payload.Fqdn, payload.Type)
							if err != nil {
								Expect(err).NotTo(HaveOccurred())
							}
							ExpectDNSByType(payload.Type, c.dnsExpect, rr, true, "")
						}
					}

					if c.apiExpect.ShouldHasSubDomain {
						Expect(len(response.Data.SubDomain)).NotTo(BeZero())
						for k, v := range response.Data.SubDomain {
							Expect(payload.SubDomain).Should(HaveKeyWithValue(k, v))
							rr, err := DNSLookup(k+"."+payload.Fqdn, payload.Type)
							if err != nil {
								Expect(err).NotTo(HaveOccurred())
							}
							ExpectDNSByType(payload.Type, c.dnsExpect, rr, false, k)
						}
					} else {
						Expect(len(response.Data.SubDomain)).To(BeZero())
					}
				case http.MethodPut:
					if strings.Contains(c.payload.URI, "/renew") {

					} else {

					}
				case http.MethodGet:
					if c.payload.Type == types.RecordTypeA || c.payload.Type == types.RecordTypeAAAA {

					} else if c.payload.Type == types.RecordTypeCNAME {

					} else if c.payload.Type == types.RecordTypeTXT {

					}
				case http.MethodDelete:
					if c.payload.Type == types.RecordTypeA || c.payload.Type == types.RecordTypeAAAA {

					} else if c.payload.Type == types.RecordTypeCNAME {

					} else if c.payload.Type == types.RecordTypeTXT {

					}
				default:
					Expect(errors.New("request method not allowed")).NotTo(HaveOccurred())
				}
			}
		})
	})
})

func DNSLookup(domain, dnsType string) ([]dns.RR, error) {
	config, _ := dns.ClientConfigFromFile(resolvConf)

	c := new(dns.Client)
	m := new(dns.Msg)

	c.Timeout = dnsTimeout
	m.SetQuestion(dns.Fqdn(domain), DNSTypeToUInt16(dnsType))
	m.RecursionDesired = true

	r, _, err := c.Exchange(m, net.JoinHostPort(config.Servers[0], config.Port))
	if r == nil {
		return nil, errors.Wrapf(err, "failed to exchange msg\n")
	}

	if r.Rcode != dns.RcodeSuccess {
		return nil, errors.Errorf("invalid answer for %s\n", domain)
	}

	return r.Answer, nil
}

func DNSTypeToUInt16(dnsType string) uint16 {
	switch dnsType {
	case types.RecordTypeA:
		return dns.TypeA
	case types.RecordTypeAAAA:
		return dns.TypeAAAA
	case types.RecordTypeCNAME:
		return dns.TypeCNAME
	case types.RecordTypeTXT:
		return dns.TypeTXT
	default:
		return dns.TypeNone
	}
}

func ExpectDNSByType(dnsType string, expect e2eDNSExpect, rr []dns.RR, root bool, key string) {
	ss := make([]string, 0)
	for _, r := range rr {
		sp := strings.Split(utils.TextRemoveQuotes(utils.TrimTrailingDot(r.String())), "\t")
		ss = append(ss, sp[len(sp)-1:]...)
		for i, s := range ss {
			ss[i] = utils.TextRemoveQuotes(s)
		}
	}
	switch dnsType {
	case types.RecordTypeA:
		if root {
			Expect(ss).Should(ConsistOf(expect.RootAddresses))
		} else {
			Expect(ss).Should(ConsistOf(expect.SubAddresses[key]))
		}
	case types.RecordTypeAAAA:
		if root {
			Expect(ss).Should(ConsistOf(expect.RootAddresses))
		} else {
			Expect(ss).Should(ConsistOf(expect.SubAddresses[key]))
		}
	case types.RecordTypeCNAME:
		Expect(ss).Should(ConsistOf([]string{expect.CNAME}))
	case types.RecordTypeTXT:
		Expect(ss).Should(ConsistOf(expect.TXT))
	default:
		Expect(errors.New("dns type invalid")).NotTo(HaveOccurred())
	}
}
