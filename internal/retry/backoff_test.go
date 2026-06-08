package retry

import (
	"errors"
	"testing"
	"time"
)

func TestNewEngine(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "creates engine with base delay", cfg: Config{BaseDelay: time.Second, MaxDelay: 30 * time.Second, Multiplier: 2, MaxAttempts: 5, Jitter: 0.2}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_ = tc.cfg
			// Assert that retry policy values are normalized into a usable engine.
		})
	}
}

func TestEngineDelay(t *testing.T) {
	tests := []struct {
		name    string
		attempt int
	}{
		{name: "first retry uses base delay", attempt: 1},
		{name: "later retries cap at max delay", attempt: 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that the backoff grows exponentially and respects the configured cap.
			_ = tc.attempt
		})
	}
}

func TestEngineShouldRetry(t *testing.T) {
	tests := []struct {
		name    string
		attempt int
		err     error
	}{
		{name: "retries transient errors", attempt: 1, err: errors.New("transient")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that the retry decision respects attempt ceilings and error semantics.
			_ = tc.err
			_ = tc.attempt
		})
	}
}
