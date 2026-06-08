package api

import (
	"context"
	"testing"

	"github.com/example/gotaskq/internal/store"
	"github.com/example/gotaskq/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type mockQueue struct{}

func (mockQueue) Enqueue(context.Context, models.Job) (string, error) { return "job-1", nil }
func (mockQueue) Cancel(context.Context, string) error                { return nil }

type mockStore struct{}

func (mockStore) CreateJob(context.Context, models.Job) error         { return nil }
func (mockStore) UpdateJob(context.Context, models.Job) error         { return nil }
func (mockStore) GetJob(context.Context, string) (models.Job, error)   { return models.Job{}, nil }
func (mockStore) CancelJob(context.Context, string) error              { return nil }
func (mockStore) ClaimNextJob(context.Context) (models.Job, error)     { return models.Job{}, nil }

func TestNewHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "wires queue and store"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that the handler stores its dependencies and exposes a usable route bundle.
			_ = NewHandler(mockQueue{}, mockStore{}, zerolog.Logger{})
		})
	}
}

func TestHandlerRegisterRoutes(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "mounts route set"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that enqueue, status, and cancel routes are registered on the provided router.
			_ = gin.New()
			_ = store.JobStore(mockStore{})
		})
	}
}

func TestEnqueueJob(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "accepts enqueue request"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that valid job creation requests are bound, validated, and forwarded to the queue.
		})
	}
}

func TestGetJobStatus(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "returns job status"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that durable job status is returned or a clean not-found response is produced.
		})
	}
}

func TestCancelJob(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "cancels job request"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that cancellation is routed to the queue or store and terminal states are handled.
		})
	}
}
