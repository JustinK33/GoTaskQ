package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/example/gotaskq/internal/queue"
	"github.com/example/gotaskq/internal/store"
	"github.com/example/gotaskq/pkg/models"
)

// JobService implements the api.Queue interface by bridging KafkaClient and PostgresStore.
type JobService struct {
	kafka *queue.KafkaClient
	store store.JobStore
	topic string
}

// NewJobService creates a job service that persists to Postgres and publishes to Kafka.
func NewJobService(kafka *queue.KafkaClient, s store.JobStore, topic string) *JobService {
	return &JobService{kafka: kafka, store: s, topic: topic}
}

// Enqueue assigns an ID, persists the job, publishes it to Kafka, and returns the ID.
func (s *JobService) Enqueue(ctx context.Context, job models.Job) (string, error) {
	if job.ID == "" {
		id, err := newJobID()
		if err != nil {
			return "", fmt.Errorf("service: generate job id: %w", err)
		}
		job.ID = id
	}
	if job.State == "" {
		job.State = models.JobStatePending
	}
	now := time.Now().UTC()
	if job.ScheduledAt == nil {
		job.ScheduledAt = &now
	}

	if err := s.store.CreateJob(ctx, job); err != nil {
		return "", fmt.Errorf("service: persist job: %w", err)
	}

	// Publish to Kafka asynchronously: Postgres is the durable source of truth.
	// If Kafka delivery fails the job remains PENDING and can be reconciled by the scheduler.
	go func() {
		if err := s.kafka.Publish(context.Background(), s.topic, job); err != nil {
			_ = err // caller already has the job ID; log via metrics in a production logger
		}
	}()

	return job.ID, nil
}

// Cancel marks the job as dead in the store.
func (s *JobService) Cancel(ctx context.Context, id string) error {
	if err := s.store.CancelJob(ctx, id); err != nil {
		return fmt.Errorf("service: cancel job %s: %w", id, err)
	}
	return nil
}

func newJobID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// UUID v4 encoding
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:]), nil
}
