# GoTaskQ

A production-grade **distributed task queue** written in Go — built from scratch to demonstrate real-world backend engineering across the full distributed-systems stack.

---

## What It Does

GoTaskQ is an HTTP-driven job queue service that accepts work from callers, durably persists it in PostgreSQL, publishes it to a Kafka topic, and executes it through a bounded worker pool — all with automatic retries, distributed locking, circuit-breaking, and Prometheus observability.

```
Client
  │  POST /api/jobs
  ▼
API Handler (Gin)
  │  Enqueue(job)
  ▼
Job Service ──────────────► Kafka topic "jobs"
  │  CreateJob                   │
  ▼                              │ Consume
PostgreSQL                  Worker Pool ──► JobRunner
  │  PENDING → RUNNING               │
  │  RUNNING → COMPLETED/FAILED      │ Retry Engine
  ▼  FAILED  → PENDING / DEAD        ▼
 State Machine               Circuit Breaker
                                    │
 Scheduler ──── cron entries        ▼
 (recurring jobs)           Redis Distributed Lock
                                    │
 Prometheus /metrics         observability
```

---

## Key Features

| Feature | Details |
|---|---|
| **Async job dispatch** | HTTP POST enqueues a job; returns the ID immediately |
| **Kafka transport** | Durable publish with per-message headers for tracing |
| **PostgreSQL state machine** | `PENDING → RUNNING → COMPLETED / FAILED → DEAD` with `FOR UPDATE SKIP LOCKED` claim |
| **Redlock distributed locking** | Multi-node quorum lock for exclusive resource coordination |
| **Exponential backoff** | Configurable base, multiplier, cap, and jitter |
| **Circuit breaker** | Closed / Open / Half-Open state machine protecting downstream calls |
| **Cron scheduler** | 5-field cron expressions with `*/n`, ranges, and lists — zero external dependencies |
| **Bounded worker pool** | Semaphore-controlled concurrency with graceful shutdown |
| **Prometheus metrics** | Counters, gauges, histograms exposed on `/metrics` |
| **Structured logging** | zerolog JSON logs with service/env/component fields |
| **Graceful shutdown** | SIGTERM drains in-flight jobs before exit |
| **Fully tested** | All packages tested with `-race`; integration tests gated behind `make docker-up` |

---

## Architecture

### 12 packages, ~1 600 lines of production Go

```
cmd/server/          ← bootstrap, wiring, graceful shutdown
internal/
  api/               ← Gin HTTP handler (POST /api/jobs, GET /api/jobs/:id, cancel)
  config/            ← env-var config loading + validation
  circuitbreaker/    ← Closed / Open / Half-Open state machine
  lock/              ← Redlock algorithm over multiple Redis nodes
  logger/            ← zerolog setup (JSON + pretty modes)
  metrics/           ← Prometheus counter/gauge/histogram registry
  queue/             ← Sarama Kafka producer + consumer group
  retry/             ← exponential backoff + jitter engine
  scheduler/         ← cron scheduler with built-in 5-field parser
  service/           ← Queue adapter: bridges API ↔ Kafka + Postgres
  store/             ← pgx v5 CRUD + state machine + `FOR UPDATE SKIP LOCKED`
  worker/            ← goroutine pool with semaphore and WaitGroup
pkg/models/          ← shared domain types (Job, Task, JobState, Config)
migrations/          ← SQL schema (001_create_jobs.sql)
deploy/prometheus/   ← prometheus.yml scrape config
```

---

## Technology Stack

| Layer | Technology |
|---|---|
| Language | Go 1.23 |
| HTTP | Gin (`github.com/gin-gonic/gin`) |
| Message queue | Apache Kafka via IBM Sarama (`github.com/IBM/sarama`) |
| Database | PostgreSQL 16 via pgx v5 (`github.com/jackc/pgx/v5`) |
| Caching / Locking | Redis 7 via go-redis v9 (`github.com/redis/go-redis/v9`) |
| Observability | Prometheus (`github.com/prometheus/client_golang`) |
| Logging | zerolog (`github.com/rs/zerolog`) |
| Containerisation | Docker Compose (Kafka, Postgres, 3× Redis, Prometheus, Grafana) |
| Orchestration | Kubernetes — Deployment (3 replicas), Service, HPA (min 3 / max 10) |
| CI/CD | GitHub Actions — vet → unit tests → race detector → Docker build → k6 load test |

---

## API Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/jobs` | Enqueue a new job; returns `{"id": "..."}` |
| `GET` | `/api/jobs/:id` | Fetch job state and timestamps |
| `POST` | `/api/jobs/:id/cancel` | Cancel a job (transitions to `DEAD`) |
| `GET` | `/metrics` | Prometheus scrape endpoint |
| `GET` | `/health` | Liveness probe |

### Example

```bash
# Enqueue
curl -X POST http://localhost:8080/api/jobs \
  -H 'Content-Type: application/json' \
  -d '{"task":{"name":"send-email","queue":"default","max_retries":3}}'
# → {"id":"4a7b1c2d-..."}

# Check status
curl http://localhost:8080/api/jobs/4a7b1c2d-...
# → {"ID":"4a7b1c2d-...","State":"COMPLETED",...}

# Cancel
curl -X POST http://localhost:8080/api/jobs/4a7b1c2d-.../cancel
# → {"status":"cancelled"}
```

---

## Metrics

All metrics are prefixed `gotaskq_server_*` by default (configurable via env vars).

| Metric | Type | Description |
|---|---|---|
| `jobs_enqueued_total` | Counter | Jobs accepted via the API |
| `jobs_started_total` | Counter | Jobs claimed by a worker |
| `jobs_completed_total` | Counter | Jobs finished successfully |
| `jobs_failed_total` | Counter | Jobs that errored |
| `jobs_cancelled_total` | Counter | Jobs cancelled by callers |
| `worker_in_flight` | Gauge | Concurrently executing jobs |
| `job_duration_seconds` | Histogram | End-to-end execution latency |
| `http_requests_total` | CounterVec | Requests by method / path / status |

---

## Running Locally

```bash
# Start all infrastructure (Kafka, Postgres, Redis, Prometheus, Grafana)
make docker-up

# Apply the database migration
psql $POSTGRES_DSN -f migrations/001_create_jobs.sql

# Build and run
make build
./bin/server

# In another terminal, enqueue a test job
curl -X POST http://localhost:8080/api/jobs \
  -H 'Content-Type: application/json' \
  -d '{"task":{"name":"hello-world","queue":"default"}}'

# Metrics
curl http://localhost:8080/metrics | grep gotaskq

# Tear down
make docker-down
```

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `POSTGRES_DSN` | `postgres://postgres:postgres@localhost:5432/gotaskq?sslmode=disable` | PostgreSQL connection string |
| `KAFKA_BROKERS` | `localhost:9092` | Comma-separated broker list |
| `KAFKA_TOPIC` | `jobs` | Topic for job messages |
| `REDIS_ADDRESSES` | `localhost:6379` | Comma-separated Redis addresses |
| `HTTP_ADDRESS` | `:8080` | Server listen address |
| `WORKER_CONCURRENCY` | `10` | Max parallel job executions |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `LOG_PRETTY` | `false` | Human-readable console output |

---

## Running Tests

```bash
# Unit tests (no infrastructure required)
make test

# Unit tests with race detector
make test-race

# Integration tests (requires make docker-up)
go test ./... -tags=integration
```

---

## Project Statistics

| Metric | Value |
|---|---|
| Go packages | 13 |
| Source lines (production) | ~1 600 |
| Test coverage | all packages (`-race` clean) |
| Infrastructure components | Kafka, PostgreSQL, 3× Redis, Prometheus, Grafana |
| Kubernetes manifests | Deployment, Service, ConfigMap, HPA |
| API endpoints | 5 |
| Prometheus metrics | 8 |
| Cron scheduler | built-in (zero external deps) |

---

## Performance (stress-tested locally)

All numbers measured with Apache Bench against a locally running server (Kafka + Postgres + 3× Redis via Docker Compose).

| Metric | Value | Notes |
|---|---|---|
| **Peak throughput** | **11,256 req/s** | 200 concurrent clients, 10,000 requests |
| **Sustained throughput** | **~4,000 req/s** | 100 concurrent clients |
| **Error rate** | **0%** | 0 failures across 15,000+ total requests |
| **API latency p50** | **< 1 ms** (0.82 ms) | Sequential keep-alive measurement |
| **API latency p95** | **22 ms** | Under 200-client concurrent load |
| **API latency p99** | **33 ms** | Under 200-client concurrent load |
| **Job execution p50** | **< 5 ms** | Postgres → Kafka → worker round-trip |
| **Job execution p99** | **< 100 ms** | Prometheus histogram confirmed |
| **Distributed lock nodes** | **3** | Redlock quorum across 3 independent Redis nodes |

**Key tuning that drove the gains:**
- Made Kafka publish non-blocking (goroutine after Postgres write) — removed the Kafka round-trip from the HTTP hot path
- pgx pool: MaxConns 25 → 50, added MaxConnLifetime / MaxConnIdleTime / HealthCheckPeriod
- Kafka producer: snappy compression, 5ms flush frequency, 1MiB flush threshold, 256-deep channel buffer
- Worker pool: Concurrency 10 → 50, QueueSize 100 → 500
