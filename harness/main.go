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
	targetPlayers := 1000
	creationRate := 2 * time.Millisecond

	p := pool.New(creationRate, targetPlayers)
	p.Init(ctx)

	compositor := pool.NewCompositor(p)
	// Scenarios defined as executions per idle second PER PLAYER
	// Matchmaking: Average of 5 mins between attempts
	compositor.AddScenario(pool.MatchmakingScenario{}, 1.0/(60.0*5))
	// Fetch Store: Average of 45 mins between checking the store
	compositor.AddScenario(pool.FetchStoreScenario{}, 1.0/(60.0*45))
	// Logout: Average session length of 30 mins
	compositor.AddScenario(pool.LogoutScenario{}, 1.0/(60.0*30))
	compositor.Start(ctx)

	// Wait for a signal to stop
	<-stop
	fmt.Println("\nShutting down...")
	cancel()
	time.Sleep(1 * time.Second)
}
