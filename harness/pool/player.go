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
	cancel    context.CancelFunc
}

func newPlayer(id int, playerCnt *int64, cancel context.CancelFunc) *Player {
	playersTotal.Inc()
	return &Player{
		id:        id,
		scenario:  make(chan Scenario),
		playerCnt: playerCnt,
		cancel:    cancel,
	}
}

func (p *Player) run(ctx context.Context, idle chan *Player) {
	// ---- PLAYER SHUTDOWN ----
	defer func() {
		atomic.AddInt64(p.playerCnt, -1)
	}()

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

	// ---- NORMAL PLAYER LOOP ----
	for {
		// Priority check: exit immediately if context is already done
		select {
		case <-ctx.Done():
			return
		default:
		}

		// 1. Enter Idle State
		select {
		case idle <- p:
			// Successfully entered idle queue
		case <-ctx.Done():
			contextCancellationsTotal.WithLabelValues("player_idle_wait").Inc()
			return
		}

		// 2. Wait for Scenario assignment
		select {
		case s, ok := <-p.scenario:
			if !ok {
				return
			}
			scenariosInFlight.WithLabelValues(s.Name()).Inc()
			scenarioStartedTotal.WithLabelValues(s.Name()).Inc()

			scenarioStart := time.Now()
			err := s.Run(ctx, p)
			duration := time.Since(scenarioStart).Seconds()
			scenarioDuration.WithLabelValues(s.Name()).Observe(duration)

			if err != nil {
				if errors.Is(err, context.Canceled) {
					scenarioCompletedTotal.WithLabelValues(s.Name(), "cancelled").Inc()
					scenariosInFlight.WithLabelValues(s.Name()).Dec()
					return // Player context was cancelled (Logout)
				}
				scenarioCompletedTotal.WithLabelValues(s.Name(), "failure").Inc()
			} else {
				scenarioCompletedTotal.WithLabelValues(s.Name(), "success").Inc()
			}
			scenariosInFlight.WithLabelValues(s.Name()).Dec()

		case <-ctx.Done():
			contextCancellationsTotal.WithLabelValues("player_scenario_wait").Inc()
			return
		}
	}
}
