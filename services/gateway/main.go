package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"gateway/gateway"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of http requests",
		},
		[]string{"path", "status"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
}

func main() {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/login", instrument(gateway.LoginHandler))
	http.HandleFunc("/store/offers", instrument(gateway.StoreOffersHandler))
	http.HandleFunc("/store/purchase", instrument(gateway.StorePurchaseHandler))

	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func instrument(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{
			ResponseWriter: w,
			status:         200,
		}
		f(recorder, r)
		httpRequestsTotal.With(prometheus.Labels{
			"path":   r.URL.Path,
			"status": strconv.Itoa(recorder.status),
		}).Inc()
	}
}
