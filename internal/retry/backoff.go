package retry

import "time"

// Config describes the exponential backoff curve used by the worker and queue layers.
// Jitter and cap settings keep retries predictable but not synchronized.
type Config struct {
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
	MaxAttempts int
	Jitter      float64
}

// Engine computes retry timing decisions for job resubmission.
// The implementation will remain deterministic enough for testing while adding jitter in production.
type Engine struct {
	Config Config
}

// NewEngine constructs a retry engine from the provided configuration.
// Inputs and outputs:
// - cfg controls delay growth, jitter, and attempt limits.
// - returns the initialized engine.
// Key implementation steps:
// 1. Normalize invalid or zero-valued timing fields.
// 2. Preserve the configured maximum attempts.
// 3. Return a reusable engine for callers.
// Gotchas:
// - A zero base delay can collapse the backoff curve.
func NewEngine(cfg Config) (engine *Engine) {
	// Implementation intentionally omitted.
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = time.Second
	}

	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 30 * time.Second
	}

	if cfg.Multiplier <= 1 {
		cfg.Multiplier = 2
	}

	if cfg.MaxDelay < cfg.BaseDelay {
		cfg.MaxDelay = cfg.BaseDelay
	}

	if cfg.Jitter < 0 {
		cfg.Jitter = 0
	}

	if cfg.Jitter > 1 {
		cfg.Jitter = 1
	}

	return &Engine{
		Config: cfg,
	}
}

// Delay returns the computed delay for the supplied attempt number.
// Inputs and outputs:
// - attempt is the current retry attempt.
// - returns the delay that should be waited before re-enqueueing.
// Key implementation steps:
// 1. Clamp the attempt to the configured maximum.
// 2. Apply exponential growth from the base delay.
// 3. Cap the delay at the configured maximum.
// 4. Add jitter to avoid retry storms.
// Gotchas:
// - Jitter is a fraction (0.0–1.0) of the computed delay; multiply first, then add, to avoid negative values.
func (e *Engine) Delay(attempt int) (delay time.Duration) {
	// Implementation intentionally omitted.
}

// ShouldRetry determines whether the current error should be retried.
// Inputs and outputs:
// - attempt is the retry counter that would be used next.
// - err is the failure encountered by the worker or queue layer.
// - returns true when the retry engine allows another attempt.
// Key implementation steps:
// 1. Check the attempt ceiling.
// 2. Inspect the error category if policy becomes error-aware.
// 3. Return the decision to the caller.
// Gotchas:
// - Permanent failures should not be retried indefinitely.
func (e *Engine) ShouldRetry(attempt int, err error) (retry bool) {
	// Implementation intentionally omitted.
	return
}
