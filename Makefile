.PHONY: test test-race vet build tidy docker-up docker-down

# Run all unit tests (skips integration tests that require docker-up)
test:
	go test ./...

# Run tests with the race detector enabled (finds concurrency bugs)
test-race:
	go test -race ./...

# Run the Go static analyzer
vet:
	go vet ./...

# Compile the server binary
build:
	go build ./cmd/server

# Download dependencies and regenerate go.sum
tidy:
	go mod tidy

# Start the full local stack (Kafka, Postgres, Redis, Prometheus, Grafana)
docker-up:
	docker compose up -d

# Stop and remove the local stack containers
docker-down:
	docker compose down
