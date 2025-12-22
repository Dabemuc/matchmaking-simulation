package pool

import (
	"context"
	"fmt"
)

type Player struct {
	id       int
	scenario chan Scenario
}

func newPlayer(id int) *Player {
	return &Player{
		id:       id,
		scenario: make(chan Scenario),
	}
}

func (p *Player) run(ctx context.Context, idle chan *Player) {
	// ---- PLAYER INITIALIZATION ----

	fmt.Printf("[player %d] starting up\n", p.id)

	// Every player must login before becoming idle
	login := LoginScenario{}
	if err := login.Run(ctx, p); err != nil {
		fmt.Printf("[player %d] login failed: %v\n", p.id, err)
		return
	}

	fmt.Printf("[player %d] login successful, entering idle pool\n", p.id)

	// ---- NORMAL PLAYER LOOP ----
	for {
		// mark player idle
		select {
		case idle <- p:
		case <-ctx.Done():
			return
		}

		// wait for scenario
		select {
		case s := <-p.scenario:
			_ = s.Run(ctx, p)
		case <-ctx.Done():
			return
		}
	}
}
