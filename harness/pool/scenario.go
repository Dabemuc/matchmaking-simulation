package pool

import (
	"context"
	"fmt"
	"time"
)

type Scenario interface {
	Run(ctx context.Context, p *Player) error
	Name() string
}

type LoginScenario struct{}

func (LoginScenario) Run(ctx context.Context, p *Player) error {
	fmt.Printf("[player %d] logging in\n", p.id)
	// TODO: auth backend call
	return sleepOrCancel(ctx, 300*time.Millisecond)
}

func (LoginScenario) Name() string {
	return "login"
}

type MatchmakingScenario struct{}

func (MatchmakingScenario) Run(ctx context.Context, p *Player) error {
	fmt.Printf("[player %d] matchmaking\n", p.id)
	// TODO: matchmaking calls
	return sleepOrCancel(ctx, 500*time.Millisecond)
}

func (MatchmakingScenario) Name() string {
	return "matchmaking"
}

type StorePurchaseScenario struct{}

func (StorePurchaseScenario) Run(ctx context.Context, p *Player) error {
	fmt.Printf("[player %d] store purchase\n", p.id)
	// TODO: store backend calls
	return sleepOrCancel(ctx, 400*time.Millisecond)
}

func (StorePurchaseScenario) Name() string {
	return "store_purchase"
}

// LogoutScenario causes a player to log out by canceling its context.
type LogoutScenario struct{}

func (LogoutScenario) Run(ctx context.Context, p *Player) error {
	fmt.Printf("[player %d] logging out\n", p.id)
	// Call the player's cancel function to terminate its run goroutine.
	p.cancel()
	return nil
}

func (LogoutScenario) Name() string {
	return "logout"
}

func sleepOrCancel(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}