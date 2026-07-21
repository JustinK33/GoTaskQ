# DataflowQ Product Plan

## Name

Use `DataflowQ` as the product name.
It is clearer than `GoTaskQ` because the project is evolving from a generic Go task queue into a reliable data workflow runtime.
The repository and Go module can be renamed later as a mechanical migration after the product direction is stable.

## Target User

The target user is a backend engineer on a small product team.
They need durable background jobs, webhook delivery, and recurring data transformations, but they do not want to operate a full workflow platform.

## Core Use Case

The core use case is operational analytics materialization.
Applications write raw data into Postgres, enqueue a SQL ELT job, and DataflowQ reliably transforms that raw data into analytics tables.
The first demo pipeline turns `raw.orders` into `analytics.daily_revenue`.

## What Exists Now

DataflowQ already has durable job state in Postgres.
It already has Kafka transport for fast worker dispatch.
It already has Redis distributed locks to reduce duplicate execution across instances.
It already has retry backoff, lease recovery, cancellation, request IDs, readiness checks, and Prometheus metrics.
It now has a built-in `sql.etl` task handler that executes validated `INSERT INTO target SELECT ...` pipelines.

## SQL Optimization Story

The existing store path uses keyset pagination instead of offset pagination for job listing.
The existing claim path uses `FOR UPDATE SKIP LOCKED` so workers can claim jobs concurrently without blocking each other.
The migrations include partial indexes for pending jobs, running lease recovery, idempotency lookups, and list pagination.
Those are useful SQL optimization points for a resume because they connect directly to runtime behavior.

## Next SQL Optimization Features

Add a `sql.explain` task that runs `EXPLAIN (FORMAT JSON)` for a pipeline and stores the plan in job metadata.
Add row-count and runtime metrics for each ELT execution.
Add pipeline watermarks so recurring jobs can process only new data.
Add destination write modes for append, upsert, and replace partition.
Add advisory locking by pipeline name so two runs of the same pipeline cannot overwrite the same target partition.
Add an index recommendation report based on common filters, join keys, and grouping columns in pipeline specs.

## Resume Positioning

Strong project title:

> DataflowQ - Reliable SQL ELT and background workflow runtime in Go.

Strong resume bullet:

> Built DataflowQ, a Go-based workflow runtime that executes SQL ELT pipelines through Kafka-backed durable jobs, Postgres state machines, Redis distributed locks, retry backoff, lease recovery, and Prometheus observability.

Technical talking points:

- Designed Postgres as the source of truth while using Kafka as a fast transport path.
- Used `FOR UPDATE SKIP LOCKED` and partial indexes for concurrent job claiming and lease recovery.
- Added a validated SQL ELT executor that limits extraction queries to read-only `SELECT` and `WITH` statements.
- Implemented idempotent job enqueue so client retries do not create duplicate workflow runs.
- Exposed Prometheus metrics and readiness probes for production operations.
