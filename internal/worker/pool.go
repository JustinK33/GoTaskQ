package worker

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/example/gotaskq/pkg/models"
)

// JobRunner performs the actual business work for a job.
type JobRunner interface {
	Run(context.Context, models.Job) error
}

// Config controls the goroutine pool and semaphore bounds.
type Config struct {
	Concurrency     int
	QueueSize       int
	ShutdownTimeout time.Duration
}

// Pool owns the in-process worker lifecycle and concurrency control.
type Pool struct {
	Runner    JobRunner
	jobs      chan models.Job
	semaphore chan struct{}
	cfg       Config
	wg        sync.WaitGroup
}

// NewPool constructs a worker pool with the given runner and concurrency policy.
func NewPool(cfg Config, runner JobRunner) *Pool {
	return &Pool{
		Runner:    runner,
		jobs:      make(chan models.Job, cfg.QueueSize),
		semaphore: make(chan struct{}, cfg.Concurrency),
		cfg:       cfg,
	}
}

// Start begins draining the pool's job queue in the background.
func (p *Pool) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case job, ok := <-p.jobs:
				if !ok {
					return
				}
				p.semaphore <- struct{}{}
				p.wg.Add(1)
				go func(j models.Job) {
					defer p.wg.Done()
					defer func() { <-p.semaphore }()
					defer func() {
						if r := recover(); r != nil {
							fmt.Fprintf(os.Stderr, "worker: panic in job %s: %v\n", j.ID, r)
						}
					}()
					_ = p.Runner.Run(ctx, j) // runner is responsible for its own error handling
				}(job)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Submit enqueues a job if context and buffer capacity permit.
// Returns false immediately when the queue is full or the context is done.
func (p *Pool) Submit(ctx context.Context, job models.Job) bool {
	if ctx.Err() != nil {
		return false
	}
	select {
	case p.jobs <- job:
		return true
	case <-ctx.Done():
		return false
	default:
		return false
	}
}

// Stop closes the job queue and waits for in-flight work to drain within the deadline.
func (p *Pool) Stop(ctx context.Context) error {
	close(p.jobs)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
