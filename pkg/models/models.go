package models

import "time"

// JobState represents the lifecycle stage of a queued task.
// The constants define the durable state machine used by the API, store, and worker layers.
type JobState string

const (
	JobStatePending   JobState = "PENDING"
	JobStateRunning   JobState = "RUNNING"
	JobStateCompleted JobState = "COMPLETED"
	JobStateFailed    JobState = "FAILED"
	JobStateDead      JobState = "DEAD"
)

// Task describes the payload and execution metadata for a job.
// The fields are broad enough to support queueing, retries, and recurring schedules.
type Task struct {
	ID             string
	Name           string
	Payload        []byte
	RetryCount     int
	MaxRetries     int
	Timeout        time.Duration
	CronExpression string
	Queue          string
	Metadata       map[string]string
}

// Job is the durable execution record that moves through the queue and store state machine.
// Timestamps and error fields allow the service to report status transitions clearly.
type Job struct {
	ID          string
	Task        Task
	State       JobState
	Attempt     int
	ScheduledAt *time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	LastError   string
	Metadata    map[string]string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// HTTPConfig configures the server listener and request timeout behavior.
// The API layer and main entrypoint consume this as part of the service bootstrap contract.
type HTTPConfig struct {
	Address      string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// KafkaConfig describes the queue transport topology.
// The consumer group and dead-letter topic are included so retries and failures can be routed explicitly.
type KafkaConfig struct {
	Brokers           []string
	Topic             string
	ConsumerGroup     string
	DeadLetterTopic   string
	ClientID          string
	RequiredAcks      int16
	CompressionCodec  int           // 0=none 1=gzip 2=snappy 3=lz4 4=zstd
	FlushFrequencyMs  int           // producer flush interval in milliseconds
	FlushBytes        int           // producer flush threshold in bytes
	ChannelBufferSize int           // sarama internal channel buffer depth
}

// RedisConfig configures the distributed locking and coordination layer.
// Multiple addresses support quorum-style lock acquisition.
type RedisConfig struct {
	Addresses []string
	Username  string
	Password  string
	Database  int
	PoolSize  int
}

// PostgresConfig configures the durable job state store.
// The DSN and pool settings are consumed by the store package.
type PostgresConfig struct {
	DSN               string
	MaxConns          int32
	MinConns          int32
	MigrationsPath    string
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

// WorkerConfig controls the goroutine pool and shutdown behavior.
// The semaphore-based worker pool will use these values to bound concurrent execution.
type WorkerConfig struct {
	Concurrency     int
	QueueSize       int
	ShutdownTimeout time.Duration
}

// SchedulerConfig configures recurring job dispatch behavior.
// The scheduler package can use these values to plan cron-style job launches.
type SchedulerConfig struct {
	Enabled           bool
	TickInterval      time.Duration
	MaxConcurrentRuns int
}

// MetricsConfig controls Prometheus registration and exposure.
// Namespace and listener settings keep metrics deployment explicit.
type MetricsConfig struct {
	Namespace     string
	Subsystem     string
	ListenAddress string
}

// LoggerConfig defines structured logging preferences for zerolog.
// The configuration is kept independent of the logger implementation details.
type LoggerConfig struct {
	Level       string
	Format      string
	ServiceName string
	Environment string
	Pretty      bool
	AddCaller   bool
}

// Config aggregates the service-wide configuration tree.
// Main and the subsystem constructors can consume this shape without knowing the source of the values.
type Config struct {
	HTTP      HTTPConfig
	Kafka     KafkaConfig
	Redis     RedisConfig
	Postgres  PostgresConfig
	Worker    WorkerConfig
	Scheduler SchedulerConfig
	Metrics   MetricsConfig
	Logger    LoggerConfig
}
