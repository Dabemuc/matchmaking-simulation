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
	targetPlayers := 10000
	creationRate := 10 * time.Millisecond

	p := pool.New(creationRate, targetPlayers)
	p.Init(ctx)

	compositor := pool.NewCompositor(p)
	// Scenarios defined as executions per second PER PLAYER
	// Matchmaking: Average of 1 minute between attempts
	compositor.AddScenario(pool.MatchmakingScenario{}, 1.0/60.0)
	// Fetch Store: Average of 2 hours between checking the store
	compositor.AddScenario(pool.FetchStoreScenario{}, 1.0/7200.0)
	// Logout: Average session length of 15 minutes
	compositor.AddScenario(pool.LogoutScenario{}, 1.0/900.0)
	compositor.Start(ctx)

	// Wait for a signal to stop
	<-stop
	fmt.Println("\nShutting down...")
	cancel()
	time.Sleep(1 * time.Second)
}
