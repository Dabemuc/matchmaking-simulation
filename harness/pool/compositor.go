package pool

import (
	"context"
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
			playerCount := c.pool.PlayerCount()
			if playerCount == 0 {
				continue
			}

			// total executions per second
			executions := int(float64(playerCount) * entry.rate)

			for i := 0; i < executions; i++ {
				go c.pool.ExecuteScenario(ctx, entry.scenario)
			}

		case <-ctx.Done():
			return
		}
	}
}
