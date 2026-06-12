package circuitbreaker

import (
	"testing"
	"time"
)

// BenchmarkBreakerAllowClosed measures Allow() overhead in the normal (Closed) path.
func BenchmarkBreakerAllowClosed(b *testing.B) {
	br := New(Config{
		FailureThreshold: 1000,
		SuccessThreshold: 2,
		OpenTimeout:      time.Hour,
		HalfOpenRequests: 1,
	})
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		br.Allow()
	}
}

// BenchmarkBreakerAllowOpen measures Allow() overhead in the open (tripping) path.
func BenchmarkBreakerAllowOpen(b *testing.B) {
	br := New(Config{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		OpenTimeout:      time.Hour,
		HalfOpenRequests: 1,
	})
	br.RecordFailure() // trip open
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		br.Allow()
	}
}

// BenchmarkBreakerRecordSuccess measures RecordSuccess() in HalfOpen state.
func BenchmarkBreakerRecordSuccess(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		br := New(Config{
			FailureThreshold: 1,
			SuccessThreshold: 2,
			OpenTimeout:      0,
			HalfOpenRequests: 1,
		})
		br.RecordFailure()
		br.Allow() // transitions to HalfOpen
		b.StartTimer()
		br.RecordSuccess()
	}
}

// BenchmarkBreakerContention measures concurrent Allow/RecordFailure under lock contention.
func BenchmarkBreakerContention(b *testing.B) {
	br := New(Config{
		FailureThreshold: 1000,
		SuccessThreshold: 2,
		OpenTimeout:      time.Hour,
		HalfOpenRequests: 1,
	})
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if br.Allow() {
				br.RecordSuccess()
			}
		}
	})
}
