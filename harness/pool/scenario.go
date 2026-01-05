package pool

import (
	"context"
	"fmt"
	"time"
)

type FollowUpScenario struct {
	Scenario Scenario
	Chance   float64
}

type ScenarioEmitter interface {
	ExecuteScenario(s Scenario)
}

type Scenario interface {
	Run(ctx context.Context, p *Player, e ScenarioEmitter) error
	Name() string
	GetFollowUpScenarios() []FollowUpScenario
}

type LoginScenario struct{}

func (LoginScenario) Run(ctx context.Context, p *Player, e ScenarioEmitter) error {
	fmt.Printf("[player %d] logging in\n", p.id)
	return Login(p.id, "password")
}

func (LoginScenario) Name() string {
	return "login"
}

func (LoginScenario) GetFollowUpScenarios() []FollowUpScenario {
	return nil
}

type MatchmakingScenario struct{}

func (MatchmakingScenario) Run(ctx context.Context, p *Player, e ScenarioEmitter) error {
	fmt.Printf("[player %d] matchmaking\n", p.id)
	// TODO: matchmaking calls
	return sleepOrCancel(ctx, 500*time.Millisecond)
}

func (MatchmakingScenario) Name() string {
	return "matchmaking"
}

func (MatchmakingScenario) GetFollowUpScenarios() []FollowUpScenario {
	return nil
}

type FetchStoreScenario struct{}

func (FetchStoreScenario) Run(ctx context.Context, p *Player, e ScenarioEmitter) error {
	fmt.Printf("[player %d] fetch store\n", p.id)
	return FetchStore()
}

func (FetchStoreScenario) Name() string {
	return "fetch_store"
}

func (FetchStoreScenario) GetFollowUpScenarios() []FollowUpScenario {
	return []FollowUpScenario{
		{
			Scenario: StorePurchaseScenario{},
			Chance:   0.1,
		},
	}
}

type StorePurchaseScenario struct{}

func (StorePurchaseScenario) Run(ctx context.Context, p *Player, e ScenarioEmitter) error {
	fmt.Printf("[player %d] store purchase\n", p.id)
	return StorePurchase(p.id)
}

func (StorePurchaseScenario) Name() string {
	return "store_purchase"
}

func (StorePurchaseScenario) GetFollowUpScenarios() []FollowUpScenario {
	return nil
}

// LogoutScenario causes a player to log out by canceling its context.
type LogoutScenario struct{}

func (LogoutScenario) Run(ctx context.Context, p *Player, e ScenarioEmitter) error {
	fmt.Printf("[player %d] logging out\n", p.id)
	// Call the player's cancel function to terminate its run goroutine.
	p.cancel()
	return nil
}

func (LogoutScenario) Name() string {
	return "logout"
}

func (LogoutScenario) GetFollowUpScenarios() []FollowUpScenario {
	return nil
}

func sleepOrCancel(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
