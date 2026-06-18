.PHONY: up down restart logs ps test test-race vet build tidy enqueue status

# Start the full stack (builds the app image, waits for health checks)
up:
	docker compose up -d --build

# Stop the stack and wipe volumes (fresh state on next `make up`)
down:
	docker compose down -v

# Full restart: wipe and rebuild from scratch
restart: down up

# Tail logs for all services; use `make logs s=app` to filter
logs:
	docker compose logs -f $(s)

# Show running container status
ps:
	docker compose ps

# Run all unit tests
test:
	go test ./...

# Run tests with the race detector
test-race:
	go test -race ./...

# Run the Go static analyzer
vet:
	go vet ./...

# Compile the server binary
build:
	go build -o bin/gotaskq ./cmd/server

# Download dependencies and tidy go.sum
tidy:
	go mod tidy

# Enqueue a sample job (stack must be running)
enqueue:
	curl -s -X POST http://localhost:8080/api/jobs \
		-H "Content-Type: application/json" \
		-d '{"task":{"name":"example","payload":"aGVsbG8="}}' | jq .

# Get job status — usage: make status id=<job-id>
status:
	curl -s http://localhost:8080/api/jobs/$(id) | jq .
