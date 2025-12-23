package pool

import (
	"context"
	"errors"
	"runtime"
	"time"
)

var ErrNoPlayerAvailable = errors.New("no player available")

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

func (p *Pool) monitor(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			poolIdleQueueDepth.Set(float64(len(p.idle)))
			goroutines.Set(float64(runtime.NumGoroutine()))
		case <-ctx.Done():
			return
		}
	}
}
func (p *Pool) Init(ctx context.Context) {
	ticker := time.NewTicker(p.rate)
	go p.monitor(ctx)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				tickStart := time.Now()
				player := newPlayer(p.nextID)
				p.nextID++
				p.playerCnt++
				go player.run(ctx, p.idle)
				tickDuration.WithLabelValues("pool").Observe(time.Since(tickStart).Seconds())

			case <-ctx.Done():
				contextCancellationsTotal.WithLabelValues("shutdown").Inc()
				return
			}
		}
	}()
}

func (p *Pool) ExecuteScenario(ctx context.Context, s Scenario) error {
	waitStart := time.Now()
	select {
	case player := <-p.idle:
		poolExecuteWaitDuration.Observe(time.Since(waitStart).Seconds())
		playersIdle.Dec()
		select {
		case player.scenario <- s:
			return nil
		case <-ctx.Done():
			contextCancellationsTotal.WithLabelValues("pool").Inc()
			return ctx.Err()
		}
	case <-ctx.Done():
		contextCancellationsTotal.WithLabelValues("pool").Inc()
		return ctx.Err()
	default:
		return ErrNoPlayerAvailable
	}
}

func (p *Pool) PlayerCount() int {
	return p.playerCnt
}
