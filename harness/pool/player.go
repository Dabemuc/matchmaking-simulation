package pool

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

type Player struct {
	id        int
	scenario  chan Scenario
	playerCnt *int64
	cancel    context.CancelFunc // Add cancel function for player's context
}

func newPlayer(id int, playerCnt *int64, cancel context.CancelFunc) *Player {
	playersTotal.Inc()
	playersActive.Inc()
	return &Player{
		id:        id,
		scenario:  make(chan Scenario),
		playerCnt: playerCnt,
		cancel:    cancel, // Assign the cancel function
	}
}

func (p *Player) run(ctx context.Context, idle chan *Player) {
	// ---- PLAYER INITIALIZATION ----
	defer func() {
		atomic.AddInt64(p.playerCnt, -1)
		playersActive.Dec()
	}()
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
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					scenarioCompletedTotal.WithLabelValues(s.Name(), "cancelled").Inc()
				} else {
					scenarioCompletedTotal.WithLabelValues(s.Name(), "failure").Inc()
				}
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
