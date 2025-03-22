// internal/job/job.go
package job

import (
	"encoding/json"
	"fmt"
	"time"
)

// Job statuses
const (
	StatusPending    = "pending"
	StatusRunning    = "running"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusRetrying   = "retrying"
	StatusDeadLetter = "dead_letter"
	StatusCancelled  = "cancelled"
)

// Priority levels
const (
	PriorityLow      = 1
	PriorityNormal   = 2
	PriorityHigh     = 3
	PriorityCritical = 4
)

// DefaultMaxAttempts is the default number of retry attempts
const DefaultMaxAttempts = 3

// Job represents a task to be processed
type Job struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	Payload     string      `json:"payload"`
	Status      string      `json:"status"`
	Priority    int         `json:"priority"`
	CreatedAt   time.Time   `json:"created_at"`
	StartedAt   time.Time   `json:"started_at,omitempty"`
	CompletedAt time.Time   `json:"completed_at,omitempty"`
	NextRetryAt time.Time   `json:"next_retry_at,omitempty"`
	Attempts    int         `json:"attempts"`
	MaxAttempts int         `json:"max_attempts"`
	Error       string      `json:"error,omitempty"`
	WorkerID    string      `json:"worker_id,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	Result      interface{} `json:"result,omitempty"`
	Timeout     int         `json:"timeout,omitempty"` // Timeout in seconds
}

// NewJob creates a new job with default values
func NewJob(jobType, payload string) *Job {
	return &Job{
		ID:          generateID(),
		Type:        jobType,
		Payload:     payload,
		Status:      StatusPending,
		Priority:    PriorityNormal,
		CreatedAt:   time.Now(),
		Attempts:    0,
		MaxAttempts: DefaultMaxAttempts,
	}
}

// SetPriority sets the job priority
func (j *Job) SetPriority(priority int) *Job {
	if priority < PriorityLow || priority > PriorityCritical {
		j.Priority = PriorityNormal
	} else {
		j.Priority = priority
	}
	return j
}

// SetMaxAttempts sets the maximum number of retry attempts
func (j *Job) SetMaxAttempts(attempts int) *Job {
	if attempts < 1 {
		j.MaxAttempts = 1
	} else {
		j.MaxAttempts = attempts
	}
	return j
}

// SetTimeout sets the job timeout in seconds
func (j *Job) SetTimeout(seconds int) *Job {
	if seconds < 1 {
		j.Timeout = 60 // Default 1 minute
	} else {
		j.Timeout = seconds
	}
	return j
}

// AddTag adds a tag to the job
func (j *Job) AddTag(tag string) *Job {
	j.Tags = append(j.Tags, tag)
	return j
}

// MarkRunning updates the job status to running
func (j *Job) MarkRunning() {
	j.Status = StatusRunning
	j.StartedAt = time.Now()
}

// MarkCompleted updates the job status to completed
func (j *Job) MarkCompleted() {
	j.Status = StatusCompleted
	j.CompletedAt = time.Now()
}

// MarkFailed updates the job status to failed and increments attempts
func (j *Job) MarkFailed(errMsg string) {
	j.Status = StatusFailed
	j.Error = errMsg
	j.Attempts++
}

// MarkRetrying updates the job status for retry
func (j *Job) MarkRetrying() {
	j.Status = StatusRetrying
	j.NextRetryAt = time.Now().Add(j.CalculateBackoff())
}

// MarkDeadLetter updates the job status to dead letter
func (j *Job) MarkDeadLetter() {
	j.Status = StatusDeadLetter
}

// ShouldRetry determines if a job should be retried based on its current state
func (j *Job) ShouldRetry() bool {
	return j.Status == StatusFailed && j.Attempts < j.MaxAttempts
}

// CalculateBackoff calculates the backoff duration for retries
func (j *Job) CalculateBackoff() time.Duration {
	// Exponential backoff: 2^attempts seconds (1, 2, 4, 8, 16...)
	return time.Second * time.Duration(1<<j.Attempts)
}

// IsExpired checks if a job has exceeded its timeout
func (j *Job) IsExpired() bool {
	if j.Timeout == 0 || j.StartedAt.IsZero() {
		return false
	}

	deadline := j.StartedAt.Add(time.Duration(j.Timeout) * time.Second)
	return time.Now().After(deadline)
}

// ToJSON serializes the job to JSON
func (j *Job) ToJSON() (string, error) {
	bytes, err := json.Marshal(j)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// FromJSON deserializes a job from JSON
func FromJSON(jsonStr string) (*Job, error) {
	var job Job
	err := json.Unmarshal([]byte(jsonStr), &job)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// generateID creates a unique ID for a job
// In production, consider using UUID or similar
func generateID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}
