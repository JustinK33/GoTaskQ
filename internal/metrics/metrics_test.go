package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

type mockRegisterer struct{}

func (mockRegisterer) Register(prometheus.Collector) error  { return nil }
func (mockRegisterer) MustRegister(...prometheus.Collector) {}
func (mockRegisterer) Unregister(prometheus.Collector) bool { return true }

func TestNewRegistry(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		subsystem string
	}{
		{name: "builds namespace-scoped collectors", namespace: "conduit", subsystem: "worker"},
		{name: "different namespace", namespace: "myapp", subsystem: "api"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRegistry(tc.namespace, tc.subsystem)
			if r == nil {
				t.Fatal("NewRegistry returned nil")
			}
			if r.Namespace != tc.namespace {
				t.Errorf("Namespace = %q, want %q", r.Namespace, tc.namespace)
			}
			if r.Subsystem != tc.subsystem {
				t.Errorf("Subsystem = %q, want %q", r.Subsystem, tc.subsystem)
			}
			if r.JobEnqueued == nil {
				t.Error("JobEnqueued collector is nil")
			}
			if r.JobDuration == nil {
				t.Error("JobDuration collector is nil")
			}
		})
	}
}

func TestRegistryRegister(t *testing.T) {
	t.Run("registers collectors without error", func(t *testing.T) {
		r := NewRegistry("conduit", "worker")
		if r == nil {
			t.Skip("NewRegistry not yet implemented")
		}
		if err := r.Register(mockRegisterer{}); err != nil {
			t.Errorf("Register returned unexpected error: %v", err)
		}
	})
}

func TestRegistryHandler(t *testing.T) {
	t.Run("returns a non-nil scrape handler", func(t *testing.T) {
		r := NewRegistry("conduit", "worker")
		if r == nil {
			t.Skip("NewRegistry not yet implemented")
		}
		h := r.Handler()
		if h == nil {
			t.Error("Handler() returned nil")
		}
	})
}
