package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var collectors = []prometheus.Collector{
	LatestCandleCollected,
	LatestCandleQueried,
	OrderBookUpdates,
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

	OrderBookUpdates = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespaceFreyr,
			Name:      "order_book_updates_total",
			Help:      "The total number of updates the specified order book received since it was first completed.",
		},
		[]string{"exchange", "pair"},
	)
)
