# Conduit Improvement Plan

## Product Direction

Conduit should become a self-hosted background job queue for Go services.
It compares most directly to Celery, Sidekiq, Asynq, Faktory, BullMQ, and lightweight Temporal use cases.
It does not compare directly to Kafka because Kafka is a transport inside this system.

## Phase 1 - Make The Current Architecture True

- Implement a Postgres-backed reconciler that claims due `PENDING` jobs and submits them to workers.
- Keep Kafka as a fast wake-up path, not the source of truth.
- Do not publish future-scheduled jobs to Kafka immediately.
- Make worker backpressure visible by refusing to commit Kafka messages when the worker queue is full.
- Prevent enqueue callers from setting server-owned lifecycle fields.
- Add indexes for list and state-filtered job views.

Status: implemented in this branch.

## Phase 2 - Make Jobs Actually Useful

- Add one supported execution model for user tasks.
- The strongest default option is webhook execution because users can adopt it without recompiling Conduit.
- A Go SDK handler mode is also useful, but it makes Conduit more of a library or framework.
- Container or Kubernetes Job execution is powerful, but it is heavier operationally.

Status: webhook execution is implemented as the first supported task mechanism.
Future decision: decide whether Go SDK handlers or container execution should be the second supported mechanism.

## Phase 3 - Make Reliability Production-Grade

- Replace read-then-write state transitions with atomic conditional updates.
- Add leases or heartbeats so abandoned `RUNNING` jobs can be retried.
- Decide how stale `RUNNING` jobs should behave when task handlers are not idempotent.
- Add an explicit migration command or migration job.
- Add durable outbox publishing or a bounded async producer instead of one goroutine per enqueue.

Status: durable leases with fencing tokens are implemented for `RUNNING` jobs.
The default lease duration is 5 minutes and stale jobs are returned to `PENDING` unless retry attempts are exhausted.
Remaining work: replace the remaining read-then-write transitions with narrower compare-and-swap methods.

## Phase 4 - Make Operations Understandable

- Split commands into clear modes such as `server`, `worker`, and `migrate`.
- Add a complete Docker Compose happy path that proves a job reaches `COMPLETED`.
- Add a deployable Kubernetes recipe with Secret template, migration job, and dependency expectations.
- Add dashboards or at least documented Prometheus panels for queue depth, latency, failures, retries, and worker saturation.

Decision needed: choose whether the first real deployment target is Docker Compose, Kubernetes, or both.

## Phase 5 - Make The API Stable

- Keep lifecycle fields server-owned.
- Add idempotency support explicitly instead of accepting caller-provided job IDs casually.
- Add retry, cancel, and dead-letter requeue endpoints.
- Add lightweight list responses that omit large payloads by default.
- Version the API once the execution model is selected.

Status: explicit `idempotency_key` support is implemented.
Remaining work: add manual retry and dead-letter requeue endpoints.
