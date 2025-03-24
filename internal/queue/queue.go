package queue

import (
	"context"
	"time"
)

// Priority levels for jobs
const (
	PriorityLow    = 0
	PriorityNormal = 1
	PriorityHigh   = 2
)

// JobStatus represents the current state of a job
type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
	StatusRetrying  JobStatus = "retrying"
	StatusCancelled JobStatus = "cancelled"
)

type Logger interface {
	Info(msg string, fields ...map[string]interface{})
	Error(msg string, fields ...map[string]interface{})
	Debug(msg string, fields ...map[string]interface{})
}

// Job represents a task to be processed by workers
type Job struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Payload     map[string]interface{} `json:"payload"`
	Priority    int                    `json:"priority"`
	ScheduledAt time.Time              `json:"scheduled_at,omitempty"`
	Status      JobStatus              `json:"status"`
	Attempts    int                    `json:"attempts"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// QueueFactory is an interface for creating queues
type QueueFactory interface {
	// CreateQueue creates a new queue with the provided configuration
	CreateQueue(config map[string]string) (Queue, error)

	Close() error
}

// Queue defines the interface for job queue implementations
type Queue interface {
	// Publish adds a job to the queue with specified priority
	Publish(ctx context.Context, job *Job) error

	// PublishDelayed adds a job to be executed at a future time
	PublishDelayed(ctx context.Context, job *Job, delay time.Duration) error

	// Consume retrieves the next available job from the queue
	Consume(ctx context.Context) (*Job, error)

	// UpdateStatus updates a job's status
	UpdateStatus(ctx context.Context, jobID string, status JobStatus, err error) error

	// GetJob retrieves a job by ID
	GetJob(ctx context.Context, jobID string) (*Job, error)

	// GetStats returns statistics about the queue
	GetStats(ctx context.Context) (map[string]interface{}, error)

	// Close closes the queue connection
	Close() error
}

// QueueFactory creates a new queue implementation
