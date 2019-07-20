package main

import (
	"fmt"
	"os"

	"github.com/rancher/rdns-server/commands"
	"github.com/rancher/rdns-server/commands/global"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	DNSVersion = "v0.6.0"
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
	app.Name = os.Args[0]
	app.Usage = fmt.Sprintf("control and configure RDNS(%s)", DNSDate)
	app.Version = DNSVersion
	app.Flags = global.Flags()
	app.Commands = commands.MainCommands
	if err := app.Run(os.Args); err != nil {
		logrus.Fatalln(err)
	}
}

func beforeFunc(c *cli.Context) error {
	if os.Getuid() != 0 {
		logrus.Fatalf("%s: need to be root\n", os.Args[0])
	}
	return nil
}

func versionPrinter(c *cli.Context) {
	if _, err := fmt.Fprintf(c.App.Writer, DNSVersion); err != nil {
		logrus.Error(err)
	}
}
