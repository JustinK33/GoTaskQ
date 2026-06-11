package api

import (
	"context"

	"github.com/example/gotaskq/internal/store"
	"github.com/example/gotaskq/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Queue is the business-level adapter over the transport layer.
// KafkaClient (queue package) implements Producer/Consumer but does NOT satisfy this interface.
// You will need to create a service type that wraps KafkaClient and PostgresStore to implement:
//   - Enqueue: publish the job to Kafka AND persist it in the store, returning the job ID.
//   - Cancel: apply the cancel transition in the store (and optionally signal the worker).
type Queue interface {
	Enqueue(context.Context, models.Job) (string, error)
	Cancel(context.Context, string) error
}

// Handler wires the Gin routes to the queue and store contracts.
// The logger is included so request-scoped fields can be attached consistently.
type Handler struct {
	Queue  Queue
	Store  store.JobStore
	Logger zerolog.Logger
}

// NewHandler creates the API handler bundle.
// Inputs and outputs:
// - queue handles enqueue and cancel operations.
// - jobs provides durable job lookup and state mutation.
// - logger provides structured request logging.
// - returns the handler bundle.
// Key implementation steps:
// 1. Capture dependencies.
// 2. Prepare the route handlers.
// 3. Return the handler bundle for router registration.
// Gotchas:
// - The API should not reach directly into transport-specific concerns.
func NewHandler(queue Queue, jobs store.JobStore, logger zerolog.Logger) (handler *Handler) {
	// Implementation intentionally omitted.
	return
}

// RegisterRoutes mounts the enqueue, status, and cancel endpoints on the provided router.
// Inputs and outputs:
// - router is the Gin router or route group receiving the endpoints.
// - returns nothing.
// Key implementation steps:
// 1. Group routes under the API prefix.
// 2. Register the enqueue, status, and cancel handlers.
// 3. Keep the route layout stable for clients.
// Gotchas:
// - Route shapes should remain versionable without breaking old clients.
func (h *Handler) RegisterRoutes(router gin.IRouter) {
	// Implementation intentionally omitted.
}

// EnqueueJob accepts a job creation request and forwards it to the queue.
// Inputs and outputs:
// - c is the Gin request context.
// - returns nothing; the implementation will write the HTTP response directly.
// Key implementation steps:
// 1. Bind and validate the request body.
// 2. Convert the request into a models.Job.
// 3. Enqueue the job.
// 4. Return the job identifier or validation error.
// Gotchas:
// - Request IDs and idempotency keys should be preserved when available.
func (h *Handler) EnqueueJob(c *gin.Context) {
	// Implementation intentionally omitted.
}

// GetJobStatus returns the current durable status for a job.
// Inputs and outputs:
// - c is the Gin request context.
// - returns nothing; the implementation will write the HTTP response directly.
// Key implementation steps:
// 1. Extract the job identifier from the path.
// 2. Query the store.
// 3. Serialize the job state.
// 4. Return the response payload.
// Gotchas:
// - Missing jobs should return a clean 404 rather than a generic failure.
func (h *Handler) GetJobStatus(c *gin.Context) {
	// Implementation intentionally omitted.
}

// CancelJob requests cancellation for a job.
// Inputs and outputs:
// - c is the Gin request context.
// - returns nothing; the implementation will write the HTTP response directly.
// Key implementation steps:
// 1. Extract the job identifier from the path.
// 2. Ask the queue and/or store to cancel the job.
// 3. Return the final cancellation status.
// 4. Surface conflicts and already-terminal states cleanly.
// Gotchas:
// - Cancellation semantics differ for pending, running, and already completed jobs.
func (h *Handler) CancelJob(c *gin.Context) {
	// Implementation intentionally omitted.
}
