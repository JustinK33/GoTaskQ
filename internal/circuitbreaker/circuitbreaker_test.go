package circuitbreaker

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		wantState State
	}{
		{
			name:      "starts in closed state",
			cfg:       Config{FailureThreshold: 3, SuccessThreshold: 2, OpenTimeout: time.Second, HalfOpenRequests: 1},
			wantState: StateClosed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := New(tc.cfg)
			if b == nil {
				t.Fatal("New returned nil")
			}
			if got := b.State(); got != tc.wantState {
				t.Errorf("State() = %q, want %q", got, tc.wantState)
			}
		})
	}
}

func TestBreakerAllow(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*Breaker)
		wantAllowed bool
	}{
		{
			name:        "allows calls when closed",
			setup:       nil,
			wantAllowed: true,
		},
		{
			name: "blocks calls when open",
			setup: func(b *Breaker) {
				for i := 0; i < b.cfg.FailureThreshold; i++ {
					b.RecordFailure()
				}
			},
			wantAllowed: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := New(Config{FailureThreshold: 3, SuccessThreshold: 2, OpenTimeout: time.Hour, HalfOpenRequests: 1})
			if b == nil {
				t.Fatal("New returned nil")
			}
			if tc.setup != nil {
				tc.setup(b)
			}
			got := b.Allow()
			if got != tc.wantAllowed {
				t.Errorf("Allow() = %v, want %v (state: %s)", got, tc.wantAllowed, b.State())
			}
		})
	}
}

func TestBreakerRecordFailure(t *testing.T) {
	tests := []struct {
		name             string
		failureCount     int
		failureThreshold int
		wantState        State
	}{
		{
			name:             "stays closed below threshold",
			failureCount:     2,
			failureThreshold: 3,
			wantState:        StateClosed,
		},
		{
			name:             "opens at threshold",
			failureCount:     3,
			failureThreshold: 3,
			wantState:        StateOpen,
		},
		{
			name:             "opens beyond threshold",
			failureCount:     5,
			failureThreshold: 3,
			wantState:        StateOpen,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := New(Config{FailureThreshold: tc.failureThreshold, SuccessThreshold: 2, OpenTimeout: time.Hour, HalfOpenRequests: 1})
			for i := 0; i < tc.failureCount; i++ {
				b.RecordFailure()
			}
			if got := b.State(); got != tc.wantState {
				t.Errorf("after %d failures: State() = %q, want %q", tc.failureCount, got, tc.wantState)
			}
		})
	}
}

func TestBreakerRecordSuccess(t *testing.T) {
	t.Run("resets failure pressure in closed state", func(t *testing.T) {
		b := New(Config{FailureThreshold: 3, SuccessThreshold: 2, OpenTimeout: time.Hour, HalfOpenRequests: 1})
		b.RecordFailure()
		b.RecordFailure()
		b.RecordSuccess()
		// After a success, breaker should remain closed and not trip on the next failure
		if got := b.State(); got != StateClosed {
			t.Errorf("State() = %q after success, want %q", got, StateClosed)
		}
	})

	t.Run("recovers to closed after enough successes in half-open", func(t *testing.T) {
		b := New(Config{FailureThreshold: 1, SuccessThreshold: 2, OpenTimeout: time.Nanosecond, HalfOpenRequests: 2})
		b.RecordFailure()
		time.Sleep(2 * time.Nanosecond)
		b.Allow()
		b.RecordSuccess()
		b.RecordSuccess()
		if got := b.State(); got != StateClosed {
			t.Errorf("State() = %q after recovery successes, want %q", got, StateClosed)
		}
	})
}

func TestBreakerState(t *testing.T) {
	t.Run("reports closed on fresh breaker", func(t *testing.T) {
		b := New(Config{FailureThreshold: 3, SuccessThreshold: 2, OpenTimeout: time.Second, HalfOpenRequests: 1})
		if got := b.State(); got != StateClosed {
			t.Errorf("State() = %q, want %q", got, StateClosed)
		}
	})

	t.Run("reports open after tripping", func(t *testing.T) {
		b := New(Config{FailureThreshold: 1, SuccessThreshold: 2, OpenTimeout: time.Hour, HalfOpenRequests: 1})
		b.RecordFailure()
		if got := b.State(); got != StateOpen {
			t.Errorf("State() = %q after trip, want %q", got, StateOpen)
		}
	})
}
