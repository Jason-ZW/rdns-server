package expires

import (
	"fmt"
	"github.com/rancher/rdns-server/types"
	"os"
	"time"

	"github.com/rancher/rdns-server/keepers"
	"github.com/rancher/rdns-server/pkg/consts"
	"github.com/rancher/rdns-server/providers"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

// expireDaemon is used to periodic recovery of expired data
// only route53 provider need using this daemon.
type expireDaemon struct {
	expire string
	rotate string
}

func StartExpireDaemon(done chan struct{}) {
	e := &expireDaemon{
		expire: os.Getenv(consts.EnvExpire),
		rotate: os.Getenv(consts.EnvRotate),
	}
	go wait.JitterUntil(e.run, time.Duration(consts.ExpireIntervalSeconds)*time.Second, .1, true, done)
}

func (e *expireDaemon) run() {
	// processing keeper records, this is database operation.
	keeper := keepers.GetKeeper()
	if err := keeper.DeleteExpiredRotate(e.calculateRotate()); err != nil {
		logrus.Errorln("failed to purge expired previously randomly generated dns name")
	}

	tokens, err := keeper.GetExpiredTokens(e.calculateExpire())
	if err != nil {
		logrus.Errorln("failed to get expired tokens")
	}

	// processing route53 records.
	for _, token := range tokens {
		payload := types.Payload{
			Fqdn: token.Domain,
		}

		// TODO: temporarily iterate over the deletion, and adjust later.
		payload.Type = types.RecordTypeA
		providers.GetProvider().Delete(payload)
		payload.Type = types.RecordTypeAAAA
		providers.GetProvider().Delete(payload)
		payload.Type = types.RecordTypeTXT
		providers.GetProvider().DeleteTXT(payload)
		payload.Type = types.RecordTypeCNAME
		providers.GetProvider().DeleteCNAME(payload)
	}

	if err := keeper.DeleteExpiredTokens(e.calculateExpire()); err != nil {
		logrus.Errorln("failed to purge expired tokens")
	}
}

func (e *expireDaemon) calculateRotate() *time.Time {
	r, err := time.ParseDuration(e.rotate)
	if err != nil {
		logrus.Fatalln("failed to parse duration")
	}
	d, _ := time.ParseDuration(fmt.Sprintf("%dns", int(r.Nanoseconds())))
	t := time.Now().Add(-d)
	return &t
}

func (e *expireDaemon) calculateExpire() *time.Time {
	d, err := time.ParseDuration(e.expire)
	if err != nil {
		logrus.Fatalf("failed to parse duration")
	}
	duration, _ := time.ParseDuration(fmt.Sprintf("%dns", int(d.Nanoseconds())))
	t := time.Now().Add(-duration)
	return &t
}
