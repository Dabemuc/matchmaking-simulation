package pool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

func getGatewayURL() string {
	hostname := os.Getenv("GATEWAY_HOSTNAME")
	if hostname == "" {
		hostname = "localhost"
	}
	return fmt.Sprintf("http://%s:8080", hostname)
}

func Login(id int, password string) error {
	url := getGatewayURL() + "/login"
	requestBody, err := json.Marshal(map[string]string{
		"id":       strconv.Itoa(id),
		"password": password,
	})
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status code: %d", resp.StatusCode)
	}

	return nil
}

func FetchStore() error {
	url := getGatewayURL() + "/store/offers"

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching store failed with status code: %d", resp.StatusCode)
	}

	return nil
}

func StorePurchase(id int) error {
	url := getGatewayURL() + "/store/purchase"

	requestBody, err := json.Marshal(map[string]string{
		"id":       strconv.Itoa(id),
		"offer_id": strconv.Itoa(int(rand.Uint())),
	})
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("store purchase failed with status code: %d", resp.StatusCode)
	}

	return nil
}

// Internal structs for matchmaking response parsing
type joinResponse struct {
	TicketID string `json:"ticketId"`
	Status   string `json:"status"`
}

type serverInfo struct {
	URL    string `json:"url"`
	GameID string `json:"gameId"`
}

type statusResponse struct {
	Status  string     `json:"status"`
	MatchID string     `json:"matchId"`
	Server  serverInfo `json:"server"`
}

func Matchmaking(id int) (*MatchInfo, error) {
	// 1. Join matchmaking
	joinURL := getGatewayURL() + "/matchmaking/join"
	requestBody, err := json.Marshal(map[string]string{
		"id": strconv.Itoa(id),
	})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(joinURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("matchmaking join failed with status code: %d", resp.StatusCode)
	}

	var joinResp joinResponse
	if err := json.NewDecoder(resp.Body).Decode(&joinResp); err != nil {
		return nil, fmt.Errorf("failed to decode join response: %v", err)
	}

	ticketID := joinResp.TicketID
	if ticketID == "" {
		return nil, fmt.Errorf("received empty ticketId")
	}

	// 2. Poll for status
	statusURL := getGatewayURL() + "/matchmaking/status"
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Timeout after 60 seconds
	timeout := time.After(60 * time.Second)

	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("matchmaking timed out for ticket %s", ticketID)
		case <-ticker.C:
			req, err := http.NewRequest("GET", statusURL, nil)
			if err != nil {
				return nil, err
			}
			q := req.URL.Query()
			q.Add("ticketId", ticketID)
			req.URL.RawQuery = q.Encode()

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				continue
			}

			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				continue
			}

			var statusResp statusResponse
			err = json.NewDecoder(resp.Body).Decode(&statusResp)
			resp.Body.Close()

			if err != nil {
				continue
			}

			if statusResp.Status == "matched" {
				return &MatchInfo{
					MatchID:   statusResp.MatchID,
					GameID:    statusResp.Server.GameID,
					ServerURL: statusResp.Server.URL,
				}, nil
			} else if statusResp.Status == "cancelled" {
				return nil, fmt.Errorf("matchmaking ticket cancelled")
			}
			// if "searching", continue polling
		}
	}
}

func ConnectToGameServer(ctx context.Context, info *MatchInfo) error {
	// The Orchestrator returns the full WebSocket URL now.
	url := info.ServerURL
	if url == "" {
		return fmt.Errorf("server url is empty")
	}

	c, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		return err
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return ctx.Err()
		case <-done:
			return nil
		case <-ticker.C:
			if err := c.WriteMessage(websocket.TextMessage, []byte("game_state_update")); err != nil {
				return err
			}
		}
	}
}
