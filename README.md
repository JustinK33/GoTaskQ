# GoTaskQ

[![ci](https://github.com/JustinK33/GoTaskQ/actions/workflows/ci.yml/badge.svg)](https://github.com/JustinK33/GoTaskQ/actions/workflows/ci.yml) [![cd](https://github.com/JustinK33/GoTaskQ/actions/workflows/cd.yml/badge.svg)](https://github.com/JustinK33/GoTaskQ/actions/workflows/cd.yml)

<p align="center">
  <img src="ARCHITECTURE.png" alt="Architecture Diagram" width="800"/>
</p>

GoTaskQ is a production-grade distributed task queue written in Go. It uses Kafka for durable transport, PostgreSQL as the source-of-truth job state store, Redis (3-node Redlock) for distributed locking, Gin for HTTP, Prometheus + Grafana for observability, and zerolog for structured logging.

## Stack

| Component | Technology |
|-----------|-----------|
| Transport | Kafka (IBM/sarama) |
| State store | PostgreSQL (pgx v5) |
| Distributed lock | Redis Redlock (go-redis v9) |
| HTTP | Gin |
| Metrics | Prometheus + Grafana |
| Logging | zerolog |
| Infra | Docker Compose, Kubernetes (kind) |

## Quick Start

**Prerequisites:** Docker, Docker Compose, `jq` (optional, for pretty output)

```bash
# 1. Clone and enter the repo
git clone https://github.com/JustinK33/GoTaskQ.git && cd GoTaskQ

# 2. Copy the env file (defaults work with Docker Compose out of the box)
cp .env.example .env

# 3. Start the full stack — waits for all health checks before starting the app
make up

# 4. Enqueue a job
make enqueue
# {"id": "5c8058ea-40d6-49f5-9547-cfb37426b368"}

# 5. Check its status
make status id=5c8058ea-40d6-49f5-9547-cfb37426b368
```

## API

All endpoints are under `http://localhost:8080`.

### Enqueue a job

```bash
curl -X POST http://localhost:8080/api/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "task": {
      "name": "send-email",
      "payload": "eyJ0byI6InVzZXJAZXhhbXBsZS5jb20ifQ=="
    }
  }'
```

```json
{"id": "5c8058ea-40d6-49f5-9547-cfb37426b368"}
```

### Get job status

```bash
curl http://localhost:8080/api/jobs/<id>
```

```json
{
  "id": "5c8058ea-40d6-49f5-9547-cfb37426b368",
  "task": {"name": "send-email", ...},
  "state": "PENDING",
  "attempt": 0,
  "scheduled_at": "2026-06-18T21:22:51Z",
  "created_at": "2026-06-18T21:22:51Z",
  "updated_at": "2026-06-18T21:22:51Z"
}
```

### Cancel a job

```bash
curl -X POST http://localhost:8080/api/jobs/<id>/cancel
```

### Health check

```bash
curl http://localhost:8080/health
# {"status": "ok"}
```

### Metrics (Prometheus)

```
http://localhost:8080/metrics
```

## Job States

```
PENDING → RUNNING → COMPLETED
                  → FAILED → PENDING  (retry)
                           → DEAD     (retries exhausted)
         → DEAD  (cancelled)
```

## Observability

| Service | URL | Credentials |
|---------|-----|-------------|
| Prometheus | http://localhost:9090 | — |
| Grafana | http://localhost:3000 | admin / admin |

## Make Targets

```
make up           Start the full Docker stack (builds app image)
make down         Stop and wipe all volumes
make restart      Full teardown and rebuild
make logs         Tail all logs  (make logs s=app for one service)
make ps           Show container status

make test         Run unit tests
make test-race    Run tests with race detector
make vet          Run go vet
make build        Compile binary to bin/gotaskq

make enqueue      POST a sample job to the running stack
make status id=… GET job status by ID
```

## Configuration

All settings are read from environment variables. Copy `.env.example` to `.env` — the defaults work with `docker compose` out of the box.

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_ADDRESS` | `:8080` | Listen address |
| `KAFKA_BROKERS` | `kafka:9092` | Comma-separated broker list |
| `KAFKA_TOPIC` | `gotaskq.jobs` | Job topic |
| `REDIS_ADDRESSES` | `redis:6379,...` | Comma-separated Redis nodes |
| `POSTGRES_DSN` | see `.env.example` | PostgreSQL connection string |
| `WORKER_CONCURRENCY` | `8` | Max parallel job executions |
| `SCHEDULER_ENABLED` | `true` | Enable cron scheduler |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |

> **Running outside Docker?** Change hostnames to `localhost` and set `POSTGRES_DSN` to use port `5433` (the host-mapped port).

## Project Layout

```
cmd/server/         service entrypoint and bootstrap wiring
internal/
  api/              HTTP handlers (enqueue, status, cancel)
  worker/           goroutine pool with bounded concurrency
  scheduler/        cron-style recurring job orchestration
  queue/            Kafka producer and consumer group
  retry/            exponential backoff with jitter
  circuitbreaker/   closed / half-open / open state machine
  lock/             Redis Redlock distributed lock
  store/            PostgreSQL job state machine
  metrics/          Prometheus collector definitions
  logger/           zerolog setup helpers
pkg/models/         shared Job, Task, and Config types
migrations/         SQL schema (auto-applied by Docker Compose)
deploy/             Kubernetes manifests and Prometheus config
loadtest/           k6 load test script
```

## Plugging in Task Handlers

`execute()` in `cmd/server/main.go` is where task dispatch lives. Register handlers by `task.name`:

```go
var handlers = map[string]func(context.Context, models.Job) error{
    "send-email":     sendEmail,
    "resize-image":   resizeImage,
}

func (jw *jobWorker) execute(ctx context.Context, job models.Job) error {
    h, ok := handlers[job.Task.Name]
    if !ok {
        return retry.ErrNoRetry // unknown task — don't retry
    }
    return h(ctx, job)
}
```
