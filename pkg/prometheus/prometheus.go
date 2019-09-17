package prometheus

import (
	"time"

	"github.com/rancher/rdns-server/keepers"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	queryDuration = 5 * time.Second

	tokenGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "rancher_dns_tokens",
		Help: "The number of the rancher dns tokens",
	})
)

func StartMetricsDaemon(done chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
			count := keepers.GetKeeper().GetTokenCount()
			tokenGauge.Set(float64(count))
			time.Sleep(queryDuration)
		}
	}
}
