package pool

import (
	"context"
	"fmt"
	"time"
)

type Pool struct {
	idle      chan *Player
	rate      time.Duration
	nextID    int
	playerCnt int
}

func New(rate time.Duration, idleCapacity int) *Pool {
	return &Pool{
		idle: make(chan *Player, idleCapacity),
		rate: rate,
	}
}

func (p *Pool) Init(ctx context.Context) {
	ticker := time.NewTicker(p.rate)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				player := newPlayer(p.nextID)
				p.nextID++
				p.playerCnt++

				activePlayers.Set(float64(p.playerCnt))

				go player.run(ctx, p.idle)

			case <-ctx.Done():
				return
			}
		}
	}()
}

func (p *Pool) ExecuteScenario(ctx context.Context, s Scenario) error {
	select {
	case player := <-p.idle:
		select {
		case player.scenario <- s:
			scenarioExecutions.WithLabelValues(fmt.Sprintf("%T", s)).Inc()
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Pool) PlayerCount() int {
	return p.playerCnt
}
