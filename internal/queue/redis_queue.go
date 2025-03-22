// internal/queue/redis_queue.go
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"BoltQ/internal/job"
	"BoltQ/pkg/logger"
	"BoltQ/pkg/metrics"

	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
)

// Queue interface defines methods for a job queue
type Queue interface {
	PublishJob(job *job.Job) error
	ConsumeJob() (*job.Job, error)
	UpdateJobStatus(jobID string, status job.Status) error
	RetryJob(job *job.Job) error
	MoveToDeadLetter(job *job.Job) error
	GetJobStatus(jobID string) (job.Status, error)
}

// RedisQueue implements the Queue interface using Redis
type RedisQueue struct {
	client *redis.Client
	ctx    context.Context
	logger *logger.Logger
}

// Priority queue names
const (
	queuePrefixPending    = "boltq:queue:pending"
	queuePrefixRetry      = "boltq:queue:retry"
	queuePrefixDeadLetter = "boltq:queue:deadletter"
	jobStatusPrefix       = "boltq:status:"
)

// NewRedisQueue creates a new RedisQueue
func NewRedisQueue(ctx context.Context, redisAddr string) *RedisQueue {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	log := logger.NewLogger("redis-queue")

	return &RedisQueue{
		client: client,
		ctx:    ctx,
		logger: log,
	}
}

// getQueueName returns the appropriate queue name based on priority
func getQueueName(priority job.Priority) string {
	return fmt.Sprintf("%s:%s", queuePrefixPending, priority)
}

// measureRedisOperation wraps Redis operations with metrics
func (q *RedisQueue) measureRedisOperation(operation string, fn func() error) error {
	timer := prometheus.NewTimer(metrics.RedisOperationDuration.WithLabelValues(operation))
	defer timer.ObserveDuration()

	err := fn()

	if err != nil {
		metrics.RedisOperations.WithLabelValues(operation, "error").Inc()
		return err
	}

	metrics.RedisOperations.WithLabelValues(operation, "success").Inc()
	return nil
}

// PublishJob adds a job to the appropriate priority queue
func (q *RedisQueue) PublishJob(j *job.Job) error {
	jobData, err := json.Marshal(j)
	if err != nil {
		q.logger.Error("Failed to marshal job data", map[string]interface{}{
			"job_id": j.ID,
			"error":  err.Error(),
		})
		return err
	}

	queueName := getQueueName(j.Priority)

	err = q.measureRedisOperation("publish", func() error {
		return q.client.LPush(q.ctx, queueName, jobData).Err()
	})

	if err != nil {
		q.logger.Error("Failed to publish job to Redis", map[string]interface{}{
			"job_id":   j.ID,
			"queue":    queueName,
			"error":    err.Error(),
			"priority": string(j.Priority),
		})
		return err
	}

	// Update job status
	err = q.UpdateJobStatus(j.ID, job.StatusPending)
	if err != nil {
		q.logger.Error("Failed to update job status", map[string]interface{}{
			"job_id": j.ID,
			"error":  err.Error(),
		})
		return err
	}

	metrics.JobsSubmitted.WithLabelValues(j.Type, string(j.Priority)).Inc()
	metrics.JobsInQueue.WithLabelValues("pending", string(j.Priority)).Inc()

	q.logger.WithJob(j.ID, "Job published to queue", map[string]interface{}{
		"type":     j.Type,
		"priority": string(j.Priority),
	})

	return nil
}

// ConsumeJob retrieves a job from the queues in priority order
func (q *RedisQueue) ConsumeJob() (*job.Job, error) {
	// Try to consume from each priority queue in order
	priorities := []job.Priority{
		job.PriorityCritical,
		job.PriorityHigh,
		job.PriorityNormal,
		job.PriorityLow,
	}

	// Also check retry queue
	queues := []string{queuePrefixRetry}
	for _, p := range priorities {
		queues = append(queues, getQueueName(p))
	}

	var jobData string
	var sourceQueue string

	err := q.measureRedisOperation("consume", func() error {
		// Use BRPOP to block until a job is available
		result, err := q.client.BRPop(q.ctx, 5*time.Second, queues...).Result()
		if err != nil {
			if err == redis.Nil {
				// No jobs available, not an error
				return nil
			}
			return err
		}

		sourceQueue = result[0]
		jobData = result[1]
		return nil
	})

	if err != nil {
		q.logger.Error("Failed to consume job from Redis", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	if jobData == "" {
		// No job found
		return nil, nil
	}

	var j job.Job
	err = json.Unmarshal([]byte(jobData), &j)
	if err != nil {
		q.logger.Error("Failed to unmarshal job data", map[string]interface{}{
			"error": err.Error(),
			"data":  jobData,
		})
		return nil, err
	}

	// Update job status to running
	err = q.UpdateJobStatus(j.ID, job.StatusRunning)
	if err != nil {
		q.logger.Error("Failed to update job status", map[string]interface{}{
			"job_id": j.ID,
			"error":  err.Error(),
		})
		// Still return the job even if status update fails
	}

	// Update queue metrics
	queueType := "pending"
	if sourceQueue == queuePrefixRetry {
		queueType = "retry"
	}
	metrics.JobsInQueue.WithLabelValues(queueType, string(j.Priority)).Dec()

	q.logger.WithJob(j.ID, "Job consumed from queue", map[string]interface{}{
		"type":     j.Type,
		"priority": string(j.Priority),
		"queue":    sourceQueue,
	})

	return &j, nil
}

// UpdateJobStatus updates the job status in Redis
func (q *RedisQueue) UpdateJobStatus(jobID string, status job.Status) error {
	statusKey := fmt.Sprintf("%s%s", jobStatusPrefix, jobID)

	err := q.measureRedisOperation("update_status", func() error {
		return q.client.Set(q.ctx, statusKey, string(status), 7*24*time.Hour).Err()
	})

	if err != nil {
		q.logger.Error("Failed to update job status", map[string]interface{}{
			"job_id": jobID,
			"status": string(status),
			"error":  err.Error(),
		})
		return err
	}

	q.logger.WithJob(jobID, "Job status updated", map[string]interface{}{
		"status": string(status),
	})

	if status == job.StatusCompleted || status == job.StatusFailed {
		metrics.JobsProcessed.WithLabelValues("", string(status)).Inc()
	}

	return nil
}

// RetryJob moves a job to the retry queue
func (q *RedisQueue) RetryJob(j *job.Job) error {
	// Increment attempts
	j.Attempts++
	j.LastAttempt = time.Now()

	jobData, err := json.Marshal(j)
	if err != nil {
		q.logger.Error("Failed to marshal job for retry", map[string]interface{}{
			"job_id": j.ID,
			"error":  err.Error(),
		})
		return err
	}

	err = q.measureRedisOperation("retry", func() error {
		return q.client.LPush(q.ctx, queuePrefixRetry, jobData).Err()
	})

	if err != nil {
		q.logger.Error("Failed to move job to retry queue", map[string]interface{}{
			"job_id": j.ID,
			"error":  err.Error(),
		})
		return err
	}

	// Update job status
	err = q.UpdateJobStatus(j.ID, job.StatusRetrying)
	if err != nil {
		q.logger.Error("Failed to update job status for retry", map[string]interface{}{
			"job_id": j.ID,
			"error":  err.Error(),
		})
		return err
	}

	metrics.JobsInQueue.WithLabelValues("retry", string(j.Priority)).Inc()

	q.logger.WithJob(j.ID, "Job moved to retry queue", map[string]interface{}{
		"attempts": j.Attempts,
		"backoff":  j.GetBackoffSeconds(),
	})

	return nil
}

// MoveToDeadLetter moves a job to the dead letter queue
func (q *RedisQueue) MoveToDeadLetter(j *job.Job) error {
	jobData, err := json.Marshal(j)
	if err != nil {
		q.logger.Error("Failed to marshal job for dead letter", map[string]interface{}{
			"job_id": j.ID,
			"error":  err.Error(),
		})
		return err
	}

	err = q.measureRedisOperation("move_to_dead_letter", func() error {
		return q.client.LPush(q.ctx, queuePrefixDeadLetter, jobData).Err()
	})

	if err != nil {
		q.logger.Error("Failed to move job to dead letter queue", map[string]interface{}{
			"job_id": j.ID,
			"error":  err.Error(),
		})
		return err
	}

	// Update job status
	err = q.UpdateJobStatus(j.ID, job.StatusFailed)
	if err != nil {
		q.logger.Error("Failed to update job status for dead letter", map[string]interface{}{
			"job_id": j.ID,
			"error":  err.Error(),
		})
		return err
	}

	metrics.JobsInQueue.WithLabelValues("dead_letter", string(j.Priority)).Inc()
	metrics.JobsProcessed.WithLabelValues(j.Type, "failed").Inc()

	q.logger.WithJob(j.ID, "Job moved to dead letter queue", map[string]interface{}{
		"attempts": j.Attempts,
		"error":    "Exceeded maximum retry attempts",
	})

	return nil
}

// GetJobStatus retrieves the current status of a job
func (q *RedisQueue) GetJobStatus(jobID string) (job.Status, error) {
	statusKey := fmt.Sprintf("%s%s", jobStatusPrefix, jobID)

	var statusStr string
	err := q.measureRedisOperation("get_status", func() error {
		var err error
		statusStr, err = q.client.Get(q.ctx, statusKey).Result()
		return err
	})

	if err != nil {
		if err == redis.Nil {
			q.logger.Warn("Job status not found", map[string]interface{}{
				"job_id": jobID,
			})
			return job.StatusUnknown, nil
		}

		q.logger.Error("Failed to get job status", map[string]interface{}{
			"job_id": jobID,
			"error":  err.Error(),
		})
		return job.StatusUnknown, err
	}

	return job.Status(statusStr), nil
}

// GetQueueStats returns statistics about the queues
func (q *RedisQueue) GetQueueStats() (map[string]int64, error) {
	priorities := []job.Priority{
		job.PriorityCritical,
		job.PriorityHigh,
		job.PriorityNormal,
		job.PriorityLow,
	}

	stats := make(map[string]int64)

	// Get counts for each priority queue
	for _, p := range priorities {
		queueName := getQueueName(p)
		count, err := q.client.LLen(q.ctx, queueName).Result()
		if err != nil {
			q.logger.Error("Failed to get queue length", map[string]interface{}{
				"queue": queueName,
				"error": err.Error(),
			})
			return nil, err
		}

		stats[string(p)] = count

		// Update metrics
		metrics.JobsInQueue.WithLabelValues("pending", string(p)).Set(float64(count))
	}

	// Get retry and dead letter queue counts
	retryCount, err := q.client.LLen(q.ctx, queuePrefixRetry).Result()
	if err != nil {
		q.logger.Error("Failed to get retry queue length", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}
	stats["retry"] = retryCount
	metrics.JobsInQueue.WithLabelValues("retry", "").Set(float64(retryCount))

	deadLetterCount, err := q.client.LLen(q.ctx, queuePrefixDeadLetter).Result()
	if err != nil {
		q.logger.Error("Failed to get dead letter queue length", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}
	stats["deadLetter"] = deadLetterCount
	metrics.JobsInQueue.WithLabelValues("dead_letter", "").Set(float64(deadLetterCount))

	return stats, nil
}
