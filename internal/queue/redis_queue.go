// internal/queue/redis_queue.go
package queue

import (
	"context"
	"fmt"
	"time"

	"BoltQ/internal/job"

	"github.com/go-redis/redis/v8"
)

const (
	// Queue keys
	mainQueueKey  = "boltq:queue:tasks"
	retryQueueKey = "boltq:queue:retry"
	deadLetterKey = "boltq:queue:deadletter"

	// Status key prefix
	statusKeyPrefix = "boltq:job:status:"

	// Stats keys
	statsProcessedKey = "boltq:stats:processed"
	statsFailedKey    = "boltq:stats:failed"
	statsRetryKey     = "boltq:stats:retry"
	statsDeadKey      = "boltq:stats:dead"

	// Lock keys
	lockKeyPrefix = "boltq:lock:job:"

	// Default TTL for job status (7 days)
	defaultStatusTTL = 7 * 24 * time.Hour
)

// RedisQueue implements a job queue using Redis
type RedisQueue struct {
	client    *redis.Client
	ctx       context.Context
	statusTTL time.Duration
}

// NewRedisQueue creates a new Redis-backed queue
func NewRedisQueue(addr, password string, db int) *RedisQueue {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		PoolSize: 10, // Connection pool size
	})

	return &RedisQueue{
		client:    client,
		ctx:       context.Background(),
		statusTTL: defaultStatusTTL,
	}
}

// Publish adds a job to the main queue
func (q *RedisQueue) Publish(task string) error {
	err := q.client.LPush(q.ctx, mainQueueKey, task).Err()
	if err != nil {
		return fmt.Errorf("failed to publish job: %v", err)
	}

	// Increment total published count
	q.client.Incr(q.ctx, "boltq:stats:published")
	return nil
}

// Consume fetches a job from the main queue
func (q *RedisQueue) Consume() (string, error) {
	task, err := q.client.RPop(q.ctx, mainQueueKey).Result()
	if err == redis.Nil {
		return "", nil // No tasks available
	}
	return task, err
}

// ConsumeJob fetches and parses a job from the queue
func (q *RedisQueue) ConsumeJob() (*job.Job, error) {
	// First try the retry queue
	taskStr, err := q.ConsumeRetry()
	if err != nil {
		return nil, err
	}

	// If no tasks in retry queue, try the main queue
	if taskStr == "" {
		taskStr, err = q.Consume()
		if err != nil {
			return nil, err
		}

		// No tasks available
		if taskStr == "" {
			return nil, nil
		}
	}

	// Parse the job
	j, err := job.FromJSON(taskStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse job JSON: %v", err)
	}

	return j, nil
}

// ConsumeRetry fetches a job from the retry queue
func (q *RedisQueue) ConsumeRetry() (string, error) {
	task, err := q.client.RPop(q.ctx, retryQueueKey).Result()
	if err == redis.Nil {
		return "", nil // No retry tasks available
	}
	return task, err
}

// PublishToRetry adds a job to the retry queue
func (q *RedisQueue) PublishToRetry(task string) error {
	err := q.client.LPush(q.ctx, retryQueueKey, task).Err()
	if err != nil {
		return fmt.Errorf("failed to publish to retry queue: %v", err)
	}

	// Increment retry count
	q.client.Incr(q.ctx, statsRetryKey)
	return nil
}

// PublishToDeadLetter adds a job to the dead letter queue
func (q *RedisQueue) PublishToDeadLetter(task string) error {
	err := q.client.LPush(q.ctx, deadLetterKey, task).Err()
	if err != nil {
		return fmt.Errorf("failed to publish to dead letter queue: %v", err)
	}

	// Increment dead letter count
	q.client.Incr(q.ctx, statsDeadKey)
	return nil
}

// SaveJobStatus stores the job status in Redis
func (q *RedisQueue) SaveJobStatus(jobID, jobJSON string) error {
	statusKey := statusKeyPrefix + jobID

	// Set the status with TTL
	err := q.client.Set(q.ctx, statusKey, jobJSON, q.statusTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to save job status: %v", err)
	}

	return nil
}

// GetJobStatus retrieves the job status from Redis
func (q *RedisQueue) GetJobStatus(jobID string) (string, error) {
	statusKey := statusKeyPrefix + jobID
	result, err := q.client.Get(q.ctx, statusKey).Result()

	if err == redis.Nil {
		return "", fmt.Errorf("job not found")
	}

	if err != nil {
		return "", fmt.Errorf("failed to get job status: %v", err)
	}

	return result, nil
}

// UpdateJobStatus updates a job's status in Redis
func (q *RedisQueue) UpdateJobStatus(j *job.Job) error {
	// Convert job to JSON
	jobJSON, err := j.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize job: %v", err)
	}

	// Save status
	return q.SaveJobStatus(j.ID, jobJSON)
}

// RetryJob handles job retry logic
func (q *RedisQueue) RetryJob(j *job.Job) error {
	// If job should be retried
	if j.ShouldRetry() {
		// Mark for retry with backoff
		j.MarkRetrying()

		// Update status
		if err := q.UpdateJobStatus(j); err != nil {
			return err
		}

		// Convert to JSON
		jobJSON, err := j.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize job for retry: %v", err)
		}

		// Add to retry queue
		return q.PublishToRetry(jobJSON)
	} else {
		// Move to dead letter queue
		j.MarkDeadLetter()

		// Update status
		if err := q.UpdateJobStatus(j); err != nil {
			return err
		}

		// Convert to JSON
		jobJSON, err := j.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize job for dead letter: %v", err)
		}

		// Add to dead letter queue
		return q.PublishToDeadLetter(jobJSON)
	}
}

// GetQueueStats returns statistics about the queues
func (q *RedisQueue) GetQueueStats() (map[string]int64, error) {
	stats := make(map[string]int64)

	// Get queue lengths
	mainQueueLen, err := q.client.LLen(q.ctx, mainQueueKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get main queue length: %v", err)
	}
	stats["pending"] = mainQueueLen

	retryQueueLen, err := q.client.LLen(q.ctx, retryQueueKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get retry queue length: %v", err)
	}
	stats["retry"] = retryQueueLen

	deadLetterLen, err := q.client.LLen(q.ctx, deadLetterKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get dead letter queue length: %v", err)
	}
	stats["dead_letter"] = deadLetterLen

	// Get counters
	processed, _ := q.client.Get(q.ctx, statsProcessedKey).Int64()
	stats["processed"] = processed

	failed, _ := q.client.Get(q.ctx, statsFailedKey).Int64()
	stats["failed"] = failed

	published, _ := q.client.Get(q.ctx, "boltq:stats:published").Int64()
	stats["published"] = published

	return stats, nil
}

// AcquireJobLock attempts to acquire a distributed lock for a job
func (q *RedisQueue) AcquireJobLock(jobID, workerID string) (bool, error) {
	lockKey := lockKeyPrefix + jobID

	// Use SET NX to ensure atomicity
	success, err := q.client.SetNX(q.ctx, lockKey, workerID, 30*time.Second).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %v", err)
	}

	return success, nil
}

// ReleaseJobLock releases a previously acquired lock
func (q *RedisQueue) ReleaseJobLock(jobID, workerID string) (bool, error) {
	lockKey := lockKeyPrefix + jobID

	// Check if we still own the lock
	val, err := q.client.Get(q.ctx, lockKey).Result()
	if err == redis.Nil {
		// Lock doesn't exist
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("failed to check lock ownership: %v", err)
	}

	// Only delete if we own the lock
	if val == workerID {
		deleted, err := q.client.Del(q.ctx, lockKey).Result()
		if err != nil {
			return false, fmt.Errorf("failed to release lock: %v", err)
		}
		return deleted > 0, nil
	}

	return false, nil
}

// Close closes the Redis connection
func (q *RedisQueue) Close() error {
	return q.client.Close()
}
