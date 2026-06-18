package models

import "time"

// JobState is the durable lifecycle stage of a queued task.
type JobState string

const (
	JobStatePending   JobState = "PENDING"
	JobStateRunning   JobState = "RUNNING"
	JobStateCompleted JobState = "COMPLETED"
	JobStateFailed    JobState = "FAILED"
	JobStateDead      JobState = "DEAD"
)

// Task describes the payload and execution metadata for a job.
type Task struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Payload        []byte            `json:"payload,omitempty"`
	RetryCount     int               `json:"retry_count"`
	MaxRetries     int               `json:"max_retries"`
	Timeout        time.Duration     `json:"timeout"`
	CronExpression string            `json:"cron_expression,omitempty"`
	Queue          string            `json:"queue,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// Job is the durable execution record that moves through the state machine.
type Job struct {
	ID          string            `json:"id"`
	Task        Task              `json:"task"`
	State       JobState          `json:"state"`
	Attempt     int               `json:"attempt"`
	ScheduledAt *time.Time        `json:"scheduled_at,omitempty"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	LastError   string            `json:"last_error,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type HTTPConfig struct {
	Address      string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type KafkaConfig struct {
	Brokers           []string
	Topic             string
	ConsumerGroup     string
	ClientID          string
	RequiredAcks      int16
	CompressionCodec  int // 0=none 1=gzip 2=snappy 3=lz4 4=zstd
	FlushFrequencyMs  int
	FlushBytes        int
	ChannelBufferSize int
}

type RedisConfig struct {
	Addresses []string
	Username  string
	Password  string
	Database  int
	PoolSize  int
}

type PostgresConfig struct {
	DSN               string
	MaxConns          int32
	MinConns          int32
	MigrationsPath    string
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

type WorkerConfig struct {
	Concurrency     int
	QueueSize       int
	ShutdownTimeout time.Duration
}

type SchedulerConfig struct {
	Enabled           bool
	TickInterval      time.Duration
	MaxConcurrentRuns int
}

type MetricsConfig struct {
	Namespace     string
	Subsystem     string
	ListenAddress string
}

type LoggerConfig struct {
	Level       string
	Format      string
	ServiceName string
	Environment string
	Pretty      bool
	AddCaller   bool
}

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
