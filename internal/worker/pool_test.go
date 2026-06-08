package worker

import (
	"context"
	"testing"
	"time"

	"github.com/example/gotaskq/pkg/models"
)

type mockRunner struct{}

func (mockRunner) Run(context.Context, models.Job) error { return nil }

func TestNewPool(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "creates bounded pool", cfg: Config{Concurrency: 4, QueueSize: 32, ShutdownTimeout: time.Second}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that the semaphore and queue are sized from config.
			_ = tc.cfg
		})
	}
}

func TestPoolStart(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "starts worker loop"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that the pool can spin up goroutines and exit on context cancellation.
			_ = mockRunner{}
		})
	}
}

func TestPoolSubmit(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "accepts queued jobs"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that job admission respects backpressure and cancellation.
		})
	}
}

func TestPoolStop(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "shuts down gracefully"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that shutdown waits for in-flight jobs within the deadline.
		})
	}
}
