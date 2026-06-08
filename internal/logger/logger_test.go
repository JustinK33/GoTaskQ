package logger

import (
	"testing"

	"github.com/rs/zerolog"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "builds a structured logger", cfg: Config{ServiceName: "gotaskq"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_ = tc.cfg
			// Assert that level, format, and metadata are reflected in the returned zerolog.Logger.
		})
	}
}

func TestConfigureGlobal(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "installs the global logger"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Assert that the global logger is set exactly once and remains deterministic.
			ConfigureGlobal(zerolog.Logger{})
		})
	}
}

func TestWithComponent(t *testing.T) {
	tests := []struct {
		name      string
		component string
	}{
		{name: "adds component field", component: "api"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_ = tc.component
			// Assert that subsystem-scoped logging preserves the parent logger behavior.
		})
	}
}
