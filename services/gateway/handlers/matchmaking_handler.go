package handlers

import (
	"encoding/json"
	"log"
	"math/rand/v2"
	"net/http"
	"time"
)

func MatchmakingHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	log.Printf("received request on %s from %s", r.URL.Path, r.RemoteAddr)

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Id string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("received matchmaking request for user: %s", req.Id)

	// Simulate matchmaking delay between 1 and 5 seconds
	// This holds the connection open to simulate a long-running process
	delay := time.Duration(1000+rand.IntN(4000)) * time.Millisecond

	select {
	case <-time.After(delay):
		log.Printf("match found for user: %s", req.Id)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "match_found",
			"match_id": "mock_match_id",
		})
	case <-r.Context().Done():
		log.Printf("matchmaking cancelled for user: %s", req.Id)
	}
}
