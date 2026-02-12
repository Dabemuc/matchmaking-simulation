package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	OngoingMatches = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "game_orchestrator_ongoing_matches",
			Help: "Number of ongoing matches managed by the orchestrator",
		},
	)
)

func init() {
	prometheus.MustRegister(OngoingMatches)
}
