#!/bin/sh
set -e

echo "Starting Docker daemon..."
dockerd-entrypoint.sh > /var/log/dockerd.log 2>&1 &

# Wait for Docker daemon to be ready
echo "Waiting for Docker daemon to be ready..."
timeout=30
while ! docker info > /dev/null 2>&1; do
    timeout=$(($timeout - 1))
    if [ $timeout -eq 0 ]; then
        echo "Error: Docker daemon failed to start within 30 seconds."
        cat /var/log/dockerd.log
        exit 1
    fi
    sleep 1
done

echo "Docker daemon is ready."

# Prepare build context for game-server image
# The orchestrator container has the game-server binary at /usr/local/bin/game-server-bin
mkdir -p /build/game-server
cp /usr/local/bin/game-server-bin /build/game-server/game-server

# Create the Dockerfile for the game server
cat <<DOCKERFILE > /build/game-server/Dockerfile
FROM alpine:latest
WORKDIR /app
COPY game-server .
RUN chmod +x game-server
ENTRYPOINT ["./game-server"]
DOCKERFILE

# Build the game-server image on the local daemon
echo "Building game-server image locally..."
cd /build/game-server
docker build -t game-server:latest .

echo "Starting game-orchestrator..."
exec /usr/local/bin/game-orchestrator
