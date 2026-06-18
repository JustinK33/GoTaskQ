package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

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

func NewHandler(queue Queue, jobs store.JobStore, logger zerolog.Logger, reg *metrics.Registry) *Handler {
	return &Handler{Queue: queue, Store: jobs, Logger: logger, Metrics: reg}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.Use(RequestID(h.Logger), RequestLogger(), Recovery(), h.metricsMiddleware())
	g := router.Group("/api/jobs")
	g.POST("", h.EnqueueJob)
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

	var job models.Job
	if err := c.ShouldBindJSON(&job); err != nil {
		RespondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if job.Task.Name == "" {
		RespondError(c, http.StatusBadRequest, "invalid_request", "task.name is required")
		return
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
