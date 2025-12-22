package main

import (
	"context"
	"time"

	"harness/pool"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := pool.New(
		100*time.Millisecond, // login rate
		1000,                 // idle capacity
	)
	p.Init(ctx)

	compositor := pool.NewCompositor(p)

	compositor.AddScenario(pool.MatchmakingScenario{}, 0.1)
	compositor.AddScenario(pool.StorePurchaseScenario{}, 0.02)

	compositor.Start(ctx)

	time.Sleep(10 * time.Second)
	cancel()
	time.Sleep(1 * time.Second)
}
