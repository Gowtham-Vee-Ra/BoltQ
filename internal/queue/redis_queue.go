package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"

	"BoltQ/internal/job"
	"BoltQ/pkg/logger"
)

var ctx = context.Background()

const (
	TaskQueue       = "task_queue"
	RetryQueue      = "retry_queue"
	DeadLetterQueue = "dead_letter_queue"
	JobStatusPrefix = "job:status:"
)

type RedisQueue struct {
	client *redis.Client
}

func NewRedisQueue(redisAddr string) *RedisQueue {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Validate connection
	_, err := client.Ping(ctx).Result()
	if err != nil {
		msg := fmt.Sprintf("Failed to connect to Redis: %v", err)
		logger.Error(&msg)
		panic(err)
	}

	return &RedisQueue{client: client}
}

// PublishJob adds a job to the queue
func (q *RedisQueue) PublishJob(j *job.Job) error {
	// Serialize the job
	jobStr, err := j.ToJSON()
	if err != nil {
		return err
	}

	// Save job status
	err = q.saveJobStatus(j)
	if err != nil {
		return err
	}

	// Add to queue
	err = q.client.LPush(ctx, TaskQueue, jobStr).Err()
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("Job added to queue: %s", j.ID)
	logger.Info(&msg)
	return nil
}

// ConsumeJob retrieves a job from the queue
func (q *RedisQueue) ConsumeJob() (*job.Job, error) {
	// First check retry queue
	result, err := q.client.RPop(ctx, RetryQueue).Result()
	if err == redis.Nil {
		// If retry queue is empty, check main queue
		result, err = q.client.RPop(ctx, TaskQueue).Result()
		if err == redis.Nil {
			return nil, nil // No jobs in either queue
		} else if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// Parse the job
	j, err := job.FromJSON(result)
	if err != nil {
		return nil, err
	}

	return j, nil
}

// RetryJob adds a job to the retry queue
func (q *RedisQueue) RetryJob(j *job.Job) error {
	// Increment attempts and update status
	j.IncrementAttempts()

	// Serialize the job
	jobStr, err := j.ToJSON()
	if err != nil {
		return err
	}

	// Update job status
	err = q.saveJobStatus(j)
	if err != nil {
		return err
	}

	// Check if job should go to retry queue or dead letter queue
	if j.Status == job.StatusDeadLetter {
		err = q.client.LPush(ctx, DeadLetterQueue, jobStr).Err()
		if err != nil {
			return err
		}
		msg := fmt.Sprintf("Job moved to dead letter queue: %s", j.ID)
		logger.Info(&msg)
	} else {
		// Add a delay before retry (exponential backoff)
		delay := time.Duration(1<<uint(j.Attempts-1)) * time.Second
		time.Sleep(delay)

		err = q.client.LPush(ctx, RetryQueue, jobStr).Err()
		if err != nil {
			return err
		}
		msg := fmt.Sprintf("Job scheduled for retry: %s (attempt %d/%d)", j.ID, j.Attempts, j.MaxAttempts)
		logger.Info(&msg)
	}

	return nil
}

// UpdateJobStatus updates the status of a job
func (q *RedisQueue) UpdateJobStatus(j *job.Job) error {
	return q.saveJobStatus(j)
}

// GetJobStatus retrieves the current status of a job
func (q *RedisQueue) GetJobStatus(jobID string) (*job.Job, error) {
	result, err := q.client.Get(ctx, JobStatusPrefix+jobID).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("job not found: %s", jobID)
	} else if err != nil {
		return nil, err
	}

	return job.FromJSON(result)
}

// saveJobStatus saves the job status to Redis
func (q *RedisQueue) saveJobStatus(j *job.Job) error {
	jobStr, err := j.ToJSON()
	if err != nil {
		return err
	}

	// Store the job with its status
	err = q.client.Set(ctx, JobStatusPrefix+j.ID, jobStr, 0).Err()
	if err != nil {
		return err
	}

	return nil
}

// CountJobs returns the number of jobs in each queue
func (q *RedisQueue) CountJobs() (map[string]int64, error) {
	counts := make(map[string]int64)

	pending, err := q.client.LLen(ctx, TaskQueue).Result()
	if err != nil {
		return nil, err
	}
	counts["pending"] = pending

	retrying, err := q.client.LLen(ctx, RetryQueue).Result()
	if err != nil {
		return nil, err
	}
	counts["retrying"] = retrying

	deadLetter, err := q.client.LLen(ctx, DeadLetterQueue).Result()
	if err != nil {
		return nil, err
	}
	counts["dead_letter"] = deadLetter

	return counts, nil
}
