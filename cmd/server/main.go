package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/example/gotaskq/internal/api"
	"github.com/example/gotaskq/internal/circuitbreaker"
	"github.com/example/gotaskq/internal/config"
	"github.com/example/gotaskq/internal/lock"
	"github.com/example/gotaskq/internal/logger"
	"github.com/example/gotaskq/internal/metrics"
	"github.com/example/gotaskq/internal/queue"
	"github.com/example/gotaskq/internal/retry"
	"github.com/example/gotaskq/internal/scheduler"
	"github.com/example/gotaskq/internal/service"
	"github.com/example/gotaskq/internal/store"
	"github.com/example/gotaskq/internal/worker"
	"github.com/example/gotaskq/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintf(os.Stderr, "gotaskq: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	log, err := logger.New(logger.Config{
		Level:       cfg.Logger.Level,
		Format:      cfg.Logger.Format,
		ServiceName: cfg.Logger.ServiceName,
		Environment: cfg.Logger.Environment,
		Pretty:      cfg.Logger.Pretty,
		AddCaller:   cfg.Logger.AddCaller,
	})
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}
	logger.ConfigureGlobal(log)

	reg := metrics.NewRegistry(cfg.Metrics.Namespace, cfg.Metrics.Subsystem)
	if err := reg.Register(prometheus.DefaultRegisterer); err != nil {
		log.Warn().Err(err).Msg("metrics already registered")
	}

	pgCfg, err := pgxpool.ParseConfig(cfg.Postgres.DSN)
	if err != nil {
		return fmt.Errorf("postgres config: %w", err)
	}
	pgCfg.MaxConns = cfg.Postgres.MaxConns
	pgCfg.MinConns = cfg.Postgres.MinConns
	pgCfg.MaxConnLifetime = cfg.Postgres.MaxConnLifetime
	pgCfg.MaxConnIdleTime = cfg.Postgres.MaxConnIdleTime
	pgCfg.HealthCheckPeriod = cfg.Postgres.HealthCheckPeriod

	pgPool, err := pgxpool.NewWithConfig(ctx, pgCfg)
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	defer pgPool.Close()

	jobStore := store.NewPostgresStore(pgPool, "jobs")

	var redisClients []redis.UniversalClient
	for _, addr := range cfg.Redis.Addresses {
		redisClients = append(redisClients, redis.NewUniversalClient(&redis.UniversalOptions{
			Addrs:    []string{addr},
			Username: cfg.Redis.Username,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.Database,
			PoolSize: cfg.Redis.PoolSize,
		}))
	}
	defer func() {
		for _, c := range redisClients {
			c.Close()
		}
	}()

	// One lock manager shared across workers prevents duplicate execution of the same
	// job ID across multiple service instances.
	lockMgr := lock.NewManager(redisClients, lock.Config{
		TTL:         30 * time.Second,
		RetryCount:  3,
		RetryDelay:  100 * time.Millisecond,
		DriftFactor: 0.01,
	})

	kafkaClient, err := queue.NewKafkaClient(cfg.Kafka)
	if err != nil {
		return fmt.Errorf("kafka: %w", err)
	}
	defer kafkaClient.Close()

	jobSvc := service.NewJobService(kafkaClient, jobStore, cfg.Kafka.Topic, logger.WithComponent(log, "service"))

	breaker := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		OpenTimeout:      30 * time.Second,
		HalfOpenRequests: 1,
	})

	retryEngine := retry.NewEngine(retry.Config{
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2,
		MaxAttempts: 5,
		Jitter:      0.1,
	})

	runner := &jobWorker{
		store:    jobStore,
		lockMgr:  lockMgr,
		breaker:  breaker,
		retry:    retryEngine,
		metrics:  reg,
		log:      logger.WithComponent(log, "worker"),
		kafka:    kafkaClient,
		topic:    cfg.Kafka.Topic,
		handlers: defaultHandlers(),
	}
	workerPool := worker.NewPool(worker.Config{
		Concurrency:     cfg.Worker.Concurrency,
		QueueSize:       cfg.Worker.QueueSize,
		ShutdownTimeout: cfg.Worker.ShutdownTimeout,
	}, runner)
	workerPool.Start(ctx)

	go func() {
		h := &kafkaJobHandler{pool: workerPool, log: logger.WithComponent(log, "consumer")}
		if err := kafkaClient.Consume(ctx, []string{cfg.Kafka.Topic}, h); err != nil && !errors.Is(err, context.Canceled) {
			log.Error().Err(err).Msg("kafka consumer stopped")
		}
	}()

	sched := scheduler.New(cfg.Scheduler.TickInterval)
	if cfg.Scheduler.Enabled {
		sched.Start(ctx)
	}
	defer sched.Stop()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	handler := api.NewHandler(jobSvc, jobStore, logger.WithComponent(log, "api"), reg)
	handler.RegisterRoutes(router)
	router.GET("/metrics", gin.WrapH(reg.Handler()))
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	srv := &http.Server{
		Addr:         cfg.HTTP.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	go func() {
		log.Info().Str("addr", cfg.HTTP.Address).Msg("server listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("listen error")
		}
	}()

	<-ctx.Done()
	log.Info().Msg("shutting down")

	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		log.Warn().Err(err).Msg("HTTP server shutdown")
	}
	if err := workerPool.Stop(shutCtx); err != nil {
		log.Warn().Err(err).Msg("worker pool drain timeout")
	}

	return nil
}

// TaskHandler is the application-side function executed for a job.
type TaskHandler func(context.Context, models.Job) error

// defaultHandlers returns the registry of task handlers. Add new entries here
// to wire up real task logic; jobs whose Task.Name is unregistered fail with
// retry.ErrNoRetry so they go straight to DEAD instead of looping forever.
func defaultHandlers() map[string]TaskHandler {
	return map[string]TaskHandler{}
}

// jobWorker implements worker.JobRunner. It wraps execution with a distributed lock,
// circuit breaker, retry policy, and per-job timeout.
type jobWorker struct {
	store    store.JobStore
	lockMgr  *lock.Manager
	breaker  *circuitbreaker.Breaker
	retry    *retry.Engine
	metrics  *metrics.Registry
	log      zerolog.Logger
	kafka    service.Publisher
	topic    string
	handlers map[string]TaskHandler
}

// maxInWorkerSleep caps how long a worker will block waiting for a job's
// scheduled_at. Longer than this and we drop back to the queue rather than
// hold the worker slot.
const maxInWorkerSleep = 60 * time.Second

func (jw *jobWorker) Run(ctx context.Context, job models.Job) error {
	// Honor scheduled_at so retry backoff actually delays execution.
	if job.ScheduledAt != nil {
		if wait := time.Until(*job.ScheduledAt); wait > 0 {
			if wait > maxInWorkerSleep {
				// Re-publish so the slot frees up; another consumer will pick
				// it up after Kafka redelivery and the math will be smaller.
				jw.log.Debug().Str("job_id", job.ID).Dur("wait", wait).Msg("scheduled too far ahead, re-publishing")
				if err := jw.kafka.Publish(ctx, jw.topic, job); err != nil {
					jw.log.Warn().Err(err).Str("job_id", job.ID).Msg("re-publish failed")
				}
				return nil
			}
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	if !jw.breaker.Allow() {
		jw.log.Warn().Str("job_id", job.ID).Msg("circuit open, skipping execution")
		return nil
	}

	// Distributed lock prevents the same job from executing twice across
	// multiple instances (e.g. if it was re-enqueued before the first run commits).
	lck, err := jw.lockMgr.Acquire(ctx, "job:exec:"+job.ID)
	if err != nil {
		jw.log.Warn().Err(err).Str("job_id", job.ID).Msg("could not acquire execution lock")
		return nil
	}
	defer jw.lockMgr.Release(ctx, lck) //nolint:errcheck

	// Move PENDING → RUNNING and stamp started_at.
	now := time.Now().UTC()
	job.Attempt++
	job.State = models.JobStateRunning
	job.StartedAt = &now
	if err := jw.store.UpdateJob(ctx, job); err != nil {
		jw.log.Error().Err(err).Str("job_id", job.ID).Msg("failed to mark job running")
		return err
	}

	jw.metrics.WorkerInFlight.Inc()
	start := time.Now()
	defer func() {
		jw.metrics.WorkerInFlight.Dec()
		jw.metrics.JobDuration.Observe(time.Since(start).Seconds())
	}()

	jw.metrics.JobStarted.Inc()

	if err := jw.execute(ctx, job); err != nil {
		jw.breaker.RecordFailure()
		jw.metrics.JobFailed.Inc()

		if jw.retry.ShouldRetry(job.Attempt, err) {
			return jw.scheduleRetry(ctx, job, err)
		}
		return jw.deadLetter(ctx, job, err)
	}

	jw.breaker.RecordSuccess()
	jw.metrics.JobCompleted.Inc()

	completedAt := time.Now().UTC()
	job.State = models.JobStateCompleted
	job.CompletedAt = &completedAt
	if err := jw.store.UpdateJob(ctx, job); err != nil {
		jw.log.Error().Err(err).Str("job_id", job.ID).Msg("failed to mark job complete")
		return err
	}
	return nil
}

// execute looks up a registered handler for job.Task.Name and runs it under
// the per-task timeout if one is set. Unknown task names short-circuit with
// retry.ErrNoRetry so they don't burn through the retry budget.
func (jw *jobWorker) execute(ctx context.Context, job models.Job) error {
	handler, ok := jw.handlers[job.Task.Name]
	if !ok {
		return fmt.Errorf("worker: no handler registered for task %q: %w", job.Task.Name, retry.ErrNoRetry)
	}

	if job.Task.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, job.Task.Timeout)
		defer cancel()
	}
	return handler(ctx, job)
}

// scheduleRetry transitions RUNNING → FAILED → PENDING, stamps scheduled_at
// with the next attempt's backoff, and re-publishes to Kafka so a worker
// (possibly this one, possibly another) picks it up after the delay.
func (jw *jobWorker) scheduleRetry(ctx context.Context, job models.Job, runErr error) error {
	delay := jw.retry.Delay(job.Attempt)
	nextRun := time.Now().UTC().Add(delay)

	// RUNNING → FAILED
	job.State = models.JobStateFailed
	job.LastError = runErr.Error()
	if err := jw.store.UpdateJob(ctx, job); err != nil {
		jw.log.Error().Err(err).Str("job_id", job.ID).Msg("failed to record retry state")
		return runErr
	}

	// FAILED → PENDING with future scheduled_at
	job.State = models.JobStatePending
	job.ScheduledAt = &nextRun
	job.StartedAt = nil
	if err := jw.store.UpdateJob(ctx, job); err != nil {
		jw.log.Error().Err(err).Str("job_id", job.ID).Msg("failed to reschedule retry")
		return runErr
	}

	if err := jw.kafka.Publish(ctx, jw.topic, job); err != nil {
		jw.log.Error().Err(err).Str("job_id", job.ID).Msg("failed to re-publish retry; will be reconciled on next ClaimNextJob")
	}

	jw.log.Warn().
		Err(runErr).
		Str("job_id", job.ID).
		Int("attempt", job.Attempt).
		Dur("retry_in", delay).
		Msg("scheduled retry")
	return runErr
}

// deadLetter transitions RUNNING → DEAD when retries are exhausted or the
// error is permanent (errors.Is(err, retry.ErrNoRetry)).
func (jw *jobWorker) deadLetter(ctx context.Context, job models.Job, runErr error) error {
	job.State = models.JobStateDead
	job.LastError = runErr.Error()
	if err := jw.store.UpdateJob(ctx, job); err != nil {
		jw.log.Error().Err(err).Str("job_id", job.ID).Msg("failed to dead-letter job")
	}
	jw.log.Error().Err(runErr).Str("job_id", job.ID).Int("attempts", job.Attempt).Msg("job exhausted retries")
	return runErr
}

// kafkaJobHandler bridges Kafka messages to the worker pool.
type kafkaJobHandler struct {
	pool *worker.Pool
	log  zerolog.Logger
}

func (h *kafkaJobHandler) Handle(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var job models.Job
	if err := json.Unmarshal(msg.Value, &job); err != nil {
		h.log.Error().Err(err).Msg("failed to unmarshal kafka message")
		return nil
	}
	if !h.pool.Submit(ctx, job) {
		h.log.Warn().Str("job_id", job.ID).Msg("worker pool full, job dropped")
	}
	return nil
}
