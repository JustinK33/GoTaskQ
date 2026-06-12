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

	jobSvc := service.NewJobService(kafkaClient, jobStore, cfg.Kafka.Topic)

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
		store:   jobStore,
		lockMgr: lockMgr,
		breaker: breaker,
		retry:   retryEngine,
		metrics: reg,
		log:     logger.WithComponent(log, "worker"),
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

// jobWorker implements worker.JobRunner. It wraps execution with a distributed lock,
// circuit breaker, and retry policy so the pool itself stays transport-agnostic.
type jobWorker struct {
	store   store.JobStore
	lockMgr *lock.Manager
	breaker *circuitbreaker.Breaker
	retry   *retry.Engine
	metrics *metrics.Registry
	log     zerolog.Logger
}

func (jw *jobWorker) Run(ctx context.Context, job models.Job) error {
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
			job.State = models.JobStateFailed
			job.LastError = err.Error()
			job.Task.RetryCount++
			if updateErr := jw.store.UpdateJob(ctx, job); updateErr != nil {
				jw.log.Error().Err(updateErr).Str("job_id", job.ID).Msg("failed to record retry state")
			}
			return err
		}

		// Retries exhausted — move to dead-letter queue.
		job.State = models.JobStateDead
		job.LastError = err.Error()
		if updateErr := jw.store.UpdateJob(ctx, job); updateErr != nil {
			jw.log.Error().Err(updateErr).Str("job_id", job.ID).Msg("failed to dead-letter job")
		}
		jw.log.Error().Err(err).Str("job_id", job.ID).Int("attempts", job.Attempt).Msg("job exhausted retries")
		return err
	}

	jw.breaker.RecordSuccess()
	jw.metrics.JobCompleted.Inc()

	now := time.Now().UTC()
	job.State = models.JobStateCompleted
	job.CompletedAt = &now
	if err := jw.store.UpdateJob(ctx, job); err != nil {
		jw.log.Error().Err(err).Str("job_id", job.ID).Msg("failed to mark job complete")
		return err
	}
	return nil
}

// execute dispatches to the task handler registered for job.Task.Name.
// A production system would maintain a handler registry (map[string]TaskHandler)
// populated at startup; here the dispatch point is left explicit for extensibility.
func (jw *jobWorker) execute(_ context.Context, _ models.Job) error {
	return nil
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
