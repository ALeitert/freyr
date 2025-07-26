package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var collectors = []prometheus.Collector{
	LatestCandleCollected,
	LatestCandleQueried,
}

const (
	namespaceFreyr string = "freyr"
)

var (
	LatestCandleCollected = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespaceFreyr,
			Name:      "latest_candle_collected_timestamp_seconds",
			Help:      "The Unix timestamp of the latest candle collected for the specified trading pair.",
		},
		[]string{"exchange", "pair"},
	)

	LatestCandleQueried = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespaceFreyr,
			Name:      "latest_candle_queried_timestamp_seconds",
			Help:      "The Unix timestamp of the latest candle queried (but not necessarily collected) for the specified trading pair.",
		},
		[]string{"exchange", "pair"},
	)
)
