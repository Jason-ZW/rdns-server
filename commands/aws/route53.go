package route53

import (
	"os"
	"strings"

	"github.com/rancher/rdns-server/commands/global"
	"github.com/rancher/rdns-server/providers"
	"github.com/rancher/rdns-server/providers/aws"
	"github.com/rancher/rdns-server/routers"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	flags = map[string]map[string]string{
		"AWS_HOSTED_ZONE_ID":    {"aws hosted zone ID.": ""},
		"AWS_ACCESS_KEY_ID":     {"aws access key ID.": ""},
		"AWS_SECRET_ACCESS_KEY": {"aws secret access key.": ""},
		"AWS_ASSUME_ROLE":       {"aws assume role.": ""},
		"AWS_RETRY":             {"aws retry times.": "3"},
		"TTL":                   {"route53 record ttl.": "60"},
	}
)

func Flags() []cli.Flag {
	fgs := make([]cli.Flag, 0)
	for key, value := range flags {
		for k, v := range value {
			f := cli.StringFlag{
				Name:   strings.ToLower(key),
				EnvVar: key,
				Usage:  k,
				Value:  v,
			}
			fgs = append(fgs, f)
		}
	}
	return fgs
}

func Action(c *cli.Context) error {
	if err := setEnvironments(c); err != nil {
		return err
	}

	provider := aws.NewR53Provider()
	providers.SetProvider(provider)

	done := make(chan struct{})
	routers.NewRouter(done)

	return nil
}

func setEnvironments(c *cli.Context) error {
	if level := c.GlobalString("level"); level != "" {
		l, err := logrus.ParseLevel(level)
		if err != nil {
			return err
		}
		if l == logrus.DebugLevel {
			logrus.SetReportCaller(true)
		}
		logrus.SetLevel(l)
	}

	for k := range flags {
		if err := os.Setenv(k, c.String(strings.ToLower(k))); err != nil {
			return err
		}
		if os.Getenv(k) == "" {
			if k == "AWS_ASSUME_ROLE" {
				continue
			}
			return errors.Errorf("expected argument: %s", strings.ToLower(k))
		}
	}

	for k := range global.GetFlags() {
		if err := os.Setenv(k, c.GlobalString(strings.ToLower(k))); err != nil {
			return err
		}
	}

	return nil
}
