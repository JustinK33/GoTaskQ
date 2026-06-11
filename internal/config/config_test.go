package config

import (
	"testing"

	"github.com/example/gotaskq/pkg/models"
)

type mockEnvironment struct {
	values map[string]string
}

func (m mockEnvironment) LookupEnv(key string) (string, bool) {
	value, ok := m.values[key]
	return value, ok
}

func TestDefault(t *testing.T) {
	t.Run("returns a config without panicking", func(t *testing.T) {
		cfg := Default()
		// Default() must return a usable zero-value config.
		// Once you implement it, assert the specific defaults here.
		_ = cfg
	})
}

func TestLoad(t *testing.T) {
	t.Run("returns config from process environment", func(t *testing.T) {
		// Load reads real environment variables; set them in the test process if needed.
		_, err := Load()
		// An unset environment may return an error or a zero config — both are valid outcomes.
		// Once you implement Validate, assert that a fully set env returns err == nil.
		_ = err
	})
}

func TestLoadFromEnvironment(t *testing.T) {
	tests := []struct {
		name    string
		env     mockEnvironment
		wantErr bool
	}{
		{
			name: "reads HTTP address from environment",
			env: mockEnvironment{values: map[string]string{
				"HTTP_ADDRESS": ":8080",
			}},
			wantErr: false,
		},
		{
			name:    "empty environment returns default or error",
			env:     mockEnvironment{values: map[string]string{}},
			wantErr: false, // adjust once you decide whether empty env is an error
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := LoadFromEnvironment(tc.env)
			if tc.wantErr && err == nil {
				t.Error("expected an error but got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			// Assert the specific field once you pick your env var names.
			_ = cfg
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     models.Config
		wantErr bool
	}{
		{
			name:    "empty config is invalid",
			cfg:     models.Config{},
			wantErr: true,
		},
		{
			name: "config with required fields is valid",
			cfg: models.Config{
				HTTP:    models.HTTPConfig{Address: ":8080", ReadTimeout: 5e9, WriteTimeout: 5e9},
				Kafka:   models.KafkaConfig{Brokers: []string{"localhost:9092"}, Topic: "jobs", ConsumerGroup: "workers"},
				Postgres: models.PostgresConfig{DSN: "postgres://localhost/gotaskq"},
				Worker:  models.WorkerConfig{Concurrency: 4, QueueSize: 32},
			},
			wantErr: false,
		},
		{
			name: "zero worker concurrency is invalid",
			cfg: models.Config{
				HTTP:    models.HTTPConfig{Address: ":8080", ReadTimeout: 5e9, WriteTimeout: 5e9},
				Kafka:   models.KafkaConfig{Brokers: []string{"localhost:9092"}, Topic: "jobs", ConsumerGroup: "workers"},
				Postgres: models.PostgresConfig{DSN: "postgres://localhost/gotaskq"},
				Worker:  models.WorkerConfig{Concurrency: 0, QueueSize: 32},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.cfg)
			if tc.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
