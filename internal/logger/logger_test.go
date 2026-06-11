package logger

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		wantLevel zerolog.Level
		wantErr   bool
	}{
		{
			name:      "debug level",
			cfg:       Config{Level: "debug", ServiceName: "gotaskq"},
			wantLevel: zerolog.DebugLevel,
		},
		{
			name:      "info level",
			cfg:       Config{Level: "info", ServiceName: "gotaskq"},
			wantLevel: zerolog.InfoLevel,
		},
		{
			name:      "warn level",
			cfg:       Config{Level: "warn", ServiceName: "gotaskq"},
			wantLevel: zerolog.WarnLevel,
		},
		{
			name:      "invalid level returns error",
			cfg:       Config{Level: "notavalidlevel"},
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			log, err := New(tc.cfg)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error for invalid level but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := log.GetLevel(); got != tc.wantLevel {
				t.Errorf("GetLevel() = %v, want %v", got, tc.wantLevel)
			}
		})
	}
}

func TestConfigureGlobal(t *testing.T) {
	t.Run("installs global logger without panic", func(t *testing.T) {
		// ConfigureGlobal sets global state; just verify it doesn't panic.
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ConfigureGlobal panicked: %v", r)
			}
		}()
		ConfigureGlobal(zerolog.Logger{})
	})
}

func TestWithComponent(t *testing.T) {
	tests := []struct {
		name      string
		component string
	}{
		{name: "adds api component field", component: "api"},
		{name: "adds worker component field", component: "worker"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			parent := zerolog.New(buf)

			child := WithComponent(parent, tc.component)

			// Write a log entry through the child logger and verify the component field appears.
			child.Info().Msg("test")
			output := buf.String()
			if output == "" {
				t.Error("WithComponent returned a logger that produces no output")
			}
			// The JSON output should contain the component name.
			if len(output) > 0 && !bytes.Contains(buf.Bytes(), []byte(tc.component)) {
				t.Errorf("log output %q missing component field %q", output, tc.component)
			}
		})
	}
}
