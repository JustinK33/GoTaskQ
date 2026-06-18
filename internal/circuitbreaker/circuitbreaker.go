package circuitbreaker

import (
	"sync"
	"time"
)

type State string

const (
	StateClosed   State = "closed"
	StateOpen     State = "open"
	StateHalfOpen State = "half-open"
)

type Config struct {
	FailureThreshold int
	SuccessThreshold int
	OpenTimeout      time.Duration
	HalfOpenRequests int
}

type Breaker struct {
	mu        sync.Mutex
	cfg       Config
	state     State
	failures  int
	successes int
	openedAt  time.Time
}

func New(cfg Config) *Breaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = 2
	}
	if cfg.OpenTimeout <= 0 {
		cfg.OpenTimeout = 60 * time.Second
	}
	if cfg.HalfOpenRequests <= 0 {
		cfg.HalfOpenRequests = 1
	}
	return &Breaker{cfg: cfg, state: StateClosed}
}

func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(b.openedAt) >= b.cfg.OpenTimeout {
			b.state = StateHalfOpen
			b.failures = 0
			b.successes = 0
			return true
		}
		return false
	case StateHalfOpen:
		return b.successes < b.cfg.HalfOpenRequests
	}
	return false
}

func (b *Breaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateHalfOpen:
		b.successes++
		if b.successes >= b.cfg.SuccessThreshold {
			b.state = StateClosed
			b.failures = 0
			b.successes = 0
		}
	case StateClosed:
		b.failures = 0
	}
}

func (b *Breaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		b.failures++
		if b.failures >= b.cfg.FailureThreshold {
			b.state = StateOpen
			b.openedAt = time.Now()
		}
	case StateHalfOpen:
		b.state = StateOpen
		b.openedAt = time.Now()
		b.failures = 0
		b.successes = 0
	}
}

func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}
