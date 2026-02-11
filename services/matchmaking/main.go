package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	rdb             *redis.Client
	orchestratorURL string
)

const (
	ticketTTL = 10 * time.Minute
	queueKey  = "queue:default"
)

// Data Structures

type JoinRequest struct {
	PlayerID string `json:"id"`
}

type JoinResponse struct {
	TicketID string `json:"ticketId"`
	Status   string `json:"status"`
}

type ServerInfo struct {
	IP     string `json:"ip,omitempty"`
	Port   int    `json:"port,omitempty"`
	URL    string `json:"url,omitempty"`
	GameID string `json:"gameId,omitempty"`
}

type Ticket struct {
	PlayerID  string     `json:"playerId"`
	Status    string     `json:"status"` // "searching", "matched", "cancelled"
	CreatedAt time.Time  `json:"createdAt"`
	MatchID   string     `json:"matchId,omitempty"`
	Server    ServerInfo `json:"server,omitempty"`
}

type Match struct {
	MatchID   string     `json:"matchId"`
	Players   []string   `json:"players"`
	Server    ServerInfo `json:"server"`
	CreatedAt time.Time  `json:"createdAt"`
}

// Orchestrator Types
type CreateGameResponse struct {
	GameID    string `json:"game_id"`
	ServerURL string `json:"server_url"`
}

func main() {
	// Configuration
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}
	orchestratorURL = os.Getenv("ORCHESTRATOR_URL")
	if orchestratorURL == "" {
		orchestratorURL = "http://game-orchestrator:8080"
	}

	// Initialize Redis
	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Start Background Worker
	go matchmakerWorker()

	// Setup Routes
	http.HandleFunc("/matchmaking/join", handleJoin)
	http.HandleFunc("/matchmaking/status", handleStatus)
	http.HandleFunc("/matchmaking/cancel", handleCancel) // Basic robustness

	port := "8081"
	log.Printf("Matchmaking service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

// Handlers

func handleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req JoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	if req.PlayerID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	ticketID := uuid.New().String()
	ticket := Ticket{
		PlayerID:  req.PlayerID,
		Status:    "searching",
		CreatedAt: time.Now(),
	}

	ticketJSON, err := json.Marshal(ticket)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()

	// Store ticket and add to queue
	// Using a pipeline to ensure efficiency, though strict atomicity between Key Set and List Push isn't critical here
	// as long as both happen.
	pipe := rdb.Pipeline()
	pipe.Set(ctx, "ticket:"+ticketID, ticketJSON, ticketTTL)
	pipe.RPush(ctx, queueKey, ticketID)
	_, err = pipe.Exec(ctx)

	if err != nil {
		log.Printf("Redis error: %v", err)
		http.Error(w, "Failed to queue", http.StatusInternalServerError)
		return
	}

	resp := JoinResponse{
		TicketID: ticketID,
		Status:   "searching",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ticketID := r.URL.Query().Get("ticketId")
	if ticketID == "" {
		http.Error(w, "ticketId required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	val, err := rdb.Get(ctx, "ticket:"+ticketID).Result()
	if err == redis.Nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Redis error: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	var ticket Ticket
	if err := json.Unmarshal([]byte(val), &ticket); err != nil {
		http.Error(w, "Data corruption", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status": ticket.Status,
	}

	if ticket.Status == "matched" {
		response["matchId"] = ticket.MatchID
		response["server"] = ticket.Server
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Assuming ticketId is passed in query for DELETE /matchmaking/cancel?ticketId=...
	// Or could be path param if using a router. Standard net/http uses query usually for simple setup.
	ticketID := r.URL.Query().Get("ticketId")
	// If path param logic was requested: DELETE /matchmaking/ticket/{ticketId}
	// Since I'm using standard mux, parsing path is harder. I'll support query param or simple suffix check.
	if ticketID == "" {
		// try parsing suffix
		prefix := "/matchmaking/ticket/"
		if len(r.URL.Path) > len(prefix) && r.URL.Path[:len(prefix)] == prefix {
			ticketID = r.URL.Path[len(prefix):]
		}
	}

	if ticketID == "" {
		http.Error(w, "ticketId required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Update ticket status to cancelled
	// First get current ticket to check status
	val, err := rdb.Get(ctx, "ticket:"+ticketID).Result()
	if err == redis.Nil {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	var ticket Ticket
	json.Unmarshal([]byte(val), &ticket)

	if ticket.Status == "searching" {
		ticket.Status = "cancelled"
		updatedJSON, _ := json.Marshal(ticket)

		// Remove from queue
		rdb.LRem(ctx, queueKey, 0, ticketID)
		rdb.Set(ctx, "ticket:"+ticketID, updatedJSON, ticketTTL)
	}

	w.WriteHeader(http.StatusOK)
}

// Worker

func matchmakerWorker() {
	log.Println("Matchmaking worker started")
	ctx := context.Background()

	// Lua script to atomically pop 10 items if length >= 10
	popScript := redis.NewScript(`
		if redis.call("LLEN", KEYS[1]) >= 10 then
			return redis.call("LPOP", KEYS[1], 10)
		else
			return nil
		end
	`)

	for {
		// Run Lua script
		result, err := popScript.Run(ctx, rdb, []string{queueKey}).Result()
		if err != nil && err != redis.Nil {
			log.Printf("Worker redis error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if err == redis.Nil || result == nil {
			// Not enough players
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Parse result
		vals, ok := result.([]interface{})
		if !ok {
			log.Printf("Worker unexpected result type")
			continue
		}

		ticketIDs := make([]string, 0, len(vals))
		for _, v := range vals {
			if s, ok := v.(string); ok {
				ticketIDs = append(ticketIDs, s)
			}
		}

		if len(ticketIDs) > 0 {
			log.Printf("Found %d players, creating match...", len(ticketIDs))
			if err := createMatch(ctx, ticketIDs); err != nil {
				log.Printf("Failed to create match: %v", err)
				// Robustness: Could push tickets back to queue here
				// For now, logging error.
			}
		}
	}
}

func createMatch(ctx context.Context, ticketIDs []string) error {
	matchID := uuid.New().String()

	// Call Orchestrator to allocate server
	serverInfo, err := allocateServer()
	if err != nil {
		return fmt.Errorf("allocating server: %w", err)
	}

	// Fetch player IDs from tickets to store in match object
	var playerIDs []string
	for _, tid := range ticketIDs {
		val, err := rdb.Get(ctx, "ticket:"+tid).Result()
		if err == nil {
			var t Ticket
			if json.Unmarshal([]byte(val), &t) == nil {
				playerIDs = append(playerIDs, t.PlayerID)
			}
		}
	}

	match := Match{
		MatchID:   matchID,
		Players:   playerIDs,
		Server:    serverInfo,
		CreatedAt: time.Now(),
	}

	matchJSON, err := json.Marshal(match)
	if err != nil {
		return err
	}

	// Store match
	if err := rdb.Set(ctx, "match:"+matchID, matchJSON, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("saving match: %w", err)
	}

	// Update tickets
	for _, tid := range ticketIDs {
		// Read-Modify-Write for ticket
		// Ideally use optimistic locking (WATCH) or Lua, but simple GET/SET is acceptable for this scope
		// assuming single worker processing this specific ticket or acceptable race with cancel.

		val, err := rdb.Get(ctx, "ticket:"+tid).Result()
		if err != nil {
			log.Printf("Failed to get ticket %s to update: %v", tid, err)
			continue
		}

		var t Ticket
		if err := json.Unmarshal([]byte(val), &t); err != nil {
			continue
		}

		t.Status = "matched"
		t.MatchID = matchID
		t.Server = serverInfo

		updatedJSON, _ := json.Marshal(t)
		rdb.Set(ctx, "ticket:"+tid, updatedJSON, ticketTTL)
	}

	log.Printf("Match %s created for tickets: %v", matchID, ticketIDs)
	return nil
}

func allocateServer() (ServerInfo, error) {
	// Request to orchestrator
	reqBody, _ := json.Marshal(map[string]string{
		"game_id": uuid.New().String(),
	})

	resp, err := http.Post(orchestratorURL+"/create", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return ServerInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ServerInfo{}, fmt.Errorf("orchestrator returned status %d", resp.StatusCode)
	}

	var gameResp CreateGameResponse
	if err := json.NewDecoder(resp.Body).Decode(&gameResp); err != nil {
		return ServerInfo{}, err
	}

	// The orchestrator returns a URL.
	// We pass this URL to the client.
	return ServerInfo{
		URL:    gameResp.ServerURL,
		GameID: gameResp.GameID,
		// If we could parse IP/Port, we would.
		// orchestratorURL usually looks like ws://hostname:8080/game/...
	}, nil
}
