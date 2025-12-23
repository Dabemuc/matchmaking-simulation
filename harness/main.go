package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"harness/pool"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	// Set up channel for listening to OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start metrics endpoint
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		fmt.Println("Prometheus metrics available at http://localhost:9464/metrics")
		http.ListenAndServe(":9464", nil)
	}()

	// Allow metrics to initialize
	time.Sleep(1 * time.Second)

	// CONFIGURATION:
	// Target 100 active players.
	// Rate limit creation to 10ms (100 players/sec max churn).
	targetPlayers := 100
	creationRate := 10 * time.Millisecond

	p := pool.New(creationRate, targetPlayers)
	p.Init(ctx)

	compositor := pool.NewCompositor(p)
	// Scenarios defined as executions per second PER PLAYER
	compositor.AddScenario(pool.MatchmakingScenario{}, 0.05)   // 1 every 20s per player
	compositor.AddScenario(pool.StorePurchaseScenario{}, 0.02) // 1 every 50s per player
	compositor.AddScenario(pool.LogoutScenario{}, 0.01)        // 1% chance per second to logout (churn)
	compositor.Start(ctx)

	// Wait for a signal to stop
	<-stop
	fmt.Println("\nShutting down...")
	cancel()
	time.Sleep(1 * time.Second)
}
