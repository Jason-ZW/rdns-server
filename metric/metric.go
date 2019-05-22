package metric

import (
	"time"

	"github.com/rancher/rdns-server/database"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	queryDuration = 5 * time.Second

	tokenGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "rdns_tokens",
		Help: "The number of the rdns tokens",
	})
)

func Metrics(done chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
			count, err := database.GetDatabase().QueryTokenCount()
			if err != nil {
				logrus.Errorf("failed to operate database TOKEN record count: %s", err.Error())
			}
			tokenGauge.Set(float64(count))
			time.Sleep(queryDuration)
		}
	}
}
