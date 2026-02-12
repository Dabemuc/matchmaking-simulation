package main

import (
	"flag"
	"log"
	"net/http"
	"os"
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

func main() {
	port := flag.String("port", "8080", "Port to listen on")
	gameID := flag.String("game_id", os.Getenv("GAME_ID"), "Unique Game ID")
	durationStr := flag.String("duration", os.Getenv("GAME_DURATION"), "Game duration (e.g. 30s)")
	flag.Parse()

	if *gameID == "" {
		*gameID = "unknown"
	}

	duration := 30 * time.Second
	if *durationStr != "" {
		if d, err := time.ParseDuration(*durationStr); err == nil {
			duration = d
		} else {
			log.Printf("Invalid duration format %s, defaulting to 30s", *durationStr)
		}
	}

	http.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		handleConnection(w, r, *gameID)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Addr: ":" + *port,
	}

	// Game shutdown timer
	go func() {
		log.Printf("Game %s started, will end in %v", *gameID, duration)
		time.Sleep(duration)
		log.Printf("Game %s time expired, shutting down", *gameID)
		os.Exit(0)
	}()

	log.Printf("Game Server %s listening on :%s", *gameID, *port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func handleConnection(w http.ResponseWriter, r *http.Request, gameID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	log.Printf("Player connected to game %s", gameID)

	// Simple game loop: Echo messages until disconnect or server shutdown
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read error (player disconnect): %v", err)
			return
		}

		// Simulate simple game state update echo
		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Println("Write error:", err)
			return
		}
	}
}
