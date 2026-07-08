package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/example/gotaskq/internal/metrics"
	"github.com/example/gotaskq/internal/store"
	"github.com/example/gotaskq/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Queue is the business-level adapter exposed to the HTTP layer.
type Queue interface {
	Enqueue(context.Context, models.Job) (string, error)
	Cancel(context.Context, string) error
}

type Handler struct {
	Queue   Queue
	Store   store.JobStore
	Logger  zerolog.Logger
	Metrics *metrics.Registry
}

type EnqueueRequest struct {
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
	Task           models.Task       `json:"task"`
	ScheduledAt    *time.Time        `json:"scheduled_at,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

func NewHandler(queue Queue, jobs store.JobStore, logger zerolog.Logger, reg *metrics.Registry) *Handler {
	return &Handler{Queue: queue, Store: jobs, Logger: logger, Metrics: reg}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.Use(RequestID(h.Logger), RequestLogger(), Recovery(), h.metricsMiddleware())
	g := router.Group("/api/jobs")
	g.POST("", h.EnqueueJob)
	g.GET("", h.ListJobs)
	g.GET("/by-idempotency-key/:key", h.GetJobByIdempotencyKey)
	g.GET("/:id", h.GetJobStatus)
	g.POST("/:id/cancel", h.CancelJob)
}

func (h *Handler) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		h.Metrics.HTTPRequests.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			strconv.Itoa(c.Writer.Status()),
		).Inc()
	}
}

func (h *Handler) EnqueueJob(c *gin.Context) {
	log := zerolog.Ctx(c.Request.Context())

	var request EnqueueRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		RespondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if request.Task.Name == "" {
		RespondError(c, http.StatusBadRequest, "invalid_request", "task.name is required")
		return
	}

	job := models.Job{
		IdempotencyKey: request.IdempotencyKey,
		Task:           request.Task,
		ScheduledAt:    request.ScheduledAt,
		Metadata:       request.Metadata,
	}

	id, err := h.Queue.Enqueue(c.Request.Context(), job)
	if err != nil {
		log.Error().Err(err).Msg("enqueue failed")
		RespondError(c, http.StatusInternalServerError, "internal_error", "failed to enqueue job")
		return
	}

	h.Metrics.JobEnqueued.Inc()
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handler) GetJobStatus(c *gin.Context) {
	log := zerolog.Ctx(c.Request.Context())
	id := c.Param("id")

	job, err := h.Store.GetJob(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "not_found", "job not found")
			return
		}
		log.Error().Err(err).Str("job_id", id).Msg("get job failed")
		RespondError(c, http.StatusInternalServerError, "internal_error", "failed to retrieve job")
		return
	}

	c.JSON(http.StatusOK, job)
}

func (h *Handler) GetJobByIdempotencyKey(c *gin.Context) {
	log := zerolog.Ctx(c.Request.Context())
	key := c.Param("key")
	if key == "" {
		RespondError(c, http.StatusBadRequest, "invalid_request", "idempotency key is required")
		return
	}

	job, err := h.Store.GetJobByIdempotencyKey(c.Request.Context(), key)
	if err != nil {
		if errors.Is(err, store.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "not_found", "job not found")
			return
		}
		log.Error().Err(err).Str("idempotency_key", key).Msg("get job by idempotency key failed")
		RespondError(c, http.StatusInternalServerError, "internal_error", "failed to retrieve job")
		return
	}

	c.JSON(http.StatusOK, job)
}

// ListJobs returns a page of jobs, optionally filtered by ?state=PENDING/RUNNING/...
// Pagination uses the opaque cursor returned in the previous page's `next_cursor`.
//
//	GET /api/jobs?state=FAILED&limit=20
//	GET /api/jobs?cursor=<token>
func (h *Handler) ListJobs(c *gin.Context) {
	log := zerolog.Ctx(c.Request.Context())

	filter := store.ListFilter{
		State:  models.JobState(c.Query("state")),
		Cursor: c.Query("cursor"),
	}
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			RespondError(c, http.StatusBadRequest, "invalid_request", "limit must be a non-negative integer")
			return
		}
		filter.Limit = n
	}

	switch filter.State {
	case "",
		models.JobStatePending, models.JobStateRunning,
		models.JobStateCompleted, models.JobStateFailed, models.JobStateDead:
	default:
		RespondError(c, http.StatusBadRequest, "invalid_request",
			"state must be one of PENDING, RUNNING, COMPLETED, FAILED, DEAD")
		return
	}

	jobs, nextCursor, err := h.Store.ListJobs(c.Request.Context(), filter)
	if err != nil {
		log.Error().Err(err).Msg("list jobs failed")
		RespondError(c, http.StatusInternalServerError, "internal_error", "failed to list jobs")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":        jobs,
		"next_cursor": nextCursor,
	})
}

func (h *Handler) CancelJob(c *gin.Context) {
	log := zerolog.Ctx(c.Request.Context())
	id := c.Param("id")

	if err := h.Queue.Cancel(c.Request.Context(), id); err != nil {
		if errors.Is(err, store.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "not_found", "job not found")
			return
		}
		if errors.Is(err, store.ErrInvalidTransition) {
			RespondError(c, http.StatusConflict, "invalid_state", "job cannot be cancelled in its current state")
			return
		}
		log.Error().Err(err).Str("job_id", id).Msg("cancel failed")
		RespondError(c, http.StatusInternalServerError, "internal_error", "failed to cancel job")
		return
	}

	h.Metrics.JobCancelled.Inc()
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}
