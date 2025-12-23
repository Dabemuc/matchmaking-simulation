package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"harness/pool"
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

	time.Sleep(1 * time.Second)

	p := pool.New(100*time.Millisecond, 1000)
	p.Init(ctx)

	compositor := pool.NewCompositor(p)
	compositor.AddScenario(pool.MatchmakingScenario{}, 0.1)
	compositor.AddScenario(pool.StorePurchaseScenario{}, 0.02)
	compositor.Start(ctx)

	// Wait for a signal to stop
	<-stop
	fmt.Println("\nShutting down...")
	cancel()
	time.Sleep(1 * time.Second)
}
