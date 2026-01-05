package pool

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"time"
)

var ErrNoPlayerAvailable = errors.New("no player available")

type Pool struct {
	idle        chan *Player
	rate        time.Duration // limit how fast we create players
	targetCount int           // goal number of players
	nextID      int
	playerCnt   int64
}

func New(creationRate time.Duration, targetCount int) *Pool {
	// Idle capacity matches target count to ensure we can hold everyone if load drops
	poolIdleCapacity.Set(float64(targetCount))
	return &Pool{
		idle:        make(chan *Player, targetCount),
		rate:        creationRate,
		targetCount: targetCount,
	}
}

func (p *Pool) monitor(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			poolIdleQueueDepth.Set(float64(len(p.idle)))
			goroutines.Set(float64(runtime.NumGoroutine()))
			// Explicitly export player count here to ensure gauge is accurate
			playersActive.Set(float64(p.PlayerCount()))
		case <-ctx.Done():
			return
		}
	}
}

func (p *Pool) Init(ctx context.Context) {
	// Monitor metrics
	go p.monitor(ctx)

	// Player Creation Loop
	go func() {
		ticker := time.NewTicker(p.rate)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				tickStart := time.Now()
				currentCount := int(atomic.LoadInt64(&p.playerCnt))

				// TARGET DRIVEN LOGIC:
				// If we have fewer players than target, create one.
				// This handles startup AND recovery from logouts.
				if currentCount < p.targetCount {
					playerCtx, playerCancel := context.WithCancel(ctx)
					playerID := p.nextID
					player := newPlayer(playerID, &p.playerCnt, playerCancel)
					p.nextID++

					atomic.AddInt64(&p.playerCnt, 1)

					go player.run(playerCtx, p.idle, p)
				}
				tickDuration.WithLabelValues("pool_creation").Observe(time.Since(tickStart).Seconds())

			case <-ctx.Done():
				contextCancellationsTotal.WithLabelValues("shutdown").Inc()
				return
			}
		}
	}()
}

func (p *Pool) ExecuteScenario(s Scenario) {
	p.doExecuteScenario(context.Background(), s)
}

func (p *Pool) DoExecuteScenario(ctx context.Context, s Scenario) error {
	return p.doExecuteScenario(ctx, s)
}

func (p *Pool) doExecuteScenario(ctx context.Context, s Scenario) error {
	waitStart := time.Now()

	// Non-blocking attempt first for speed
	select {
	case player := <-p.idle:
		return p.dispatch(ctx, player, s, waitStart)
	default:
		// Fallthrough to wait logic
	}

	// Wait with context
	select {
	case player := <-p.idle:
		return p.dispatch(ctx, player, s, waitStart)
	case <-ctx.Done():
		contextCancellationsTotal.WithLabelValues("pool").Inc()
		return ctx.Err()
	default:
		// In a load test, if no player is idle immediately or very quickly,
		// we often want to fail fast to record "starvation" rather than blocking forever.
		return ErrNoPlayerAvailable
	}
}

func (p *Pool) dispatch(ctx context.Context, player *Player, s Scenario, waitStart time.Time) error {
	poolExecuteWaitDuration.Observe(time.Since(waitStart).Seconds())
	select {
	case player.scenario <- s:
		return nil
	case <-ctx.Done():
		contextCancellationsTotal.WithLabelValues("pool").Inc()
		return ctx.Err()
	}
}

func (p *Pool) PlayerCount() int {
	return int(atomic.LoadInt64(&p.playerCnt))
}
