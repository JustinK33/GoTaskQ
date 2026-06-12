package retry

import (
	"errors"
	"testing"
	"time"
)

var errTest = errors.New("test error")

// BenchmarkDelay measures exponential backoff computation cost.
func BenchmarkDelay(b *testing.B) {
	engine := NewEngine(Config{
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
		MaxAttempts: 10,
		Jitter:      0.1,
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Delay(i % 10)
	}
}

// BenchmarkDelayNoJitter measures delay without jitter (deterministic path).
func BenchmarkDelayNoJitter(b *testing.B) {
	engine := NewEngine(Config{
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
		MaxAttempts: 10,
		Jitter:      0,
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Delay(i % 10)
	}
}

// BenchmarkShouldRetry measures the retry eligibility check.
func BenchmarkShouldRetry(b *testing.B) {
	engine := NewEngine(Config{
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
		MaxAttempts: 5,
		Jitter:      0.1,
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.ShouldRetry(i%6, errTest)
	}
}
