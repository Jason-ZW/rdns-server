package main

import (
	"fmt"
	"os"

	"github.com/rancher/rdns-server/command/etcd"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	DNSVersion = "v0.4.8"
	DNSDate    string
)

func init() {
	cli.VersionPrinter = versionPrinter
}

func main() {
	app := cli.NewApp()
	app.Author = "Rancher Labs, Inc."
	app.Before = beforeFunc
	app.EnableBashCompletion = true
	app.HideHelp = true
	app.Name = os.Args[0]
	app.Usage = fmt.Sprintf("control and configure RDNS(%s)", DNSDate)
	app.Version = DNSVersion
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "debug, d",
			EnvVar: "DEBUG",
			Usage:  "used to set debug mode.",
		},
		cli.StringFlag{
			Name:   "listen, l",
			EnvVar: "LISTEN",
			Usage:  "used to set listen port.",
			Value:  ":9333",
		},
		cli.StringFlag{
			Name:   "frozen, f",
			EnvVar: "FROZEN",
			Usage:  "used to set domain frozen time.",
			Value:  "2160h",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:    "etcd",
			Aliases: []string{"e"},
			Usage:   "use etcd backend",
			Flags:   etcd.Flags(),
			Action:  etcd.Action,
		},
	}
	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func beforeFunc(c *cli.Context) error {
	if os.Getuid() != 0 {
		logrus.Fatalf("%s: need to be root", os.Args[0])
	}
	return nil
}

func versionPrinter(c *cli.Context) {
	if _, err := fmt.Fprintf(c.App.Writer, DNSVersion); err != nil {
		logrus.Error(err)
	}
}
