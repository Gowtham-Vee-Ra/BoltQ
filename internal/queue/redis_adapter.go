// internal/queue/redis_adapter.go
package queue

import (
	"context"
	"time"
)

// RedisQueueAdapter adapts RedisQueue to implement the Queue interface
type RedisQueueAdapter struct {
	redisQueue *RedisQueue
}

// NewRedisQueueAdapter creates a new adapter for RedisQueue
func NewRedisQueueAdapter(redisQueue *RedisQueue) Queue {
	return &RedisQueueAdapter{
		redisQueue: redisQueue,
	}
}

// Publish adds a job to the queue with specified priority
func (a *RedisQueueAdapter) Publish(ctx context.Context, job *Job) error {
	// Convert Job to Task
	task := &Task{
		ID:          job.ID,
		Type:        job.Type,
		Data:        job.Payload,
		Priority:    job.Priority,
		CreatedAt:   job.CreatedAt,
		ScheduledAt: job.ScheduledAt,
		Status:      string(job.Status),
		Attempts:    job.Attempts,
		LastError:   job.Error,
	}

	return a.redisQueue.Publish(task)
}

// PublishDelayed adds a job to be executed at a future time
func (a *RedisQueueAdapter) PublishDelayed(ctx context.Context, job *Job, delay time.Duration) error {
	// Convert Job to Task
	task := &Task{
		ID:          job.ID,
		Type:        job.Type,
		Data:        job.Payload,
		Priority:    job.Priority,
		CreatedAt:   job.CreatedAt,
		ScheduledAt: job.ScheduledAt,
		Status:      string(job.Status),
		Attempts:    job.Attempts,
		LastError:   job.Error,
	}

	delaySeconds := int(delay.Seconds())
	return a.redisQueue.PublishDelayed(task, delaySeconds)
}

// Consume retrieves the next available job from the queue
func (a *RedisQueueAdapter) Consume(ctx context.Context) (*Job, error) {
	task, err := a.redisQueue.Consume()
	if err != nil {
		return nil, err
	}

	// Convert Task to Job
	job := &Job{
		ID:          task.ID,
		Type:        task.Type,
		Payload:     task.Data,
		Priority:    task.Priority,
		ScheduledAt: task.ScheduledAt,
		Status:      JobStatus(task.Status),
		Attempts:    task.Attempts,
		Error:       task.LastError,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   time.Now(),
	}

	return job, nil
}

// UpdateStatus updates a job's status
func (a *RedisQueueAdapter) UpdateStatus(ctx context.Context, jobID string, status JobStatus, err error) error {
	// Get current task
	task, getErr := a.redisQueue.GetTaskStatus(jobID)
	if getErr != nil {
		return getErr
	}

	// Update status
	task.Status = string(status)
	if err != nil {
		task.LastError = err.Error()
	}

	return a.redisQueue.UpdateStatus(task)
}

// GetJob retrieves a job by ID
func (a *RedisQueueAdapter) GetJob(ctx context.Context, jobID string) (*Job, error) {
	task, err := a.redisQueue.GetTaskStatus(jobID)
	if err != nil {
		return nil, err
	}

	// Convert Task to Job
	job := &Job{
		ID:          task.ID,
		Type:        task.Type,
		Payload:     task.Data,
		Priority:    task.Priority,
		ScheduledAt: task.ScheduledAt,
		Status:      JobStatus(task.Status),
		Attempts:    task.Attempts,
		Error:       task.LastError,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   time.Now(),
	}

	return job, nil
}

// GetStats returns statistics about the queue
func (a *RedisQueueAdapter) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return a.redisQueue.GetQueueStats()
}

// Close closes the queue connection
func (a *RedisQueueAdapter) Close() error {
	return a.redisQueue.Close()
}
