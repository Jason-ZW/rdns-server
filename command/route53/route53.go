package route53

import (
	"net/http"
	"os"

	"github.com/rancher/rdns-server/backend"
	"github.com/rancher/rdns-server/backend/route53"
	"github.com/rancher/rdns-server/database"
	"github.com/rancher/rdns-server/database/mysql"
	"github.com/rancher/rdns-server/metric"
	"github.com/rancher/rdns-server/purge"
	"github.com/rancher/rdns-server/service"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:   "aws_hosted_zone_id",
			EnvVar: "AWS_HOSTED_ZONE_ID",
			Usage:  "used to set aws hosted zone ID.",
		},
		cli.StringFlag{
			Name:   "aws_access_key_id",
			EnvVar: "AWS_ACCESS_KEY_ID",
			Usage:  "used to set aws access key ID.",
		},
		cli.StringFlag{
			Name:   "aws_secret_access_key",
			EnvVar: "AWS_SECRET_ACCESS_KEY",
			Usage:  "used to set aws secret access key.",
		},
		cli.StringFlag{
			Name:   "database",
			EnvVar: "DATABASE",
			Usage:  "used to set database.",
			Value:  "mysql",
		},
		cli.StringFlag{
			Name:   "dsn",
			EnvVar: "DSN",
			Usage:  "used to set database dsn.",
		},
		cli.StringFlag{
			Name:   "ttl",
			EnvVar: "TTL",
			Usage:  "used to set record ttl.",
			Value:  "240h",
		},
	}
}

func Action(c *cli.Context) error {
	if c.GlobalBool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if err := setEnv(c); err != nil {
		return errors.Wrapf(err, "set environment error")
	}

	if c.String("database") == mysql.DataBaseName {
		d, err := mysql.NewDatabase(c.String("dsn"))
		if err != nil {
			return err
		}
		defer d.Close()
		database.SetDatabase(d)
	} else {
		return errors.New("no suitable database found")
	}

	b, err := route53.NewBackend()
	if err != nil {
		return err
	}
	backend.SetBackend(b)

	done := make(chan struct{})

	go metric.Metrics(done)

	go purge.StartPurgerDaemon(done)

	go func() {
		r := service.NewRouter()
		if err := http.ListenAndServe(c.GlobalString("listen"), r); err != nil {
			logrus.Error(err)
			done <- struct{}{}
		}
	}()

	<-done
	return nil
}

func setEnv(c *cli.Context) error {
	zID := c.String("aws_hosted_zone_id")
	kID := c.String("aws_access_key_id")
	key := c.String("aws_secret_access_key")
	db := c.String("database")
	dsn := c.String("dsn")
	ttl := c.String("ttl")
	frozen := c.GlobalString("frozen")

	if zID == "" || kID == "" || key == "" || dsn == "" {
		return errors.New("not enough arguments expected 4")
	}

	if err := os.Setenv("AWS_HOSTED_ZONE_ID", zID); err != nil {
		return err
	}
	if err := os.Setenv("AWS_ACCESS_KEY_ID", kID); err != nil {
		return err
	}
	if err := os.Setenv("AWS_SECRET_ACCESS_KEY", key); err != nil {
		return err
	}
	if err := os.Setenv("DATABASE", db); err != nil {
		return err
	}
	if err := os.Setenv("DSN", dsn); err != nil {
		return err
	}
	if err := os.Setenv("TTL", ttl); err != nil {
		return err
	}
	if err := os.Setenv("FROZEN", frozen); err != nil {
		return err
	}

	return nil
}
