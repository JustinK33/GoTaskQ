package lock

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestNewManager(t *testing.T) {
	t.Run("captures clients and config", func(t *testing.T) {
		cfg := Config{Quorum: 2, TTL: time.Second, RetryCount: 3, RetryDelay: 100 * time.Millisecond, DriftFactor: 0.01}
		m := NewManager([]redis.UniversalClient{}, cfg)
		if m == nil {
			t.Fatal("NewManager returned nil")
		}
	})
}

func TestManagerAcquire(t *testing.T) {
	t.Skip("requires running Redis nodes: start the stack with `make docker-up`")
}

func TestManagerRelease(t *testing.T) {
	t.Skip("requires running Redis nodes: start the stack with `make docker-up`")
}
