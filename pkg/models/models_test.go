package models

import "testing"

func TestJobStateConstants(t *testing.T) {
	tests := []struct {
		name  string
		state JobState
		want  string
	}{
		{name: "pending", state: JobStatePending, want: "PENDING"},
		{name: "running", state: JobStateRunning, want: "RUNNING"},
		{name: "completed", state: JobStateCompleted, want: "COMPLETED"},
		{name: "failed", state: JobStateFailed, want: "FAILED"},
		{name: "dead", state: JobStateDead, want: "DEAD"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.state) != tc.want {
				t.Errorf("JobState value = %q, want %q", tc.state, tc.want)
			}
		})
	}
}

func TestJobAndTaskShapes(t *testing.T) {
	t.Run("job timestamps are nullable pointers", func(t *testing.T) {
		j := Job{}
		if j.ScheduledAt != nil {
			t.Error("ScheduledAt should be nil on zero-value Job")
		}
		if j.StartedAt != nil {
			t.Error("StartedAt should be nil on zero-value Job")
		}
		if j.CompletedAt != nil {
			t.Error("CompletedAt should be nil on zero-value Job")
		}
	})

	t.Run("task payload round-trips through job", func(t *testing.T) {
		payload := []byte(`{"email":"user@example.com"}`)
		j := Job{
			Task:  Task{Name: "send-email", Payload: payload, MaxRetries: 3, Queue: "default"},
			State: JobStatePending,
		}

		if j.Task.Name != "send-email" {
			t.Errorf("Task.Name = %q, want %q", j.Task.Name, "send-email")
		}
		if string(j.Task.Payload) != string(payload) {
			t.Errorf("Task.Payload = %q, want %q", j.Task.Payload, payload)
		}
		if j.State != JobStatePending {
			t.Errorf("State = %q, want %q", j.State, JobStatePending)
		}
	})
}
