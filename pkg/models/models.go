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

// Job is the durable execution record that moves through the state machine.
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
