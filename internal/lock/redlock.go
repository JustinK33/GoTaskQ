package lock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Quorum      int
	TTL         time.Duration
	RetryCount  int
	RetryDelay  time.Duration
	DriftFactor float64
}

// Lock is a held distributed-lock lease.
type Lock struct {
	Resource string
	Value    string
	Expiry   time.Time
}

// Manager runs the Redlock quorum protocol across multiple Redis clients.
type Manager struct {
	clients []redis.UniversalClient
	cfg     Config
}

func NewManager(clients []redis.UniversalClient, cfg Config) *Manager {
	if cfg.Quorum <= 0 {
		cfg.Quorum = len(clients)/2 + 1
	}
	if cfg.TTL <= 0 {
		cfg.TTL = 10 * time.Second
	}
	if cfg.RetryCount <= 0 {
		cfg.RetryCount = 3
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = 200 * time.Millisecond
	}
	if cfg.DriftFactor <= 0 {
		cfg.DriftFactor = 0.01
	}
	return &Manager{clients: clients, cfg: cfg}
}

// Acquire claims a Redlock-style quorum lease on resource. Retries up to
// cfg.RetryCount times with cfg.RetryDelay between attempts.
func (m *Manager) Acquire(ctx context.Context, resource string) (Lock, error) {
	value, err := generateToken()
	if err != nil {
		return Lock{}, fmt.Errorf("lock: generate token: %w", err)
	}

	for attempt := 0; attempt < m.cfg.RetryCount; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return Lock{}, ctx.Err()
			case <-time.After(m.cfg.RetryDelay):
			}
		}

		start := time.Now()
		acquired := 0

		for _, client := range m.clients {
			ok, setErr := client.SetNX(ctx, resource, value, m.cfg.TTL).Result()
			if setErr == nil && ok {
				acquired++
			}
		}

		elapsed := time.Since(start)
		drift := time.Duration(float64(m.cfg.TTL)*m.cfg.DriftFactor) + 2*time.Millisecond
		validity := m.cfg.TTL - elapsed - drift

		if acquired >= m.cfg.Quorum && validity > 0 {
			return Lock{
				Resource: resource,
				Value:    value,
				Expiry:   time.Now().Add(validity),
			}, nil
		}

		// Partial win - release all acquired locks before retrying.
		for _, client := range m.clients {
			releaseSingle(ctx, client, resource, value)
		}
	}

	return Lock{}, fmt.Errorf("lock: could not acquire lock on %s after %d attempts", resource, m.cfg.RetryCount)
}

// releaseScript atomically deletes the key only when the value matches the caller's token.
var releaseScript = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
else
	return 0
end
`)

func releaseSingle(ctx context.Context, client redis.UniversalClient, resource, value string) {
	releaseScript.Run(ctx, client, []string{resource}, value)
}

func (m *Manager) Release(ctx context.Context, lock Lock) error {
	var errs []error
	for _, client := range m.clients {
		err := releaseScript.Run(ctx, client, []string{lock.Resource}, lock.Value).Err()
		if err != nil && !errors.Is(err, redis.Nil) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
