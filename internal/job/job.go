// internal/job/job.go
package job

import (
	"encoding/json"
	"math"
	"time"
)

// Status represents the status of a job
type Status string

const (
	StatusUnknown   Status = "unknown"
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusRetrying  Status = "retrying"
	StatusCancelled Status = "cancelled"
)

// Priority represents the priority of a job
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityNormal   Priority = "normal"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// Job represents a task to be processed
type Job struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Data        map[string]interface{} `json:"data"`
	Status      Status                 `json:"status"`
	Priority    Priority               `json:"priority"`
	Tags        []string               `json:"tags,omitempty"`
	Attempts    int                    `json:"attempts"`
	MaxRetries  int                    `json:"max_retries"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	StartedAt   time.Time              `json:"started_at,omitempty"`
	FinishedAt  time.Time              `json:"finished_at,omitempty"`
	LastAttempt time.Time              `json:"last_attempt,omitempty"`
	Timeout     int                    `json:"timeout"`
	TraceID     string                 `json:"trace_id,omitempty"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

func (j *Job) SetPriority(low Priority) {
	panic("unimplemented")
}

// String returns a string representation of the job
func (j *Job) String() string {
	data, _ := json.Marshal(j)
	return string(data)
}

// MarkRunning updates the job status to running
func (j *Job) MarkRunning() {
	j.Status = StatusRunning
	j.StartedAt = time.Now()
	j.UpdatedAt = time.Now()
}

// MarkCompleted updates the job status to completed
func (j *Job) MarkCompleted(result map[string]interface{}) {
	j.Status = StatusCompleted
	j.FinishedAt = time.Now()
	j.UpdatedAt = time.Now()
	j.Result = result
}

// MarkFailed updates the job status to failed
func (j *Job) MarkFailed(err error) {
	j.Status = StatusFailed
	j.FinishedAt = time.Now()
	j.UpdatedAt = time.Now()
	if err != nil {
		j.Error = err.Error()
	}
}

// MarkRetrying updates the job status to retrying
func (j *Job) MarkRetrying(err error) {
	j.Status = StatusRetrying
	j.UpdatedAt = time.Now()
	j.Attempts++
	j.LastAttempt = time.Now()
	if err != nil {
		j.Error = err.Error()
	}
}

// MarkCancelled updates the job status to cancelled
func (j *Job) MarkCancelled() {
	j.Status = StatusCancelled
	j.FinishedAt = time.Now()
	j.UpdatedAt = time.Now()
}

// GetBackoffSeconds calculates the backoff duration in seconds
func (j *Job) GetBackoffSeconds() int {
	// Exponential backoff: 2^attempts seconds
	backoff := int(math.Pow(2, float64(j.Attempts)))

	// Cap at 5 minutes
	if backoff > 300 {
		backoff = 300
	}

	return backoff
}

// IsExpired checks if the job has expired based on timeout
func (j *Job) IsExpired() bool {
	if j.Status != StatusRunning || j.StartedAt.IsZero() {
		return false
	}

	if j.Timeout <= 0 {
		return false
	}

	timeout := time.Duration(j.Timeout) * time.Second
	return time.Since(j.StartedAt) > timeout
}
