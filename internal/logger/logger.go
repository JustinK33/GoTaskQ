package logger

import "github.com/rs/zerolog"

// Config captures zerolog setup options for the application.
// The logger package keeps configuration separate so main can stay focused on wiring.
type Config struct {
	Level       string
	Format      string
	ServiceName string
	Environment string
	Pretty      bool
	AddCaller   bool
}

// New builds the service logger using the supplied configuration.
// Inputs and outputs:
// - cfg controls log level, format, and metadata.
// - returns the configured logger and any initialization error.
// Key implementation steps:
// 1. Parse the requested log level.
// 2. Configure time encoding and output formatting.
// 3. Attach service metadata.
// 4. Return a ready-to-use zerolog.Logger.
// Gotchas:
// - Keep JSON and pretty output behaviors consistent across environments.
func New(cfg Config) (log zerolog.Logger, err error) {
	// Implementation intentionally omitted.
	return
}

// ConfigureGlobal installs the logger as the process-wide default logger.
// Inputs and outputs:
// - log is the configured zerolog logger to expose globally.
// - returns nothing.
// Key implementation steps:
// 1. Set the global zerolog default.
// 2. Apply any project-wide field normalization.
// 3. Keep the configuration deterministic for tests.
// Gotchas:
// - Global state should be set once during bootstrap only.
func ConfigureGlobal(log zerolog.Logger) {
	// Implementation intentionally omitted.
}

// WithComponent returns a logger scoped to a subsystem or package name.
// Inputs and outputs:
// - log is the parent structured logger.
// - component names the subsystem being traced.
// - returns a child logger with the component field attached.
// Key implementation steps:
// 1. Add subsystem metadata.
// 2. Preserve the parent's output destination.
// 3. Return the child logger for downstream use.
// Gotchas:
// - Keep component names stable so dashboards remain searchable.
func WithComponent(log zerolog.Logger, component string) (child zerolog.Logger) {
	// Implementation intentionally omitted.
	return
}
