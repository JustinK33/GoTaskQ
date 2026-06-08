# GoTaskQ

GoTaskQ is a production-oriented distributed task queue scaffold written in Go.
It is organized around Kafka for transport, PostgreSQL for durable job state, Redis for distributed locks, Gin for HTTP, Prometheus for metrics, and zerolog for structured logging.

This repository currently contains the full project structure, interface definitions, and test skeletons for the core subsystems. The implementation bodies are intentionally empty so the codebase can serve as a clean production starting point.

## Stack

- Go
- Kafka via `sarama`
- Redis via `go-redis`
- PostgreSQL via `pgx`
- Gin
- Prometheus via `client_golang`
- zerolog
- Docker and Docker Compose
- GitHub Actions

## Project Layout

- `cmd/server`: service entrypoint and bootstrap wiring
- `internal/api`: enqueue, status, and cancel HTTP handlers
- `internal/worker`: goroutine pool with bounded concurrency
- `internal/scheduler`: cron-style recurring job orchestration
- `internal/queue`: Kafka producer and consumer wrappers
- `internal/retry`: exponential backoff with jitter
- `internal/circuitbreaker`: open / half-open / closed state machine
- `internal/lock`: Redis Redlock coordination
- `internal/store`: PostgreSQL job state machine
- `internal/metrics`: Prometheus collector definitions
- `internal/logger`: zerolog setup helpers
- `pkg/models`: shared job, task, and config types

## Getting Started

### Prerequisites

- Go 1.23 or newer
- Docker and Docker Compose
- Kafka, Redis, and PostgreSQL if running outside Compose

### Local Development

1. Copy `.env.example` to `.env` and update values for your environment.
2. Start the stack:

```bash
docker compose up --build
```

3. Run tests:

```bash
go test ./...
```

4. Run static checks:

```bash
go vet ./...
```

## Configuration

All runtime settings are expected to be sourced from environment variables. Refer to `.env.example` for the initial contract.

Key values include:

- `HTTP_ADDRESS`
- `KAFKA_BROKERS`
- `KAFKA_TOPIC`
- `REDIS_ADDRESSES`
- `POSTGRES_DSN`
- `WORKER_CONCURRENCY`
- `SCHEDULER_ENABLED`
- `METRICS_LISTEN_ADDRESS`

## Docker

The repository includes:

- `Dockerfile` for the service image
- `docker-compose.yml` for local infrastructure
- `.dockerignore` to keep the image build context small

The Compose stack provisions:

- Zookeeper
- Kafka
- Redis
- PostgreSQL
- Prometheus
- Grafana
- the GoTaskQ application container

## CI

GitHub Actions runs:

- `go test ./...`
- `go vet ./...`
- a k6 smoke/load test step

## Status

The codebase is scaffolded for implementation. The current files define the architecture and public contracts, but the runtime behavior still needs to be filled in package by package.
