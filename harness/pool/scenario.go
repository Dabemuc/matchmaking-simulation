package pool

import (
	"context"
	"fmt"
	"time"
)

type Scenario interface {
	Run(ctx context.Context, p *Player) error
}

type LoginScenario struct{}

func (LoginScenario) Run(ctx context.Context, p *Player) error {
	fmt.Printf("[player %d] logging in\n", p.id)
	// TODO: auth backend call
	return sleepOrCancel(ctx, 300*time.Millisecond)
}

type MatchmakingScenario struct{}

func (MatchmakingScenario) Run(ctx context.Context, p *Player) error {
	fmt.Printf("[player %d] matchmaking\n", p.id)
	// TODO: matchmaking calls
	return sleepOrCancel(ctx, 500*time.Millisecond)
}

type StorePurchaseScenario struct{}

func (StorePurchaseScenario) Run(ctx context.Context, p *Player) error {
	fmt.Printf("[player %d] store purchase\n", p.id)
	// TODO: store backend calls
	return sleepOrCancel(ctx, 400*time.Millisecond)
}

func sleepOrCancel(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
