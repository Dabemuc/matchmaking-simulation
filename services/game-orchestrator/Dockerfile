FROM golang:1.24-alpine AS builder

WORKDIR /src

# Copy service source code
# The build context is the repository root
COPY services/game-server ./services/game-server
COPY services/game-orchestrator ./services/game-orchestrator

# Build Game Server
WORKDIR /src/services/game-server
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/game-server-bin .

# Build Game Orchestrator
WORKDIR /src/services/game-orchestrator
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/game-orchestrator .

# Final Stage: Docker-in-Docker
FROM docker:dind

# Copy compiled binaries
COPY --from=builder /bin/game-orchestrator /usr/local/bin/game-orchestrator
COPY --from=builder /bin/game-server-bin /usr/local/bin/game-server-bin

# Copy startup script
COPY services/game-orchestrator/startup.sh /usr/local/bin/startup.sh
RUN chmod +x /usr/local/bin/startup.sh

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/startup.sh"]
