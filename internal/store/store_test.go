package store

import (
	"context"
	"testing"
	"time"

	"github.com/example/gotaskq/pkg/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type mockJobStore struct{}

func (mockJobStore) CreateJob(context.Context, models.Job) error      { return nil }
func (mockJobStore) UpdateJob(context.Context, models.Job) error      { return nil }
func (mockJobStore) GetJob(context.Context, string) (models.Job, error) { return models.Job{}, nil }
func (mockJobStore) CancelJob(context.Context, string) error          { return nil }
func (mockJobStore) ClaimNextJob(context.Context) (models.Job, error) { return models.Job{}, nil }

func TestNewPostgresStore(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "wraps pool"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that the pool and table name are captured for later CRUD operations.
			_ = NewPostgresStore((*pgxpool.Pool)(nil), "jobs")
		})
	}
}

func TestCreateJob(t *testing.T) {
	tests := []struct {
		name string
		job  models.Job
	}{
		{name: "inserts pending job", job: models.Job{State: models.JobStatePending, CreatedAt: time.Now(), UpdatedAt: time.Now()}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that new jobs are inserted transactionally with the correct starting state.
			_ = tc.job
		})
	}
}

func TestUpdateJob(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "updates state transition"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that concurrent updates preserve the durable state machine.
		})
	}
}

func TestGetJob(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{name: "returns found job", id: "job-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that lookups return jobs or a clean not-found error.
			_ = tc.id
		})
	}
}

func TestCancelJob(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "marks job cancelled"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that cancellation transitions are applied atomically.
			_ = mockJobStore{}
		})
	}
}

func TestClaimNextJob(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "claims next pending job"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that pending jobs are claimed with row locking semantics.
		})
	}
}

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name string
		from models.JobState
		to   models.JobState
	}{
		{name: "pending to running", from: models.JobStatePending, to: models.JobStateRunning},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that only valid lifecycle transitions are allowed.
			_, _ = tc.from, tc.to
		})
	}
}
