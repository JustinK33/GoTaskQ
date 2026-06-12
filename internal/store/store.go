package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/example/gotaskq/pkg/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// JobStore defines the PostgreSQL state machine contract for jobs.
type JobStore interface {
	CreateJob(context.Context, models.Job) error
	UpdateJob(context.Context, models.Job) error
	GetJob(context.Context, string) (models.Job, error)
	CancelJob(context.Context, string) error
	ClaimNextJob(context.Context) (models.Job, error)
}

// PostgresStore wraps the pgx pool used to persist job state.
type PostgresStore struct {
	Pool      *pgxpool.Pool
	TableName string
}

// NewPostgresStore creates a store wrapper around the supplied pgx pool.
func NewPostgresStore(pool *pgxpool.Pool, tableName string) *PostgresStore {
	return &PostgresStore{Pool: pool, TableName: tableName}
}

// CreateJob inserts a new job record into the store.
func (s *PostgresStore) CreateJob(ctx context.Context, job models.Job) error {
	taskMetaBytes, err := json.Marshal(job.Task.Metadata)
	if err != nil {
		return fmt.Errorf("store: marshal task metadata: %w", err)
	}
	metaBytes, err := json.Marshal(job.Metadata)
	if err != nil {
		return fmt.Errorf("store: marshal metadata: %w", err)
	}

	now := time.Now().UTC()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	job.UpdatedAt = now

	query := fmt.Sprintf(`
		INSERT INTO %s (
			id, task_id, task_name, task_payload,
			task_retry_count, task_max_retries, task_timeout_ns,
			task_cron_expr, task_queue, task_metadata,
			state, attempt, last_error,
			scheduled_at, started_at, completed_at,
			created_at, updated_at, metadata
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
			$11,$12,$13,$14,$15,$16,$17,$18,$19
		)`, s.TableName)

	_, err = s.Pool.Exec(ctx, query,
		job.ID,
		job.Task.ID,
		job.Task.Name,
		job.Task.Payload,
		job.Task.RetryCount,
		job.Task.MaxRetries,
		job.Task.Timeout.Nanoseconds(),
		job.Task.CronExpression,
		job.Task.Queue,
		taskMetaBytes,
		string(job.State),
		job.Attempt,
		job.LastError,
		job.ScheduledAt,
		job.StartedAt,
		job.CompletedAt,
		job.CreatedAt,
		job.UpdatedAt,
		metaBytes,
	)
	return err
}

// GetJob loads a job record by identifier.
func (s *PostgresStore) GetJob(ctx context.Context, id string) (models.Job, error) {
	query := fmt.Sprintf(`
		SELECT
			id, task_id, task_name, task_payload,
			task_retry_count, task_max_retries, task_timeout_ns,
			task_cron_expr, task_queue, task_metadata,
			state, attempt, last_error,
			scheduled_at, started_at, completed_at,
			created_at, updated_at, metadata
		FROM %s WHERE id = $1`, s.TableName)

	row := s.Pool.QueryRow(ctx, query, id)

	var (
		job          models.Job
		taskMetaJSON []byte
		metaJSON     []byte
		timeoutNS    int64
		state        string
	)

	err := row.Scan(
		&job.ID,
		&job.Task.ID,
		&job.Task.Name,
		&job.Task.Payload,
		&job.Task.RetryCount,
		&job.Task.MaxRetries,
		&timeoutNS,
		&job.Task.CronExpression,
		&job.Task.Queue,
		&taskMetaJSON,
		&state,
		&job.Attempt,
		&job.LastError,
		&job.ScheduledAt,
		&job.StartedAt,
		&job.CompletedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
		&metaJSON,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Job{}, ErrJobNotFound
		}
		return models.Job{}, err
	}

	job.State = models.JobState(state)
	job.Task.Timeout = time.Duration(timeoutNS)

	if len(taskMetaJSON) > 0 {
		if err := json.Unmarshal(taskMetaJSON, &job.Task.Metadata); err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Str("job_id", id).Msg("store: failed to unmarshal task metadata")
		}
	}
	if len(metaJSON) > 0 {
		if err := json.Unmarshal(metaJSON, &job.Metadata); err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Str("job_id", id).Msg("store: failed to unmarshal job metadata")
		}
	}

	return job, nil
}

// UpdateJob validates the state transition and persists the updated job snapshot.
func (s *PostgresStore) UpdateJob(ctx context.Context, job models.Job) error {
	existing, err := s.GetJob(ctx, job.ID)
	if err != nil {
		return err
	}
	if !CanTransition(existing.State, job.State) {
		return ErrInvalidTransition
	}

	metaBytes, _ := json.Marshal(job.Metadata)
	taskMetaBytes, _ := json.Marshal(job.Task.Metadata)
	job.UpdatedAt = time.Now().UTC()

	query := fmt.Sprintf(`
		UPDATE %s SET
			state            = $1,
			attempt          = $2,
			last_error       = $3,
			scheduled_at     = $4,
			started_at       = $5,
			completed_at     = $6,
			updated_at       = $7,
			metadata         = $8,
			task_metadata    = $9,
			task_retry_count = $10
		WHERE id = $11`, s.TableName)

	tag, err := s.Pool.Exec(ctx, query,
		string(job.State),
		job.Attempt,
		job.LastError,
		job.ScheduledAt,
		job.StartedAt,
		job.CompletedAt,
		job.UpdatedAt,
		metaBytes,
		taskMetaBytes,
		job.Task.RetryCount,
		job.ID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrJobNotFound
	}
	return nil
}

// CancelJob marks a job as dead, respecting the state machine rules.
func (s *PostgresStore) CancelJob(ctx context.Context, id string) error {
	job, err := s.GetJob(ctx, id)
	if err != nil {
		return err
	}
	if !CanTransition(job.State, models.JobStateDead) {
		return ErrInvalidTransition
	}

	query := fmt.Sprintf(
		`UPDATE %s SET state = $1, updated_at = $2 WHERE id = $3`,
		s.TableName,
	)
	_, err = s.Pool.Exec(ctx, query, string(models.JobStateDead), time.Now().UTC(), id)
	return err
}

// ClaimNextJob atomically selects the next PENDING job and marks it RUNNING.
func (s *PostgresStore) ClaimNextJob(ctx context.Context) (models.Job, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return models.Job{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	selectQ := fmt.Sprintf(`
		SELECT id FROM %s
		WHERE state = 'PENDING'
		ORDER BY scheduled_at ASC NULLS LAST
		LIMIT 1
		FOR UPDATE SKIP LOCKED`, s.TableName)

	var id string
	if err := tx.QueryRow(ctx, selectQ).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Job{}, ErrJobNotFound
		}
		return models.Job{}, err
	}

	now := time.Now().UTC()
	updateQ := fmt.Sprintf(
		`UPDATE %s SET state = 'RUNNING', started_at = $1, updated_at = $2 WHERE id = $3`,
		s.TableName,
	)
	if _, err := tx.Exec(ctx, updateQ, now, now, id); err != nil {
		return models.Job{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Job{}, err
	}

	return s.GetJob(ctx, id)
}

// CanTransition reports whether a job may move from one state to another.
//
// Valid transitions:
//
//	PENDING  → RUNNING
//	RUNNING  → COMPLETED | FAILED | DEAD
//	FAILED   → PENDING  (retry re-enqueue)
//	Any      → DEAD     (dead-letter escalation)
func CanTransition(from, to models.JobState) bool {
	if from == models.JobStateCompleted || from == models.JobStateDead {
		return false
	}
	switch from {
	case models.JobStatePending:
		return to == models.JobStateRunning || to == models.JobStateDead
	case models.JobStateRunning:
		return to == models.JobStateCompleted || to == models.JobStateFailed || to == models.JobStateDead
	case models.JobStateFailed:
		return to == models.JobStatePending || to == models.JobStateDead
	default:
		return false
	}
}
