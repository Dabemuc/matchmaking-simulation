package handlers

import (
	"gateway/metrics"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func GameHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	metrics.OngoingMatches.Inc()
	defer metrics.OngoingMatches.Dec()

	gameID := r.URL.Query().Get("game_id")
	// Simulate game duration determined by server
	gameDuration := time.Duration(10+rand.IntN(21)) * time.Second
	endTime := time.Now().Add(gameDuration)

	log.Printf("New game connection for game_id: %s, duration: %v", gameID, gameDuration)

	for {
		if time.Now().After(endTime) {
			log.Printf("Game finished for game_id: %s", gameID)
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Game Over"))
			return
		}

		// Set read deadline to ensure we can check the time even if client is silent
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))

		messageType, p, err := conn.ReadMessage()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			log.Printf("Game connection closed for game_id %s: %v", gameID, err)
			return
		}

		// Echo the message back to simulate state update acknowledgment
		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Println(err)
			return
		}
	}
}
