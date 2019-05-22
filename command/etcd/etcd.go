package etcd

import (
	"net/http"
	"strings"

	"github.com/rancher/rdns-server/backend"
	"github.com/rancher/rdns-server/backend/etcd"
	"github.com/rancher/rdns-server/service"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:   "domain, do",
			EnvVar: "DOMAIN",
			Usage:  "used to set etcd domain.",
			Value:  "lb.rancher.cloud",
		},
		cli.StringFlag{
			Name:   "endpoints, ep",
			EnvVar: "ENDPOINTS",
			Usage:  "used to set etcd endpoints.",
			Value:  "http://127.0.0.1:2379",
		},
		cli.StringFlag{
			Name:   "path, p",
			EnvVar: "PATH",
			Usage:  "used to set etcd prefix path.",
			Value:  "/rdnsv3",
		},
	}
}

func Action(c *cli.Context) error {
	if c.Bool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	}
	ep := strings.Split(c.String("endpoints"), ",")
	b, err := etcd.NewBackend(ep, c.String("path"), c.String("domain"), c.String("frozen"))
	if err != nil {
		return errors.Wrapf(err, "Failed to init backend %s", etcd.Name)
	}
	defer func() {
		if err := b.ClientV3.Close(); err != nil {
			logrus.Fatalf("Failed to close etcd client: %v", err)
		}
	}()
	backend.SetBackend(b)
	done := make(chan error)
	go func() {
		r := service.NewRouter(c.String("domain"))
		done <- http.ListenAndServe(c.String("listen"), r)
	}()
	return <-done
}
