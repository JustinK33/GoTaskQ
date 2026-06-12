package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/example/gotaskq/pkg/models"
)

// Environment abstracts environment lookups so config loading can be tested without process state.
type Environment interface {
	LookupEnv(key string) (string, bool)
}

type osEnv struct{}

func (o osEnv) LookupEnv(key string) (string, bool) { return os.LookupEnv(key) }

// Default returns sensible production-ready defaults for all subsystems.
func Default() models.Config {
	return models.Config{
		HTTP: models.HTTPConfig{
			Address:      ":8080",
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Kafka: models.KafkaConfig{
			Brokers:           []string{"localhost:9092"},
			Topic:             "jobs",
			ConsumerGroup:     "gotaskq-workers",
			DeadLetterTopic:   "jobs-dead",
			ClientID:          "gotaskq",
			RequiredAcks:      1,
			CompressionCodec:  2,       // snappy
			FlushFrequencyMs:  5,
			FlushBytes:        1048576, // 1 MiB
			ChannelBufferSize: 256,
		},
		Redis: models.RedisConfig{
			Addresses: []string{"localhost:6379", "localhost:6380", "localhost:6381"},
			Database:  0,
			PoolSize:  10,
		},
		Postgres: models.PostgresConfig{
			DSN:               "postgres://postgres:postgres@localhost:5432/gotaskq?sslmode=disable",
			MaxConns:          50,
			MinConns:          10,
			MigrationsPath:    "migrations",
			MaxConnLifetime:   30 * time.Minute,
			MaxConnIdleTime:   5 * time.Minute,
			HealthCheckPeriod: time.Minute,
		},
		Worker: models.WorkerConfig{
			Concurrency:     50,
			QueueSize:       500,
			ShutdownTimeout: 30 * time.Second,
		},
		Scheduler: models.SchedulerConfig{
			Enabled:           true,
			TickInterval:      time.Minute,
			MaxConcurrentRuns: 5,
		},
		Metrics: models.MetricsConfig{
			Namespace:     "gotaskq",
			Subsystem:     "server",
			ListenAddress: ":9090",
		},
		Logger: models.LoggerConfig{
			Level:       "info",
			Format:      "json",
			ServiceName: "gotaskq",
			Environment: "development",
		},
	}
}

// Load reads the process environment and returns a validated config.
func Load() (models.Config, error) {
	return LoadFromEnvironment(osEnv{})
}

// LoadFromEnvironment resolves config from the supplied environment reader, merging with defaults.
func LoadFromEnvironment(env Environment) (models.Config, error) {
	cfg := Default()

	str := func(key string, dest *string) {
		if v, ok := env.LookupEnv(key); ok && v != "" {
			*dest = v
		}
	}
	strs := func(key string, dest *[]string) {
		if v, ok := env.LookupEnv(key); ok && v != "" {
			*dest = strings.Split(v, ",")
		}
	}
	dur := func(key string, dest *time.Duration) {
		if v, ok := env.LookupEnv(key); ok && v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				*dest = d
			}
		}
	}
	intV := func(key string, dest *int) {
		if v, ok := env.LookupEnv(key); ok && v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				*dest = n
			}
		}
	}
	int32V := func(key string, dest *int32) {
		if v, ok := env.LookupEnv(key); ok && v != "" {
			if n, err := strconv.ParseInt(v, 10, 32); err == nil {
				*dest = int32(n)
			}
		}
	}
	int16V := func(key string, dest *int16) {
		if v, ok := env.LookupEnv(key); ok && v != "" {
			if n, err := strconv.ParseInt(v, 10, 16); err == nil {
				*dest = int16(n)
			}
		}
	}
	boolV := func(key string, dest *bool) {
		if v, ok := env.LookupEnv(key); ok && v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				*dest = b
			}
		}
	}

	str("HTTP_ADDRESS", &cfg.HTTP.Address)
	dur("HTTP_READ_TIMEOUT", &cfg.HTTP.ReadTimeout)
	dur("HTTP_WRITE_TIMEOUT", &cfg.HTTP.WriteTimeout)
	dur("HTTP_IDLE_TIMEOUT", &cfg.HTTP.IdleTimeout)

	strs("KAFKA_BROKERS", &cfg.Kafka.Brokers)
	str("KAFKA_TOPIC", &cfg.Kafka.Topic)
	str("KAFKA_CONSUMER_GROUP", &cfg.Kafka.ConsumerGroup)
	str("KAFKA_DEAD_LETTER_TOPIC", &cfg.Kafka.DeadLetterTopic)
	str("KAFKA_CLIENT_ID", &cfg.Kafka.ClientID)
	int16V("KAFKA_REQUIRED_ACKS", &cfg.Kafka.RequiredAcks)
	intV("KAFKA_COMPRESSION_CODEC", &cfg.Kafka.CompressionCodec)
	intV("KAFKA_FLUSH_FREQUENCY_MS", &cfg.Kafka.FlushFrequencyMs)
	intV("KAFKA_FLUSH_BYTES", &cfg.Kafka.FlushBytes)
	intV("KAFKA_CHANNEL_BUFFER_SIZE", &cfg.Kafka.ChannelBufferSize)

	strs("REDIS_ADDRESSES", &cfg.Redis.Addresses)
	str("REDIS_USERNAME", &cfg.Redis.Username)
	str("REDIS_PASSWORD", &cfg.Redis.Password)
	intV("REDIS_DATABASE", &cfg.Redis.Database)
	intV("REDIS_POOL_SIZE", &cfg.Redis.PoolSize)

	str("POSTGRES_DSN", &cfg.Postgres.DSN)
	int32V("POSTGRES_MAX_CONNS", &cfg.Postgres.MaxConns)
	int32V("POSTGRES_MIN_CONNS", &cfg.Postgres.MinConns)
	str("POSTGRES_MIGRATIONS_PATH", &cfg.Postgres.MigrationsPath)
	dur("POSTGRES_MAX_CONN_LIFETIME", &cfg.Postgres.MaxConnLifetime)
	dur("POSTGRES_MAX_CONN_IDLE_TIME", &cfg.Postgres.MaxConnIdleTime)
	dur("POSTGRES_HEALTH_CHECK_PERIOD", &cfg.Postgres.HealthCheckPeriod)

	intV("WORKER_CONCURRENCY", &cfg.Worker.Concurrency)
	intV("WORKER_QUEUE_SIZE", &cfg.Worker.QueueSize)
	dur("WORKER_SHUTDOWN_TIMEOUT", &cfg.Worker.ShutdownTimeout)

	boolV("SCHEDULER_ENABLED", &cfg.Scheduler.Enabled)
	dur("SCHEDULER_TICK_INTERVAL", &cfg.Scheduler.TickInterval)
	intV("SCHEDULER_MAX_CONCURRENT_RUNS", &cfg.Scheduler.MaxConcurrentRuns)

	str("METRICS_NAMESPACE", &cfg.Metrics.Namespace)
	str("METRICS_SUBSYSTEM", &cfg.Metrics.Subsystem)
	str("METRICS_LISTEN_ADDRESS", &cfg.Metrics.ListenAddress)

	str("LOG_LEVEL", &cfg.Logger.Level)
	str("LOG_FORMAT", &cfg.Logger.Format)
	str("LOG_SERVICE_NAME", &cfg.Logger.ServiceName)
	str("LOG_ENVIRONMENT", &cfg.Logger.Environment)
	boolV("LOG_PRETTY", &cfg.Logger.Pretty)
	boolV("LOG_ADD_CALLER", &cfg.Logger.AddCaller)

	return cfg, Validate(cfg)
}

// Validate checks required fields and rejects incoherent values.
func Validate(cfg models.Config) error {
	if cfg.Postgres.DSN == "" {
		return fmt.Errorf("config: POSTGRES_DSN is required")
	}
	if len(cfg.Kafka.Brokers) == 0 {
		return fmt.Errorf("config: KAFKA_BROKERS is required")
	}
	if cfg.Kafka.Topic == "" {
		return fmt.Errorf("config: KAFKA_TOPIC is required")
	}
	if cfg.Worker.Concurrency <= 0 {
		return fmt.Errorf("config: WORKER_CONCURRENCY must be > 0, got %d", cfg.Worker.Concurrency)
	}
	return nil
}
