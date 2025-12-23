package pool

import (
	"context"
	"errors"
	"time"
)

type scenarioEntry struct {
	scenario Scenario
	rate     float64 // per-player executions per second
}

type Compositor struct {
	pool      *Pool
	scenarios []scenarioEntry
}

func NewCompositor(p *Pool) *Compositor {
	return &Compositor{
		pool: p,
	}
}

func (c *Compositor) AddScenario(s Scenario, perPlayerRate float64) {
	c.scenarios = append(c.scenarios, scenarioEntry{
		scenario: s,
		rate:     perPlayerRate,
	})
	compositorDesiredRate.WithLabelValues(s.Name()).Set(perPlayerRate)
}

func (c *Compositor) Start(ctx context.Context) {
	for _, entry := range c.scenarios {
		entry := entry // capture loop variable
		go c.run(ctx, entry)
	}
}

func (c *Compositor) run(ctx context.Context, entry scenarioEntry) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tickStart := time.Now()
			playerCount := c.pool.PlayerCount()
			if playerCount == 0 {
				continue
			}

			// total executions per second
			executions := int(float64(playerCount) * entry.rate)
			compositorTickExecutions.WithLabelValues(entry.scenario.Name()).Observe(float64(executions))

			for i := 0; i < executions; i++ {
				err := c.pool.ExecuteScenario(ctx, entry.scenario)
				if err != nil {
					if errors.Is(err, ErrNoPlayerAvailable) {
						compositorIdleStarvationTotal.WithLabelValues(entry.scenario.Name()).Inc()
					} else {
						errorsTotal.WithLabelValues("compositor", "execute_scenario").Inc()
					}
				}
			}
			tickDuration.WithLabelValues("compositor").Observe(time.Since(tickStart).Seconds())

		case <-ctx.Done():
			return
		}
	}
}
