FROM golang:1.24-alpine

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copy the code
COPY . .

# Build
RUN go build -o matchmaking-app main.go

# Expose matchmaking api port
EXPOSE 8081

CMD [ "./matchmaking-app" ]
