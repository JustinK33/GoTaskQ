package retry

import (
	"errors"
	"testing"
	"time"
)

func TestNewEngine(t *testing.T) {
	tests := []struct {
		name            string
		cfg             Config
		wantMaxAttempts int
	}{
		{
			name:            "stores config values",
			cfg:             Config{BaseDelay: time.Second, MaxDelay: 30 * time.Second, Multiplier: 2, MaxAttempts: 5, Jitter: 0.2},
			wantMaxAttempts: 5,
		},
		{
			name:            "zero max attempts is preserved",
			cfg:             Config{BaseDelay: time.Second, MaxDelay: 30 * time.Second, Multiplier: 2, MaxAttempts: 0},
			wantMaxAttempts: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			engine := NewEngine(tc.cfg)
			if engine == nil {
				t.Fatal("NewEngine returned nil")
			}
			if engine.Config.MaxAttempts != tc.wantMaxAttempts {
				t.Errorf("MaxAttempts = %d, want %d", engine.Config.MaxAttempts, tc.wantMaxAttempts)
			}
		})
	}
}

func TestEngineDelay(t *testing.T) {
	cfg := Config{
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    10 * time.Second,
		Multiplier:  2,
		MaxAttempts: 10,
		Jitter:      0, // no jitter for deterministic tests
	}

	tests := []struct {
		name        string
		attempt     int
		wantAtLeast time.Duration
		wantAtMost  time.Duration
	}{
		{
			name:        "first attempt uses base delay",
			attempt:     1,
			wantAtLeast: 100 * time.Millisecond,
			wantAtMost:  10 * time.Second,
		},
		{
			name:        "later attempts cap at max delay",
			attempt:     100,
			wantAtLeast: time.Duration(0),
			wantAtMost:  10 * time.Second,
		},
		{
			name:        "attempt zero uses base delay",
			attempt:     0,
			wantAtLeast: 100 * time.Millisecond,
			wantAtMost:  10 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			engine := NewEngine(cfg)
			got := engine.Delay(tc.attempt)
			if got < tc.wantAtLeast {
				t.Errorf("Delay(%d) = %v, want >= %v", tc.attempt, got, tc.wantAtLeast)
			}
			if got > tc.wantAtMost {
				t.Errorf("Delay(%d) = %v, want <= %v", tc.attempt, got, tc.wantAtMost)
			}
		})
	}
}

func TestEngineDelayGrowth(t *testing.T) {
	cfg := Config{
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    10 * time.Second,
		Multiplier:  2,
		MaxAttempts: 10,
		Jitter:      0,
	}
	engine := NewEngine(cfg)

	d1 := engine.Delay(1)
	d2 := engine.Delay(2)
	if d2 <= d1 {
		t.Errorf("delay should grow: Delay(1)=%v Delay(2)=%v", d1, d2)
	}
}

func TestEngineShouldRetry(t *testing.T) {
	tests := []struct {
		name        string
		maxAttempts int
		attempt     int
		err         error
		want        bool
	}{
		{
			name:        "retries when under the ceiling",
			maxAttempts: 5,
			attempt:     1,
			err:         errors.New("transient"),
			want:        true,
		},
		{
			name:        "stops at max attempts",
			maxAttempts: 5,
			attempt:     5,
			err:         errors.New("transient"),
			want:        false,
		},
		{
			name:        "stops beyond max attempts",
			maxAttempts: 5,
			attempt:     10,
			err:         errors.New("transient"),
			want:        false,
		},
		{
			name:        "nil error still counts against ceiling",
			maxAttempts: 3,
			attempt:     1,
			err:         nil,
			want:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			engine := NewEngine(Config{MaxAttempts: tc.maxAttempts, BaseDelay: time.Millisecond, MaxDelay: time.Second, Multiplier: 2})
			got := engine.ShouldRetry(tc.attempt, tc.err)
			if got != tc.want {
				t.Errorf("ShouldRetry(%d, %v) = %v, want %v", tc.attempt, tc.err, got, tc.want)
			}
		})
	}
}
