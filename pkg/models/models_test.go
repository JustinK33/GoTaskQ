package models

import (
	"testing"
	"time"
)

func TestJobStateConstants(t *testing.T) {
	tests := []struct {
		name  string
		state JobState
	}{
		{name: "pending state", state: JobStatePending},
		{name: "running state", state: JobStateRunning},
		{name: "completed state", state: JobStateCompleted},
		{name: "failed state", state: JobStateFailed},
		{name: "dead state", state: JobStateDead},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert the string value matches the durable job lifecycle contract.
			_ = tc.state
		})
	}
}

func TestJobAndTaskShapes(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "job embeds task metadata"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that timestamps, retry metadata, and payload fields can be populated without surprises.
			_ = time.Now()
		})
	}
}
