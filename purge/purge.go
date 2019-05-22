package purge

import (
	"fmt"
	"os"
	"time"

	"github.com/rancher/rdns-server/backend"
	"github.com/rancher/rdns-server/database"
	"github.com/rancher/rdns-server/model"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	intervalSeconds int64 = 600
)

type purger struct {
}

func StartPurgerDaemon(done chan struct{}) {
	p := &purger{}
	go wait.JitterUntil(p.purge, time.Duration(intervalSeconds)*time.Second, .1, true, done)
}

func (p *purger) purge() {
	frozen, err := time.ParseDuration(os.Getenv("FROZEN"))
	if err != nil {
		logrus.Fatalf("failed to parse flag: %s", "frozen")
	}

	ttl, err := time.ParseDuration(os.Getenv("TTL"))
	if err != nil {
		logrus.Fatalf("failed to parse flag: %s", "ttl")
	}

	// check frozen records, delete the frozen record which is expired
	if err := database.GetDatabase().DeleteFrozenByTime(subtractExpiration(time.Now(), int(frozen.Seconds()))); err != nil {
		logrus.Errorf("failed to delete expired FROZEN records: %s", err.Error())
	}

	// check token records, delete the token record which is expired
	// this ensures that associated records are also deleted
	tokens, err := database.GetDatabase().QueryExpiredTokens(subtractExpiration(time.Now(), int(ttl.Seconds())))
	if err != nil {
		logrus.Errorf("failed to get expired TOKEN records: %s", err.Error())
	}

	for _, token := range tokens {
		// delete route53 A records & sub A records & wildcard records
		opts := &model.DomainOptions{
			Fqdn: token.Fqdn,
		}
		a, err := backend.GetBackend().Get(opts)
		if err == nil && a.Fqdn != "" {
			if err := backend.GetBackend().Delete(opts); err != nil {
				logrus.Errorf("failed to delete expired A record %s: %s", opts.Fqdn, err.Error())
				continue
			}
		}

		// delete route53 TXT records
		ts, err := database.GetDatabase().QueryExpiredRecordTXTs(token.ID)
		for _, t := range ts {
			tOpts := &model.DomainOptions{
				Fqdn: t.Fqdn,
			}
			if err := backend.GetBackend().DeleteText(tOpts); err != nil {
				logrus.Errorf("failed to delete expired TXT record %s: %s", tOpts.Fqdn, err.Error())
				continue
			}
		}

		// delete token records & referenced records
		if err := database.GetDatabase().DeleteToken(token.Token); err != nil {
			logrus.Errorf("failed to delete expired TOKEN record %s: %s", token.Token, err.Error())
		}
	}
}

func subtractExpiration(create time.Time, ttl int) *time.Time {
	duration, _ := time.ParseDuration(fmt.Sprintf("%ds", ttl))
	e := create.Add(-duration)
	return &e
}
