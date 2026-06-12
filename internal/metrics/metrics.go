package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var _ = promhttp.Handler

// Registry groups the Prometheus collectors exposed by GoTaskQ.
type Registry struct {
	Namespace      string
	Subsystem      string
	JobEnqueued    prometheus.Counter
	JobStarted     prometheus.Counter
	JobCompleted   prometheus.Counter
	JobFailed      prometheus.Counter
	JobCancelled   prometheus.Counter
	WorkerInFlight prometheus.Gauge
	JobDuration    prometheus.Histogram
	HTTPRequests   *prometheus.CounterVec
}

// NewRegistry constructs all collectors for the given namespace and subsystem.
func NewRegistry(namespace, subsystem string) *Registry {
	r := &Registry{
		Namespace: namespace,
		Subsystem: subsystem,
	}

	r.JobEnqueued = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "jobs_enqueued_total",
		Help:      "Total number of jobs enqueued.",
	})
	r.JobStarted = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "jobs_started_total",
		Help:      "Total number of jobs started.",
	})
	r.JobCompleted = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "jobs_completed_total",
		Help:      "Total number of jobs completed successfully.",
	})
	r.JobFailed = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "jobs_failed_total",
		Help:      "Total number of jobs that failed.",
	})
	r.JobCancelled = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "jobs_cancelled_total",
		Help:      "Total number of jobs cancelled.",
	})
	r.WorkerInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "worker_in_flight",
		Help:      "Number of jobs currently being processed.",
	})
	r.JobDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "job_duration_seconds",
		Help:      "Execution duration of jobs in seconds.",
		Buckets:   prometheus.DefBuckets,
	})
	r.HTTPRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "http_requests_total",
		Help:      "Total HTTP requests partitioned by method, path, and status.",
	}, []string{"method", "path", "status"})

	return r
}

// Register binds all collectors to the supplied Prometheus registerer.
func (r *Registry) Register(registerer prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		r.JobEnqueued,
		r.JobStarted,
		r.JobCompleted,
		r.JobFailed,
		r.JobCancelled,
		r.WorkerInFlight,
		r.JobDuration,
		r.HTTPRequests,
	}
	for _, c := range collectors {
		if err := registerer.Register(c); err != nil {
			return err
		}
	}
	return nil
}

// Handler returns the Prometheus scrape endpoint handler.
func (r *Registry) Handler() http.Handler {
	return promhttp.Handler()
}
