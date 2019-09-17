package commands

import (
	route53 "github.com/rancher/rdns-server/commands/aws"
	"github.com/rancher/rdns-server/commands/coredns"

	"github.com/urfave/cli"
)

var (
	MainCommands = []cli.Command{
		PromptCommand(),
		{
			Name:   "route53",
			Usage:  "use aws route53 provider",
			Flags:  route53.Flags(),
			Action: route53.Action,
		},
		{
			Name:   "coredns",
			Usage:  "use coredns provider",
			Flags:  coredns.Flags(),
			Action: coredns.Action,
		},
	}
)
