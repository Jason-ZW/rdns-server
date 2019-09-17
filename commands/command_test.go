package commands_test

import (
	"io/ioutil"
	"strings"

	"github.com/rancher/rdns-server/commands"
	"github.com/rancher/rdns-server/commands/global"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli"
)

var _ = Describe("command", func() {
	var (
		app   *cli.App
		cases []struct {
			args   []string
			expect map[string][]string
		}
	)

	BeforeEach(func() {
		cases = []struct {
			args   []string
			expect map[string][]string
		}{
			{
				args: []string{
					"rdns-server",
					"prompt",
				},
				expect: map[string][]string{
					"prompt": {},
				},
			},
			{
				args: []string{
					"rdns-server",
					"route53",
					"--aws_hosted_zone_id=1234567890",
					"--aws_access_key_id=ABCDEFGHIJKLMN",
					"--aws_secret_access_key=ABCDEFGHIJKLMN",
					"--aws_assume_role=admin",
					"--aws_retry=3",
					"--ttl=10",
					"--db_migrate=up",
					"--db_dsn=root@random@tcp(127.0.0.1:3306)/rdns?parseTime=true",
				},
				expect: map[string][]string{
					"route53": {
						"AWS_HOSTED_ZONE_ID",
						"AWS_ACCESS_KEY_ID",
						"AWS_SECRET_ACCESS_KEY",
						"AWS_ASSUME_ROLE",
						"AWS_RETRY",
						"TTL",
						"DB_MIGRATE",
						"DB_DSN",
					},
				},
			},
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
				expect: map[string][]string{
					"coredns": {
						"ETCD_ENDPOINTS",
						"ETCD_PREFIX",
						"CORE_DNS_FILE",
						"CORE_DNS_PORT",
						"CORE_DNS_CPU",
						"CORE_DNS_DB_FILE",
						"CORE_DNS_DB_ZONE",
						"TTL",
					},
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
				return nil
			}
			cs = append(cs, cmd)
		}
		app.Commands = cs
		app.Flags = global.Flags()
	})

	Describe("run main command", func() {
		It("run main command should correctly", func() {
			for n, c := range cases {
				err := app.Run(c.args)
				if err != nil {
					Expect(err).NotTo(HaveOccurred())
				}
				var expectCmd string
				for k := range c.expect {
					expectCmd = k
					break
				}
				cmd := app.Commands[n]
				Expect(cmd.Name).To(Equal(expectCmd))
				uppers := make([]string, 0)
				for _, flag := range cmd.Flags {
					uppers = append(uppers, strings.ToUpper(flag.GetName()))
				}
				Expect(uppers).Should(ConsistOf(c.expect[expectCmd]))
			}
		})
	})
})
