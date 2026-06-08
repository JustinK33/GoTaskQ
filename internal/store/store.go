package store

import (
	"context"

	"github.com/example/gotaskq/pkg/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// JobStore defines the PostgreSQL state machine contract for jobs.
// The API is intentionally explicit so the queue, API, and worker layers can share a durable source of truth.
type JobStore interface {
	CreateJob(context.Context, models.Job) error
	UpdateJob(context.Context, models.Job) error
	GetJob(context.Context, string) (models.Job, error)
	CancelJob(context.Context, string) error
	ClaimNextJob(context.Context) (models.Job, error)
}

// PostgresStore wraps the pgx pool used to persist job state.
// The table name is configurable so different environments can separate data safely.
type PostgresStore struct {
	Pool      *pgxpool.Pool
	TableName string
}

// NewPostgresStore creates a store wrapper around the supplied pgx pool.
// Inputs and outputs:
// - pool is the configured PostgreSQL connection pool.
// - tableName selects the durable jobs table.
// - returns the store wrapper.
// Key implementation steps:
// 1. Capture the pool reference.
// 2. Record the target table name.
// 3. Return the wrapper for later CRUD operations.
// Gotchas:
// - Queries should always target the same state machine columns.
func NewPostgresStore(pool *pgxpool.Pool, tableName string) (store *PostgresStore) {
	// Implementation intentionally omitted.
	return
}

// CreateJob persists a new job record.
// Inputs and outputs:
// - ctx controls cancellation and deadlines.
// - job is the durable execution record to create.
// - returns an error when the insert fails or the row is invalid.
// Key implementation steps:
// 1. Validate the initial job state.
// 2. Insert the row transactionally.
// 3. Surface the created identifier or conflict error.
// Gotchas:
// - Duplicate IDs should be handled deterministically.
func (s *PostgresStore) CreateJob(ctx context.Context, job models.Job) (err error) {
	// Implementation intentionally omitted.
	return
}

// UpdateJob writes the latest in-memory job snapshot back to PostgreSQL.
// Inputs and outputs:
// - ctx controls cancellation and deadlines.
// - job is the updated durable record.
// - returns an error if the update fails or no rows match.
// Key implementation steps:
// 1. Validate the new state transition.
// 2. Persist the updated columns.
// 3. Ensure the row still exists.
// Gotchas:
// - Lost updates are a concern without row versioning or compare-and-swap semantics.
func (s *PostgresStore) UpdateJob(ctx context.Context, job models.Job) (err error) {
	// Implementation intentionally omitted.
	return
}

// GetJob loads a job record by identifier.
// Inputs and outputs:
// - ctx controls cancellation and deadlines.
// - id is the durable job identifier.
// - returns the matched job and an error if the lookup fails.
// Key implementation steps:
// 1. Query by job ID.
// 2. Map columns into the shared model.
// 3. Return a not-found error when no record exists.
// Gotchas:
// - The state machine should preserve timestamps exactly as stored.
func (s *PostgresStore) GetJob(ctx context.Context, id string) (job models.Job, err error) {
	// Implementation intentionally omitted.
	return
}

// CancelJob marks an existing job as cancelled or dead, depending on the state machine rules.
// Inputs and outputs:
// - ctx controls cancellation and deadlines.
// - id identifies the job to cancel.
// - returns an error when the update cannot be completed.
// Key implementation steps:
// 1. Load the current job state.
// 2. Apply the cancel transition.
// 3. Persist the transition atomically.
// Gotchas:
// - In-flight jobs may require additional coordination with the worker or lock layer.
func (s *PostgresStore) CancelJob(ctx context.Context, id string) (err error) {
	// Implementation intentionally omitted.
	return
}

// ClaimNextJob selects the next runnable job from the queue table.
// Inputs and outputs:
// - ctx controls cancellation and deadlines.
// - returns the next job to execute and an error when selection fails.
// Key implementation steps:
// 1. Select a pending job with row locking.
// 2. Transition it to running.
// 3. Return the claimed job to the worker.
// Gotchas:
// - Concurrent workers must not claim the same row.
func (s *PostgresStore) ClaimNextJob(ctx context.Context) (job models.Job, err error) {
	// Implementation intentionally omitted.
	return
}

// CanTransition reports whether a job can move from one state to another.
// Inputs and outputs:
// - from is the current state.
// - to is the target state.
// - returns true when the transition is allowed.
// Key implementation steps:
// 1. Check terminal states.
// 2. Enforce the allowed lifecycle transitions.
// 3. Return the transition decision.
// Gotchas:
// - Dead-letter flows may permit transitions that normal completion does not.
func CanTransition(from, to models.JobState) (allowed bool) {
	// Implementation intentionally omitted.
	return
}
