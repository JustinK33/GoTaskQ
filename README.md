# Conduit

[![ci](https://github.com/JustinK33/Conduit/actions/workflows/ci.yml/badge.svg)](https://github.com/JustinK33/Conduit/actions/workflows/ci.yml) [![cd](https://github.com/JustinK33/Conduit/actions/workflows/cd.yml/badge.svg)](https://github.com/JustinK33/Conduit/actions/workflows/cd.yml)

A reliable job queue and lightweight ELT runtime for background work that needs to survive crashes and retries.

<p align="center">
  <img src="ARCHITECTURE.png" alt="Architecture Diagram" width="800"/>
</p>
<p align="center"><em>Job flow from HTTP/Kafka intake through the worker pool, Postgres state store, and Redis locking, to task execution.</em></p>

## What It Does

Conduit runs your background jobs and data pipelines so they survive crashes, retries, and restarts without you having to babysit them.
It handles webhook calls and SQL ELT pipelines out of the box, with retries, crash recovery, and Prometheus metrics built in.
It's meant for teams that need dependable background work without standing up a full workflow orchestration platform.

## Tech Stack

- Kafka (transport)
- PostgreSQL (`pgx`)
- Redis (Redlock)
- Gin (HTTP)
- Prometheus / Grafana
- zerolog
- Docker Compose / Kubernetes

## Install and Run

```bash
git clone https://github.com/JustinK33/Conduit.git && cd Conduit
cp .env.example .env
make up
```

Enqueue a sample job and check its status:

```bash
make enqueue url=https://example.com/webhook
make status id=<id-from-above>
```
