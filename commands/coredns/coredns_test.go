package coredns_test

import (
	"io/ioutil"
	"os"

	"github.com/rancher/rdns-server/commands"
	"github.com/rancher/rdns-server/commands/coredns"
	"github.com/rancher/rdns-server/commands/global"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli"
)

var _ = Describe("coredns", func() {
	var (
		app   *cli.App
		cases []struct {
			args   []string
			expect map[string]string
		}
	)

	BeforeEach(func() {
		cases = []struct {
			args   []string
			expect map[string]string
		}{
			{
				args: []string{
					"rdns-server",
					"coredns",
					"--etcd_endpoints=http://127.0.0.1:2379",
					"--etcd_prefix=/rdnsv3",
					"--core_dns_file=someplace",
					"--core_dns_port=53",
					"--core_dns_cpu=4",
					"--core_dns_db_file=someplace",
					"--core_dns_db_zone=rancher.example",
					"--ttl=10",
				},
				expect: map[string]string{
					"ETCD_ENDPOINTS":   "http://127.0.0.1:2379",
					"ETCD_PREFIX":      "/rdnsv3",
					"CORE_DNS_FILE":    "someplace",
					"CORE_DNS_PORT":    "53",
					"CORE_DNS_CPU":     "4",
					"CORE_DNS_DB_FILE": "someplace",
					"CORE_DNS_DB_ZONE": "rancher.example",
					"TTL":              "10",
				},
			},
		}
	})

	JustBeforeEach(func() {
		app = cli.NewApp()
		app.Writer = ioutil.Discard
		app.Name = "rdns-server"
		cs := make([]cli.Command, 0)
		for _, cmd := range commands.MainCommands {
			// ignore command's action.
			cmd.Action = func(c *cli.Context) error {
				return coredns.SetEnvironments(c)
			}
			cs = append(cs, cmd)
		}
		app.Commands = cs
		app.Flags = global.Flags()
	})

	Describe("check coredns environments", func() {
		It("check coredns environments should correctly", func() {
			for _, c := range cases {
				err := app.Run(c.args)
				if err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
				for k, v := range c.expect {
					Expect(os.Getenv(k)).To(Equal(v))
				}
			}
		})
	})
})
