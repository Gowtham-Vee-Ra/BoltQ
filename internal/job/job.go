package job

import (
	"encoding/json"
	"time"
)

// Status represents the current state of a job
type Status string

const (
	StatusPending    Status = "pending"
	StatusRunning    Status = "running"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
	StatusRetrying   Status = "retrying"
	StatusDeadLetter Status = "dead_letter"
)

// Job represents a task to be processed by workers
type Job struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Payload     map[string]interface{} `json:"payload"`
	Status      Status                 `json:"status"`
	Attempts    int                    `json:"attempts"`
	MaxAttempts int                    `json:"max_attempts"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Error       string                 `json:"error,omitempty"`
}

// NewJob creates a new job with default values
func NewJob(jobType string, payload map[string]interface{}) *Job {
	now := time.Now()
	return &Job{
		ID:          generateID(),
		Type:        jobType,
		Payload:     payload,
		Status:      StatusPending,
		Attempts:    0,
		MaxAttempts: 3, // Default to 3 attempts
		CreatedAt:   now,
		UpdatedAt:   now,
	}
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
func FromJSON(data string) (*Job, error) {
	job := &Job{}
	err := json.Unmarshal([]byte(data), job)
	if err != nil {
		return nil, err
	}
	return job, nil
}

// ShouldRetry determines if a job should be retried
func (j *Job) ShouldRetry() bool {
	return j.Status == StatusFailed && j.Attempts < j.MaxAttempts
}

// IncrementAttempts increments the attempts counter and updates status
func (j *Job) IncrementAttempts() {
	j.Attempts++
	j.UpdatedAt = time.Now()

	if j.Attempts >= j.MaxAttempts {
		j.Status = StatusDeadLetter
	} else {
		j.Status = StatusRetrying
	}
}

// MarkRunning marks the job as running
func (j *Job) MarkRunning() {
	j.Status = StatusRunning
	j.UpdatedAt = time.Now()
}

// MarkCompleted marks the job as completed
func (j *Job) MarkCompleted() {
	j.Status = StatusCompleted
	j.UpdatedAt = time.Now()
}

// MarkFailed marks the job as failed with an error message
func (j *Job) MarkFailed(errMsg string) {
	j.Status = StatusFailed
	j.Error = errMsg
	j.UpdatedAt = time.Now()
}

// Helper function to generate a unique ID (simplified version)
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// Helper function to generate a random string
func randomString(length int) string {
	// Simple implementation for now
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond) // Ensure uniqueness
	}
	return string(result)
}
