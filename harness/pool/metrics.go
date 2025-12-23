package pool

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// 1️⃣ Player lifecycle metrics
	playersTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "harness",
			Name:      "players_total",
			Help:      "Total players ever created.",
		},
	)
	playersActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "harness",
			Name:      "players_active",
			Help:      "Current number of alive players.",
		},
	)
	playersIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "harness",
			Name:      "players_idle",
			Help:      "Players currently idle and available for scenarios.",
		},
	)
	playerLoginTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "harness",
			Name:      "player_login_total",
			Help:      "Login attempts outcome.",
		},
		[]string{"status"},
	)
	playerLoginDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "harness",
			Name:      "player_login_duration_seconds",
			Help:      "Login latency.",
			Buckets:   prometheus.DefBuckets,
		},
	)

	// 2️⃣ Scenario execution metrics
	scenarioStartedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "harness",
			Name:      "scenario_started_total",
			Help:      "Scenario executions started.",
		},
		[]string{"scenario"},
	)
	scenarioCompletedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "harness",
			Name:      "scenario_completed_total",
			Help:      "Scenario outcomes.",
		},
		[]string{"scenario", "status"},
	)
	scenarioDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "harness",
			Name:      "scenario_duration_seconds",
			Help:      "Time spent executing each scenario.",
			Buckets:   []float64{0.05, 0.1, 0.2, 0.5, 1, 2, 5},
		},
		[]string{"scenario"},
	)
	scenariosInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "harness",
			Name:      "scenarios_in_flight",
			Help:      "Concurrent executions per scenario.",
		},
		[]string{"scenario"},
	)

	// 3️⃣ Compositor metrics
	compositorDesiredRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "harness",
			Name:      "compositor_desired_rate",
			Help:      "Desired executions/sec per player.",
		},
		[]string{"scenario"},
	)
	compositorTickExecutions = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "harness",
			Name:      "compositor_tick_executions",
			Help:      "Number of executions scheduled per tick.",
			Buckets:   prometheus.LinearBuckets(0, 1, 10),
		},
		[]string{"scenario"},
	)
	compositorIdleStarvationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "harness",
			Name:      "compositor_idle_starvation_total",
			Help:      "Scenario wanted to run but no idle player was available.",
		},
		[]string{"scenario"},
	)

	// 4️⃣ Pool pressure & backpressure
	poolExecuteWaitDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "harness",
			Name:      "pool_execute_wait_duration_seconds",
			Help:      "Time waiting for an idle player.",
			Buckets:   prometheus.DefBuckets,
		},
	)
	poolIdleQueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "harness",
			Name:      "pool_idle_queue_depth",
			Help:      "Current length of idle channel.",
		},
	)
	// 5️⃣ Cancellation & shutdown safety
	contextCancellationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "harness",
			Name:      "context_cancellations_total",
			Help:      "Where cancellations occurred.",
		},
		[]string{"source"},
	)
	goroutines = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "harness",
			Name:      "goroutines",
			Help:      "Number of goroutines.",
		},
	)

	// 6️⃣ Optional but powerful
	tickDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "harness",
			Name:      "tick_duration_seconds",
			Help:      "Internal scheduler loop duration.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"component"},
	)
	errorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "harness",
			Name:      "errors_total",
			Help:      "Unexpected internal errors.",
		},
		[]string{"component", "reason"},
	)
)

func init() {
	// All metrics are now using promauto, so explicit registration is not needed.
}
