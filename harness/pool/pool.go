package pool

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var ErrNoPlayerAvailable = errors.New("no player available")

type Pool struct {
	idle      chan *Player
	rate      time.Duration
	nextID    int
	playerCnt int64
	// A map to keep track of active players and their cancel functions.
	// This allows the LogoutScenario to trigger the cancellation of a specific player.
	activePlayers sync.Map // map[int]context.CancelFunc
}

func New(rate time.Duration, idleCapacity int) *Pool {
	poolIdleCapacity.Set(float64(idleCapacity))
	return &Pool{
		idle: make(chan *Player, idleCapacity),
		rate: rate,
		activePlayers: sync.Map{},
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

				// Saturation Logic: Only create a new player if the idle channel is not full.
				if len(p.idle) == cap(p.idle) {
					// fmt.Printf("Pool saturated, skipping player creation. Idle: %d/%d\n", len(p.idle), cap(p.idle))
					continue
				}

				playerCtx, playerCancel := context.WithCancel(ctx)
				playerID := p.nextID
				player := newPlayer(playerID, &p.playerCnt, playerCancel)
				p.nextID++
				atomic.AddInt64(&p.playerCnt, 1)
				p.activePlayers.Store(playerID, playerCancel) // Store the cancel function

				go func() {
					player.run(playerCtx, p.idle)
					// When player.run exits, remove its cancel function from the map
					p.activePlayers.Delete(playerID)
				}()
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
	return int(atomic.LoadInt64(&p.playerCnt))
}
