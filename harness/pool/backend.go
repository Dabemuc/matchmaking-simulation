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

func Matchmaking(id int) (*MatchInfo, error) {
	url := getGatewayURL() + "/matchmaking"

	requestBody, err := json.Marshal(map[string]string{
		"id": strconv.Itoa(id),
	})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("matchmaking failed with status code: %d", resp.StatusCode)
	}

	var info MatchInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return &info, nil
}

func ConnectToGameServer(ctx context.Context, info *MatchInfo) error {
	url := fmt.Sprintf("%s?game_id=%s", info.ServerURL, info.GameID)

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
