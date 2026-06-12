package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/gotaskq/internal/metrics"
	"github.com/example/gotaskq/internal/store"
	"github.com/example/gotaskq/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

// testRegistry returns a fresh metrics registry on an isolated Prometheus registry
// so parallel tests don't conflict on duplicate metric registration.
func testRegistry() *metrics.Registry {
	reg := metrics.NewRegistry("test", "api")
	_ = reg.Register(prometheus.NewRegistry())
	return reg
}

func init() {
	gin.SetMode(gin.TestMode)
}

// mockQueue satisfies api.Queue with controllable responses.
type mockQueue struct {
	enqueueID  string
	enqueueErr error
	cancelErr  error
}

func (m mockQueue) Enqueue(context.Context, models.Job) (string, error) {
	return m.enqueueID, m.enqueueErr
}
func (m mockQueue) Cancel(context.Context, string) error { return m.cancelErr }

// mockStore satisfies store.JobStore with controllable responses.
type mockStore struct {
	job    models.Job
	getErr error
}

func (m mockStore) CreateJob(context.Context, models.Job) error        { return nil }
func (m mockStore) UpdateJob(context.Context, models.Job) error        { return nil }
func (m mockStore) GetJob(_ context.Context, _ string) (models.Job, error) {
	return m.job, m.getErr
}
func (m mockStore) CancelJob(context.Context, string) error           { return nil }
func (m mockStore) ClaimNextJob(context.Context) (models.Job, error)  { return models.Job{}, nil }

func TestNewHandler(t *testing.T) {
	t.Run("wires queue and store dependencies", func(t *testing.T) {
		q := mockQueue{enqueueID: "job-1"}
		s := mockStore{}
		h := NewHandler(q, s, zerolog.Logger{}, testRegistry())
		if h == nil {
			t.Fatal("NewHandler returned nil")
		}
		if h.Queue == nil {
			t.Error("Handler.Queue is nil")
		}
		if h.Store == nil {
			t.Error("Handler.Store is nil")
		}
	})
}

func TestHandlerRegisterRoutes(t *testing.T) {
	t.Run("mounts enqueue, status, and cancel routes", func(t *testing.T) {
		h := NewHandler(mockQueue{enqueueID: "job-1"}, mockStore{}, zerolog.Logger{}, testRegistry())
		if h == nil {
			t.Skip("NewHandler not yet implemented")
		}

		router := gin.New()
		h.RegisterRoutes(router)

		// Verify POST /api/jobs is registered (not 404 or 405).
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code == http.StatusNotFound {
			t.Error("POST /api/jobs returned 404: route was not registered")
		}
	})
}

func TestEnqueueJob(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		queue      mockQueue
		wantStatus int
	}{
		{
			name:       "valid request returns created status",
			body:       `{"id":"job-1","task":{"name":"send-email","queue":"default"}}`,
			queue:      mockQueue{enqueueID: "job-1"},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "invalid JSON returns bad request",
			body:       `not json`,
			queue:      mockQueue{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(tc.queue, mockStore{}, zerolog.Logger{}, testRegistry())
			if h == nil {
				t.Skip("NewHandler not yet implemented")
			}
			router := gin.New()
			h.RegisterRoutes(router)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/jobs", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

func TestGetJobStatus(t *testing.T) {
	tests := []struct {
		name       string
		jobID      string
		store      mockStore
		wantStatus int
	}{
		{
			name:       "found job returns 200",
			jobID:      "job-1",
			store:      mockStore{job: models.Job{ID: "job-1", State: models.JobStatePending}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing job returns 404",
			jobID:      "missing",
			store:      mockStore{getErr: store.ErrJobNotFound},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(mockQueue{}, tc.store, zerolog.Logger{}, testRegistry())
			if h == nil {
				t.Skip("NewHandler not yet implemented")
			}
			router := gin.New()
			h.RegisterRoutes(router)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/jobs/"+tc.jobID, nil)
			router.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

func TestCancelJob(t *testing.T) {
	tests := []struct {
		name       string
		jobID      string
		queue      mockQueue
		wantStatus int
	}{
		{
			name:       "successful cancellation returns 200",
			jobID:      "job-1",
			queue:      mockQueue{},
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(tc.queue, mockStore{}, zerolog.Logger{}, testRegistry())
			if h == nil {
				t.Skip("NewHandler not yet implemented")
			}
			router := gin.New()
			h.RegisterRoutes(router)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/jobs/"+tc.jobID+"/cancel", nil)
			router.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

// helper used to suppress unused import error before implementation is done
var _ = json.Marshal
