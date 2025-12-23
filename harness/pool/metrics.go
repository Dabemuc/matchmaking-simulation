package pool

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	activePlayers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_players",
			Help: "Current number of active players",
		},
	)

	scenarioExecutions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scenario_executions_total",
			Help: "Total number of times a scenario was executed",
		},
		[]string{"scenario"},
	)
)

func init() {
	prometheus.MustRegister(activePlayers)
	prometheus.MustRegister(scenarioExecutions)
}
