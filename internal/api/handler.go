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
	router.Use(RequestID(h.Logger), RequestLogger(), h.metricsMiddleware())
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
	var job models.Job
	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if job.Task.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task.name is required"})
		return
	}

	id, err := h.Queue.Enqueue(c.Request.Context(), job)
	if err != nil {
		h.Logger.Error().Err(err).Msg("enqueue failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue job"})
		return
	}

	h.Metrics.JobEnqueued.Inc()
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handler) GetJobStatus(c *gin.Context) {
	id := c.Param("id")

	job, err := h.Store.GetJob(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrJobNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		h.Logger.Error().Err(err).Str("job_id", id).Msg("get job failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve job"})
		return
	}

	c.JSON(http.StatusOK, job)
}

func (h *Handler) CancelJob(c *gin.Context) {
	id := c.Param("id")

	if err := h.Queue.Cancel(c.Request.Context(), id); err != nil {
		if errors.Is(err, store.ErrJobNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		h.Logger.Error().Err(err).Str("job_id", id).Msg("cancel failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel job"})
		return
	}

	h.Metrics.JobCancelled.Inc()
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}
