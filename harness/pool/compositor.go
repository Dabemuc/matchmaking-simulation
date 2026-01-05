package pool

import (
	"context"
	"errors"
	"time"
)

type scenarioEntry struct {
	scenario    Scenario
	rate        float64
	accumulator float64
}

type Compositor struct {
	pool      *Pool
	scenarios []*scenarioEntry
}

func NewCompositor(p *Pool) *Compositor {
	return &Compositor{
		pool: p,
	}
}

func (c *Compositor) AddScenario(s Scenario, perPlayerRate float64) {
	c.scenarios = append(c.scenarios, &scenarioEntry{
		scenario: s,
		rate:     perPlayerRate,
	})
	compositorDesiredRate.WithLabelValues(s.Name()).Set(perPlayerRate)
}

func (c *Compositor) Start(ctx context.Context) {
	go c.run(ctx)
}

func (c *Compositor) run(ctx context.Context) {
	tickInterval := 100 * time.Millisecond
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		select {
		case tickTime := <-ticker.C:
			tickStart := time.Now()
			lag := tickStart.Sub(tickTime).Seconds()
			playerCount := c.pool.PlayerCount()

			for _, entry := range c.scenarios {
				compositorTickLagSeconds.WithLabelValues(entry.scenario.Name()).Set(lag)

				if playerCount == 0 {
					continue
				}

				earned := float64(playerCount) * entry.rate * tickInterval.Seconds()
				entry.accumulator += earned
				executions := int(entry.accumulator)
				entry.accumulator -= float64(executions)

				if executions > 0 {
					compositorTickExecutions.WithLabelValues(entry.scenario.Name()).Observe(float64(executions))

					// FIXED: We pass a timeout to the dispatch goroutine to prevent leaks
					go func(scen Scenario, count int) {
						for i := 0; i < count; i++ {
							// Each dispatch attempt has a deadline to prevent goroutine piling
							dispatchCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)

							scenarioAttemptedTotal.WithLabelValues(scen.Name()).Inc()
							err := c.pool.DoExecuteScenario(dispatchCtx, scen)

							if err != nil {
								if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrNoPlayerAvailable) {
									compositorIdleStarvationTotal.WithLabelValues(scen.Name()).Inc()
								} else if !errors.Is(err, context.Canceled) {
									errorsTotal.WithLabelValues("compositor", "execute_scenario").Inc()
								}
							}
							cancel()
						}
					}(entry.scenario, executions)
				}
			}
			tickDuration.WithLabelValues("compositor").Observe(time.Since(tickStart).Seconds())

		case <-ctx.Done():
			return
		}
	}
}
