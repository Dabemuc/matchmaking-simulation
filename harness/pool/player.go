package pool

import (
	"context"
	"fmt"
	"time"
)

type Player struct {
	id       int
	scenario chan Scenario
}

func newPlayer(id int) *Player {
	playersTotal.Inc()
	playersActive.Inc()
	return &Player{
		id:       id,
		scenario: make(chan Scenario),
	}
}

func (p *Player) run(ctx context.Context, idle chan *Player) {
	// ---- PLAYER INITIALIZATION ----
	defer playersActive.Dec()
	fmt.Printf("[player %d] starting up\n", p.id)

	// Every player must login before becoming idle
	login := LoginScenario{}
	loginStart := time.Now()
	if err := login.Run(ctx, p); err != nil {
		fmt.Printf("[player %d] login failed: %v\n", p.id, err)
		playerLoginTotal.WithLabelValues("failure").Inc()
		return
	}
	playerLoginDuration.Observe(time.Since(loginStart).Seconds())
	playerLoginTotal.WithLabelValues("success").Inc()

	fmt.Printf("[player %d] login successful, entering idle pool\n", p.id)

	// ---- NORMAL PLAYER LOOP ----
	for {
		// mark player idle
		select {
		case idle <- p:
			playersIdle.Inc()
		case <-ctx.Done():
			contextCancellationsTotal.WithLabelValues("player").Inc()
			return
		}

		// wait for scenario
		select {
		case s := <-p.scenario:
			scenariosInFlight.WithLabelValues(s.Name()).Inc()
			scenarioStartedTotal.WithLabelValues(s.Name()).Inc()
			scenarioStart := time.Now()
			err := s.Run(ctx, p)
			scenarioDuration.WithLabelValues(s.Name()).Observe(time.Since(scenarioStart).Seconds())
			if err != nil {
				scenarioCompletedTotal.WithLabelValues(s.Name(), "failure").Inc()
			} else {
				scenarioCompletedTotal.WithLabelValues(s.Name(), "success").Inc()
			}
			scenariosInFlight.WithLabelValues(s.Name()).Dec()
		case <-ctx.Done():
			contextCancellationsTotal.WithLabelValues("player").Inc()
			return
		}
	}
}
