-- Jobs table: the durable backing store for Conduit.
-- Column names map directly to the fields in pkg/models.Job and pkg/models.Task.
-- Run this once against your local Postgres before using PostgresStore.

CREATE TABLE IF NOT EXISTS jobs (
    -- Identity
    id                  TEXT        PRIMARY KEY,
    idempotency_key     TEXT,

    -- Task fields (flattened from models.Task)
    task_id             TEXT        NOT NULL,
    task_name           TEXT        NOT NULL,
    task_payload        BYTEA,
    task_retry_count    INT         NOT NULL DEFAULT 0,
    task_max_retries    INT         NOT NULL DEFAULT 3,
    task_timeout_ns     BIGINT      NOT NULL DEFAULT 0,  -- stored as nanoseconds, cast to time.Duration
    task_cron_expr      TEXT,
    task_queue          TEXT        NOT NULL DEFAULT 'default',
    task_metadata       JSONB,

    -- Job state machine
    state               TEXT        NOT NULL DEFAULT 'PENDING',
    attempt             INT         NOT NULL DEFAULT 0,
    last_error          TEXT,

    -- Timestamps (nullable = not yet reached that lifecycle stage)
    scheduled_at        TIMESTAMPTZ,
    started_at          TIMESTAMPTZ,
    lease_expires_at    TIMESTAMPTZ,
    lease_token         TEXT,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Arbitrary caller metadata
    metadata            JSONB
);

-- Speeds up ClaimNextJob: only scans PENDING rows ordered by scheduled_at.
CREATE INDEX IF NOT EXISTS jobs_pending_idx
    ON jobs (scheduled_at ASC)
    WHERE state = 'PENDING';

-- Prevents duplicate job creation when clients retry enqueue calls.
CREATE UNIQUE INDEX IF NOT EXISTS jobs_idempotency_key_idx
    ON jobs (idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- Speeds up stale RUNNING lease recovery.
CREATE INDEX IF NOT EXISTS jobs_running_lease_idx
    ON jobs (lease_expires_at ASC)
    WHERE state = 'RUNNING';

-- Speeds up unfiltered keyset pagination in ListJobs.
CREATE INDEX IF NOT EXISTS jobs_created_idx
    ON jobs (created_at DESC, id DESC);

-- Speeds up state-filtered keyset pagination in ListJobs.
CREATE INDEX IF NOT EXISTS jobs_state_created_idx
    ON jobs (state, created_at DESC, id DESC);
