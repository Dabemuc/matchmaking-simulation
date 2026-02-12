# Project Overview

This document provides an in-depth architectural overview of the Matchmaking Simulation project. The project is designed to simulate a scalable game matchmaking system, inspired by architectures used in large-scale online games like Valorant.

## 1. General Architecture

### Technology Stack
- **Language:** Go (Golang) is used for all custom services (`harness`, `gateway`, `game-orchestrator`, `matchmaking`). The code relies heavily on goroutines and channels for concurrency.
- **Containerization:** All components are containerized using Docker.
- **Orchestration:** `docker-compose` is used for local deployment and orchestration of the service mesh.
- **Database:** Redis is used as the primary data store for volatile state (player sessions, queues) and message brokerage.
- **Monitoring:** A comprehensive monitoring stack is included:
    - **Prometheus:** Scrapes metrics from the `/metrics` endpoint of every service (standardized on port `9090` internally or exposed ports).
    - **Grafana:** Visualizes metrics via dashboards (port `3000`).

### Deployment & Configuration
The project is defined in `docker-compose.yml`, creating a bridge network named `monitoring` where all services communicate.
- **Resource Management:** Resource limits (CPU/Memory) are strictly defined for each service to simulate realistic constraints.
- **Concurrency:** `GOMAXPROCS` is explicitly configured per service via environment variables to align with the container's CPU limits, preventing Go runtime scheduler thrashing.

---

## 2. Component Specifics

### Harness (Load Generator)
*Directory: `harness/`*

The Harness is a sophisticated load generation tool designed to simulate thousands of concurrent players behaving realistically.

*   **Pool (`pool` package):**
    *   **Purpose:** Manages the lifecycle of thousands of `Player` goroutines.
    *   **Mechanism:** Maintains a `targetCount` (e.g., 10,000) of active players. It handles the creation rate to avoid thundering herds during startup and maintains an `idle` channel acting as a pool of available players.
*   **Compositor:**
    *   **Purpose:** The "brain" of the simulation.
    *   **Mechanism:** It does not simply loop scenarios. Instead, it defines **Scenarios** and assigns them a **Rate** (executions per idle second per player).
    *   **Dispatch:** On every tick, it calculates how many scenarios should run based on the current pool size and rates, then dispatches them to idle players.
*   **Player:**
    *   **Implementation:** Each player is a persistent goroutine.
    *   **Lifecycle:** `Login` -> `Idle Loop` -> `Execute Scenario` -> `Maybe Follow-up` -> `Idle Loop`.
    *   **Context:** Holds session state, including `MatchInfo` once a match is found.
*   **Scenarios:**
    *   `Matchmaking`: Requests a match, polls for status, and waits for a server assignment.
    *   `InGame`: Connects to a simulated game server via WebSocket and holds the connection for a duration.
    *   `FetchStore` / `StorePurchase`: Simulates e-commerce transactions.
    *   `Logout`: Terminates the player routine (simulating session end).

### Gateway (API Gateway)
*Directory: `services/gateway/`*

The Gateway acts as the single entry point for all client traffic, abstracting the internal microservice topology.

*   **Reverse Proxy:**
    *   **Matchmaking:** Forwards HTTP requests to the `matchmaking` service (e.g., `/matchmaking/join`).

*   **Instrumentation:**
    *   Wraps all handlers to record HTTP request counts, status codes, and latencies for Prometheus.

### Matchmaking (Logic Service)
*Directory: `services/matchmaking/`*

The core service responsible for forming matches from the queue.

*   **API:**
    *   `POST /matchmaking/join`: Creates a **Ticket** in Redis and pushes the Ticket ID to a Redis List (`queue:default`).
    *   `GET /matchmaking/status`: Polls the status of a specific ticket.
*   **Worker (`matchmakerWorker`):**
    *   A background goroutine that continually polls Redis.
    *   **Batching:** Uses Lua scripts to pop batches of players (e.g., 10) from the queue atomically.
    *   **Logic:** Currently implements a basic FIFO grouping.
    *   **Provisioning:** Upon forming a group, it calls the `game-orchestrator` to allocate a server.
    *   **State Update:** Creates a `Match` object in Redis and updates all player Tickets with the `matched` status and the Server URL.

### Game Orchestrator (Infrastructure Provisioning)
*Directory: `services/game-orchestrator/`*

Manages the lifecycle of game server instances using Docker-in-Docker (DinD).

*   **Privileged Access:** The container runs with `privileged: true` to access the Docker socket.
*   **Provisioning API (`/create`):**
    *   Receives a request for a new game server.
    *   Uses the Docker Client API to spin up a ephemeral container (e.g., based on `game-server` image or self-reference).
    *   Configures the container with `AutoRemove` and environment variables for the specific match (Game ID).
*   **Proxying (`/game/{id}/connect`):**
    *   Acts as a reverse proxy for the dynamically created containers.
    *   Clients connect to the Orchestrator, which inspects the target container's IP and proxies the WebSocket traffic there.

### Game Server
*Directory: `services/game-server/`*

A lightweight, ephemeral service representing a dedicated game server for a single match.

*   **Lifecycle:** Dynamically provisioned by the Game Orchestrator. It runs for a set duration (e.g., 30s) and then terminates.
*   **Connectivity:** Accepts WebSocket connections at `/connect`.
*   **Logic:** Simulates a game loop by reading client messages and echoing them back to simulate state updates.

### Redis (State & Broker)
*   **Queue:** `queue:default` (List) - Stores Ticket IDs waiting for a match.
*   **Tickets:** `ticket:{id}` (String/JSON) - Stores player status (`searching`, `matched`), creation time, and assigned server.
*   **Matches:** `match:{id}` (String/JSON) - Stores the roster and server details for a formed match.