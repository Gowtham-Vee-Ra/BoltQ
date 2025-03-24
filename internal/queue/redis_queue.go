// internal/queue/redis_queue.go
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

const (
	// Queue names
	TaskQueuePrefix = "task_queue"
	DelayedTasksKey = "delayed_tasks"
	DeadLetterQueue = "dead_letter_queue"
)

// Task represents a job to be processed
type Task struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Data        map[string]interface{} `json:"data"`
	Priority    int                    `json:"priority"`
	CreatedAt   time.Time              `json:"created_at"`
	ScheduledAt time.Time              `json:"scheduled_at,omitempty"`
	Status      string                 `json:"status"`
	Attempts    int                    `json:"attempts"`
	LastError   string                 `json:"last_error,omitempty"`
}

// RedisQueue implements a Redis-backed task queue
type RedisQueue struct {
	client *redis.Client
	logger Logger
}

// NewRedisQueue creates a new Redis queue
func NewRedisQueue(client *redis.Client, logger Logger) *RedisQueue {
	return &RedisQueue{
		client: client,
		logger: logger,
	}
}

// Publish adds a task to the queue immediately
func (q *RedisQueue) Publish(task *Task) error {
	task.CreatedAt = time.Now()
	task.Status = "pending"

	return q.publishToQueue(task, getQueueName(task.Priority))
}

// PublishDelayed schedules a task for future execution
func (q *RedisQueue) PublishDelayed(task *Task, delaySeconds int) error {
	task.CreatedAt = time.Now()
	task.ScheduledAt = time.Now().Add(time.Duration(delaySeconds) * time.Second)
	task.Status = "scheduled"

	taskJSON, err := json.Marshal(task)
	if err != nil {
		return err
	}

	// Store in a Redis sorted set with score = unix timestamp when task should execute
	score := float64(task.ScheduledAt.Unix())
	err = q.client.ZAdd(ctx, DelayedTasksKey, &redis.Z{
		Score:  score,
		Member: string(taskJSON),
	}).Err()

	if err != nil {
		return err
	}

	q.logger.Info(fmt.Sprintf("Task %s scheduled for %s", task.ID, task.ScheduledAt.Format(time.RFC3339)))
	return nil
}

// ProcessDelayedTasks moves ready tasks from delayed set to regular queue
func (q *RedisQueue) ProcessDelayedTasks() (int, error) {
	now := time.Now().Unix()

	// Find tasks that are ready to be processed (score <= current timestamp)
	tasks, err := q.client.ZRangeByScore(ctx, DelayedTasksKey, &redis.ZRangeBy{
		Min: "0",
		Max: fmt.Sprintf("%d", now),
	}).Result()

	if err != nil {
		return 0, err
	}

	count := 0

	// Process each ready task
	for _, taskJSON := range tasks {
		var task Task
		if err := json.Unmarshal([]byte(taskJSON), &task); err != nil {
			q.logger.Info(fmt.Sprintf("Error unmarshalling delayed task: %v", err))
			continue
		}

		// Update status and publish to appropriate queue
		task.Status = "pending"
		if err := q.publishToQueue(&task, getQueueName(task.Priority)); err != nil {
			q.logger.Info(fmt.Sprintf("Error publishing delayed task %s: %v", task.ID, err))
			continue
		}

		// Remove from delayed set
		if err := q.client.ZRem(ctx, DelayedTasksKey, taskJSON).Err(); err != nil {
			q.logger.Info(fmt.Sprintf("Error removing task %s from delayed set: %v", task.ID, err))
			continue
		}

		count++
	}

	return count, nil
}

// Consume retrieves a task from the queue, checking high priority first
func (q *RedisQueue) Consume() (*Task, error) {
	// Try to consume from high priority to low priority
	for priority := PriorityHigh; priority <= PriorityLow; priority++ {
		queueName := getQueueName(priority)
		taskJSON, err := q.client.RPop(ctx, queueName).Result()

		if err == redis.Nil {
			// No tasks in this queue, try the next one
			continue
		}

		if err != nil {
			return nil, err
		}

		var task Task
		if err := json.Unmarshal([]byte(taskJSON), &task); err != nil {
			return nil, err
		}

		// Update status
		task.Status = "running"
		if err := q.UpdateStatus(&task); err != nil {
			q.logger.Info(fmt.Sprintf("Failed to update status for task %s: %v", task.ID, err))
		}

		return &task, nil
	}

	// No tasks in any queue
	return nil, redis.Nil
}

// MoveToDeadLetterQueue moves a failed task to the dead letter queue
func (q *RedisQueue) MoveToDeadLetterQueue(task *Task, err error) error {
	task.Status = "failed"
	task.LastError = err.Error()

	taskJSON, jsonErr := json.Marshal(task)
	if jsonErr != nil {
		return jsonErr
	}

	return q.client.LPush(ctx, DeadLetterQueue, string(taskJSON)).Err()
}

// RetryTask schedules a task for retry with exponential backoff
func (q *RedisQueue) RetryTask(task *Task, err error) error {
	task.Attempts++
	task.Status = "retrying"
	task.LastError = err.Error()

	// Calculate backoff time: 2^attempts seconds, capped at 5 minutes
	backoffSeconds := 1 << uint(task.Attempts)
	if backoffSeconds > 300 {
		backoffSeconds = 300
	}

	return q.PublishDelayed(task, backoffSeconds)
}

// UpdateStatus updates a task's status in Redis
func (q *RedisQueue) UpdateStatus(task *Task) error {
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return err
	}

	// Store status with TTL
	key := fmt.Sprintf("task:%s", task.ID)
	return q.client.Set(ctx, key, string(taskJSON), 24*time.Hour).Err()
}

// GetTaskStatus retrieves a task's current status
func (q *RedisQueue) GetTaskStatus(taskID string) (*Task, error) {
	key := fmt.Sprintf("task:%s", taskID)
	taskJSON, err := q.client.Get(ctx, key).Result()

	if err == redis.Nil {
		return nil, fmt.Errorf("task not found")
	}

	if err != nil {
		return nil, err
	}

	var task Task
	if err := json.Unmarshal([]byte(taskJSON), &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// GetQueueStats returns statistics about the queues
func (q *RedisQueue) GetQueueStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get counts for each priority queue
	for priority := PriorityHigh; priority <= PriorityLow; priority++ {
		queueName := getQueueName(priority)
		count, err := q.client.LLen(ctx, queueName).Result()
		if err != nil {
			return nil, err
		}
		stats[queueName] = count
	}

	// Get count of delayed tasks
	delayedCount, err := q.client.ZCard(ctx, DelayedTasksKey).Result()
	if err != nil {
		return nil, err
	}
	stats[DelayedTasksKey] = delayedCount

	// Get count of dead letter queue
	deadLetterCount, err := q.client.LLen(ctx, DeadLetterQueue).Result()
	if err != nil {
		return nil, err
	}
	stats[DeadLetterQueue] = deadLetterCount

	return stats, nil
}

// Close closes the Redis queue and its connections
func (q *RedisQueue) Close() error {
	return q.client.Close()
}

// Helper function to get the queue name for a priority level
func getQueueName(priority int) string {
	return fmt.Sprintf("%s:%d", TaskQueuePrefix, priority)
}

// Helper to publish a task to a specific queue
func (q *RedisQueue) publishToQueue(task *Task, queueName string) error {
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return err
	}

	err = q.client.LPush(ctx, queueName, string(taskJSON)).Err()
	if err != nil {
		return err
	}

	q.logger.Info(fmt.Sprintf("Task %s added to queue %s", task.ID, queueName))
	return nil
}

// RedisQueueFactory creates Redis-backed queues
type RedisQueueFactory struct {
	logger Logger
}

// NewRedisQueueFactory creates a new Redis queue factory
func NewRedisQueueFactory(logger Logger) QueueFactory {
	return &RedisQueueFactory{
		logger: logger,
	}
}

// CreateQueue creates a new Redis queue with the provided configuration
func (f *RedisQueueFactory) CreateQueue(config map[string]string) (Queue, error) {
	// Get Redis address from config with default fallback
	redisAddr := "localhost:6379"
	if addr, ok := config["addr"]; ok {
		redisAddr = addr
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Ping Redis to ensure connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		f.logger.Info("Failed to connect to Redis: " + err.Error())
		return nil, err
	}

	f.logger.Info("Connected to Redis at " + redisAddr)

	// Create Redis queue
	redisQueue := NewRedisQueue(client, f.logger)

	// Wrap with adapter to implement the Queue interface
	return NewRedisQueueAdapter(redisQueue), nil
}

// Close for the factory (not really needed, but implements the interface)
func (f *RedisQueueFactory) Close() error {
	// Nothing to close in the factory
	return nil
}
