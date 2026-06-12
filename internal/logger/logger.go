package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Config captures zerolog setup options for the application.
type Config struct {
	Level       string
	Format      string
	ServiceName string
	Environment string
	Pretty      bool
	AddCaller   bool
}

// New builds the service logger using the supplied configuration.
func New(cfg Config) (zerolog.Logger, error) {
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		return zerolog.Logger{}, fmt.Errorf("logger: invalid level %q: %w", cfg.Level, err)
	}

	var output io.Writer
	if cfg.Pretty {
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	} else {
		output = os.Stdout
	}

	ctx := zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Str("service", cfg.ServiceName).
		Str("env", cfg.Environment)

	if cfg.AddCaller {
		ctx = ctx.Caller()
	}

	return ctx.Logger(), nil
}

// ConfigureGlobal installs the logger as the process-wide default.
func ConfigureGlobal(log zerolog.Logger) {
	zerolog.DefaultContextLogger = &log
}

// WithComponent returns a logger scoped to a named subsystem.
func WithComponent(log zerolog.Logger, component string) zerolog.Logger {
	return log.With().Str("component", component).Logger()
}
