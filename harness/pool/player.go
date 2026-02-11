package pool

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync/atomic"
	"time"
)

type MatchInfo struct {
	MatchID   string `json:"match_id"`
	GameID    string `json:"game_id"`
	ServerURL string `json:"server_url"`
}

type Player struct {
	id        int
	scenario  chan Scenario
	playerCnt *int64
	cancel    context.CancelFunc
	matchInfo *MatchInfo
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

func (p *Player) run(ctx context.Context, idle chan *Player, emitter ScenarioEmitter) {
	// ---- PLAYER SHUTDOWN ----
	defer func() {
		atomic.AddInt64(p.playerCnt, -1)
	}()

	// Every player must login before becoming idle
	login := LoginScenario{}
	loginStart := time.Now()
	if err := login.Run(ctx, p, emitter); err != nil {
		fmt.Printf("[player %d] login failed: %v\n", p.id, err)
		playerLoginTotal.WithLabelValues("failure").Inc()
		return
	}
	playerLoginDuration.Observe(time.Since(loginStart).Seconds())
	playerLoginTotal.WithLabelValues("success").Inc()

	var pendingScenarios []Scenario

	// ---- NORMAL PLAYER LOOP ----
	for {
		// Priority check: exit immediately if context is already done
		select {
		case <-ctx.Done():
			return
		default:
		}

		var s Scenario

		if len(pendingScenarios) > 0 {
			// Take the next pending scenario
			s = pendingScenarios[0]
			pendingScenarios = pendingScenarios[1:]
		} else {
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
			case assigned, ok := <-p.scenario:
				if !ok {
					return
				}
				s = assigned
			case <-ctx.Done():
				contextCancellationsTotal.WithLabelValues("player_scenario_wait").Inc()
				return
			}
		}

		// Execute Scenario
		scenariosInFlight.WithLabelValues(s.Name()).Inc()
		scenarioStartedTotal.WithLabelValues(s.Name()).Inc()

		scenarioStart := time.Now()
		err := s.Run(ctx, p, emitter)
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
			newScenarios := p.getFollowUpScenarios(s)
			if len(newScenarios) > 0 {
				pendingScenarios = append(newScenarios, pendingScenarios...)
			}
		}
		scenariosInFlight.WithLabelValues(s.Name()).Dec()
	}
}

func (p *Player) getFollowUpScenarios(s Scenario) []Scenario {
	followUpScenarios := s.GetFollowUpScenarios()
	if followUpScenarios == nil {
		return nil
	}

	var result []Scenario
	for _, followUp := range followUpScenarios {
		if rand.Float64() < followUp.Chance {
			result = append(result, followUp.Scenario)
		}
	}
	return result
}
