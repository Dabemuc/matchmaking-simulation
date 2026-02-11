package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"game-orchestrator/metrics"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

type CreateGameRequest struct {
	GameID string `json:"game_id"`
}

type CreateGameResponse struct {
	GameID    string `json:"game_id"`
	ServerURL string `json:"server_url"`
}

var (
	dockerClient *client.Client
	networkName  string
	imageName    string
)

func main() {
	var err error
	dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Error creating docker client: %v", err)
	}
	defer dockerClient.Close()

	// Environment configuration
	networkName = os.Getenv("DOCKER_NETWORK_NAME")
	if networkName == "" {
		// Default to bridge network for dind
		networkName = "bridge"
	}
	imageName = os.Getenv("GAME_SERVER_IMAGE")
	if imageName == "" {
		imageName = "game-server:latest"
	}

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/create", handleCreateGame)
	http.HandleFunc("/game/", handleGameProxy)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	port := "8080"
	log.Printf("Game Orchestrator listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleCreateGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse optional request body for custom ID
	var req CreateGameRequest
	body, err := io.ReadAll(r.Body)
	if err == nil && len(body) > 0 {
		json.Unmarshal(body, &req)
	}

	gameID := req.GameID
	if gameID == "" {
		gameID = uuid.New().String()
	}

	containerName := fmt.Sprintf("game-%s", gameID)

	// Configure the container
	config := &container.Config{
		Image: imageName,
		Env: []string{
			fmt.Sprintf("GAME_ID=%s", gameID),
			"GAME_DURATION=30s", // Default duration
		},
		ExposedPorts: nat.PortSet{
			"8080/tcp": struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		AutoRemove: true, // Clean up container after it exits
	}

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {},
		},
	}

	ctx := context.Background()

	// Create the container
	resp, err := dockerClient.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		log.Printf("Error creating container: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create game server: %v", err), http.StatusInternalServerError)
		return
	}

	// Start the container
	if err := dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		log.Printf("Error starting container: %v", err)
		// Try to clean up if start fails
		_ = dockerClient.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		http.Error(w, fmt.Sprintf("Failed to start game server: %v", err), http.StatusInternalServerError)
		return
	}

	metrics.OngoingMatches.Inc()
	go func(id string) {
		// Wait for container to exit to decrement metric
		statusCh, errCh := dockerClient.ContainerWait(context.Background(), id, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				log.Printf("Error waiting for container %s: %v", id, err)
			}
		case <-statusCh:
		}
		metrics.OngoingMatches.Dec()
	}(resp.ID)

	log.Printf("Started game server container %s (%s)", containerName, resp.ID)

	// Determine orchestrator hostname for the return URL
	hostname := os.Getenv("ORCHESTRATOR_HOSTNAME")
	if hostname == "" {
		hostname = "game-orchestrator"
	}

	// Construct the response
	// The URL points to the orchestrator's proxy endpoint
	response := CreateGameResponse{
		GameID:    gameID,
		ServerURL: fmt.Sprintf("ws://%s:8080/game/%s/connect", hostname, gameID),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleGameProxy(w http.ResponseWriter, r *http.Request) {
	// Expected path: /game/{game_id}/connect
	parts := strings.Split(r.URL.Path, "/")
	// ["", "game", "{game_id}", "connect"]
	if len(parts) < 4 {
		http.NotFound(w, r)
		return
	}
	gameID := parts[2]

	containerName := fmt.Sprintf("game-%s", gameID)
	info, err := dockerClient.ContainerInspect(context.Background(), containerName)
	if err != nil {
		log.Printf("Error inspecting container %s: %v", containerName, err)
		http.Error(w, "Game server not found", http.StatusNotFound)
		return
	}

	ip := info.NetworkSettings.IPAddress
	if ip == "" {
		for _, net := range info.NetworkSettings.Networks {
			ip = net.IPAddress
			break
		}
	}

	if ip == "" {
		log.Printf("Container %s has no IP", containerName)
		http.Error(w, "Game server has no IP", http.StatusInternalServerError)
		return
	}

	targetHost := fmt.Sprintf("%s:8080", ip)

	// Construct the target URL for the ReverseProxy
	targetURL := &url.URL{
		Scheme: "http",
		Host:   targetHost,
		Path:   "/connect",
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Modify the request before sending it to the target
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetHost // Set Host header to match target
		req.URL.Path = "/connect"
		// Clear RequestURI to allow standard lib to re-generate it
		req.RequestURI = ""
	}

	// Go's ReverseProxy automatically handles WebSocket upgrades
	proxy.ServeHTTP(w, r)
}
