package route53_test

import (
	"io/ioutil"
	"os"

	"github.com/rancher/rdns-server/commands"
	route53 "github.com/rancher/rdns-server/commands/aws"
	"github.com/rancher/rdns-server/commands/global"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli"
)

var _ = Describe("route53", func() {
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
				expect: map[string]string{
					"AWS_HOSTED_ZONE_ID":    "1234567890",
					"AWS_ACCESS_KEY_ID":     "ABCDEFGHIJKLMN",
					"AWS_SECRET_ACCESS_KEY": "ABCDEFGHIJKLMN",
					"AWS_ASSUME_ROLE":       "admin",
					"AWS_RETRY":             "3",
					"TTL":                   "10",
					"DB_MIGRATE":            "up",
					"DB_DSN":                "root@random@tcp(127.0.0.1:3306)/rdns?parseTime=true",
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
				return route53.SetEnvironments(c)
			}
			cs = append(cs, cmd)
		}
		app.Commands = cs
		app.Flags = global.Flags()
	})

	Describe("check route53 environments", func() {
		It("check route53 environments should correctly", func() {
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
