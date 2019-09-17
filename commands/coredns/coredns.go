package coredns

import (
	"os"
	"strings"

	"github.com/rancher/rdns-server/commands/global"
	"github.com/rancher/rdns-server/routers"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	flags = map[string]map[string]string{
		"ETCD_ENDPOINTS":   {"etcd endpoints.": "http://127.0.0.1:2379"},
		"ETCD_PREFIX":      {"etcd prefix.": "/rdnsv3"},
		"CORE_DNS_FILE":    {"coredns file.": "/etc/rdns/config/Corefile"},
		"CORE_DNS_PORT":    {"coredns listen port.": "53"},
		"CORE_DNS_CPU":     {"coredns cpu, a number (e.g. 3) or a percent (e.g. 50%).": "50%"},
		"CORE_DNS_DB_FILE": {"coredns file plugin db's file name (e.g. /etc/rdns/config/dbfile).": ""},
		"CORE_DNS_DB_ZONE": {"coredns file plugin db's zone (e.g. api.rancher.example).": ""},
		"TTL":              {"coredns record ttl.": "60"},
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
	if err := SetEnvironments(c); err != nil {
		return err
	}

	done := make(chan struct{})
	routers.NewRouter(done)

	return nil
}

func SetEnvironments(c *cli.Context) error {
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
			if k == "CORE_DNS_DB_FILE" || k == "CORE_DNS_DB_ZONE" {
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
