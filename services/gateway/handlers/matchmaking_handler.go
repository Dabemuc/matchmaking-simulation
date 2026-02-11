package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gateway/metrics"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
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

	metrics.PlayersInMatchmaking.Inc()
	defer metrics.PlayersInMatchmaking.Dec()

	// Simulate matchmaking delay between 1 and 5 seconds
	// This holds the connection open to simulate a long-running process
	delay := time.Duration(1000+rand.IntN(4000)) * time.Millisecond

	select {
	case <-time.After(delay):
		log.Printf("match found for user: %s", req.Id)

		orchestratorHost := os.Getenv("ORCHESTRATOR_HOSTNAME")
		if orchestratorHost == "" {
			orchestratorHost = "game-orchestrator"
		}
		orchestratorURL := fmt.Sprintf("http://%s:8080/create", orchestratorHost)

		createReq := map[string]string{
			"game_id": uuid.New().String(),
		}
		reqBody, _ := json.Marshal(createReq)

		resp, err := http.Post(orchestratorURL, "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			log.Printf("failed to request game creation: %v", err)
			http.Error(w, "Matchmaking failed", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("game creation failed with status: %d", resp.StatusCode)
			http.Error(w, "Matchmaking failed", http.StatusInternalServerError)
			return
		}

		var gameInfo struct {
			GameID    string `json:"game_id"`
			ServerURL string `json:"server_url"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&gameInfo); err != nil {
			log.Printf("failed to decode game info: %v", err)
			http.Error(w, "Matchmaking failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":     "match_found",
			"match_id":   uuid.New().String(),
			"game_id":    gameInfo.GameID,
			"server_url": gameInfo.ServerURL,
		})
	case <-r.Context().Done():
		log.Printf("matchmaking cancelled for user: %s", req.Id)
	}
}
