package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	OngoingMatches = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gateway_ongoing_matches",
			Help: "Number of ongoing matches",
		},
	)

	PlayersInMatchmaking = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gateway_players_in_matchmaking",
			Help: "Number of players currently in matchmaking",
		},
	)
)

func init() {
	// Register metrics with the global prometheus registry
	prometheus.MustRegister(OngoingMatches)
	prometheus.MustRegister(PlayersInMatchmaking)
}
