package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	http.HandleFunc("/login",instrument(loginHandler))

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


func loginHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("received request on %s from %s", r.URL.Path, r.RemoteAddr)

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("received login request for user: %s", req.Username)

	w.WriteHeader(http.StatusOK)
}
