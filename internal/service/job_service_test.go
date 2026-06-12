package service

import (
	"context"
	"errors"
	"testing"

	"github.com/example/gotaskq/pkg/models"
	"github.com/rs/zerolog"
)

// mockStore satisfies store.JobStore with controllable responses.
type mockStore struct {
	createErr  error
	cancelErr  error
	createdJob models.Job
}

func (m *mockStore) CreateJob(_ context.Context, job models.Job) error {
	m.createdJob = job
	return m.createErr
}
func (m *mockStore) UpdateJob(_ context.Context, _ models.Job) error        { return nil }
func (m *mockStore) GetJob(_ context.Context, _ string) (models.Job, error) { return models.Job{}, nil }
func (m *mockStore) CancelJob(_ context.Context, _ string) error            { return m.cancelErr }
func (m *mockStore) ClaimNextJob(_ context.Context) (models.Job, error)     { return models.Job{}, nil }

// mockPublisher satisfies Publisher with controllable responses.
type mockPublisher struct {
	publishErr error
}

func (m *mockPublisher) Publish(_ context.Context, _ string, _ models.Job) error {
	return m.publishErr
}

func newTestService(s *mockStore, p *mockPublisher) *JobService {
	return NewJobService(p, s, "test-topic", zerolog.Nop())
}

func TestEnqueueSuccess(t *testing.T) {
	ms := &mockStore{}
	svc := newTestService(ms, &mockPublisher{})

	id, err := svc.Enqueue(context.Background(), models.Job{Task: models.Task{Name: "send-email"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty job ID")
	}
	if ms.createdJob.ID != id {
		t.Errorf("stored job ID = %q, want %q", ms.createdJob.ID, id)
	}
	if ms.createdJob.State != models.JobStatePending {
		t.Errorf("stored job state = %q, want %q", ms.createdJob.State, models.JobStatePending)
	}
}

func TestEnqueuePreservesSuppliedID(t *testing.T) {
	ms := &mockStore{}
	svc := newTestService(ms, &mockPublisher{})

	id, err := svc.Enqueue(context.Background(), models.Job{ID: "my-id", Task: models.Task{Name: "process-image"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "my-id" {
		t.Errorf("Enqueue returned ID %q, want %q", id, "my-id")
	}
}

func TestEnqueueStoreError(t *testing.T) {
	storeErr := errors.New("postgres: connection refused")
	ms := &mockStore{createErr: storeErr}
	svc := newTestService(ms, &mockPublisher{})

	_, err := svc.Enqueue(context.Background(), models.Job{Task: models.Task{Name: "failing-task"}})
	if err == nil {
		t.Fatal("expected error when store fails, got nil")
	}
	if !errors.Is(err, storeErr) {
		t.Errorf("error chain does not contain store error; got %v", err)
	}
}

func TestCancelSuccess(t *testing.T) {
	svc := newTestService(&mockStore{}, &mockPublisher{})

	if err := svc.Cancel(context.Background(), "job-123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCancelError(t *testing.T) {
	cancelErr := errors.New("store: job not found")
	ms := &mockStore{cancelErr: cancelErr}
	svc := newTestService(ms, &mockPublisher{})

	err := svc.Cancel(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error when cancel fails, got nil")
	}
	if !errors.Is(err, cancelErr) {
		t.Errorf("error chain does not contain cancel error; got %v", err)
	}
}
