package global

import (
	"strings"

	"github.com/urfave/cli"
)

var (
	flags = map[string]map[string]string{
		"LEVEL":  {"output log level.": "info"},
		"PORT":   {"server listen port.": "9333"},
		"DOMAIN": {"root domain (zone) which will be served.": "rancher.example"},
		"EXPIRE": {"record expire duration.": "240h"},
		"ROTATE": {"previously randomly generated dns name are not to be re-used during this period.": "2160h"},
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
	fgs = append(fgs, cli.HelpFlag)
	return fgs
}

func GetFlags() map[string]map[string]string {
	return flags
}
