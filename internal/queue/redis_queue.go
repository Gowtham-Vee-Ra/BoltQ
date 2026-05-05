package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	ctx             = context.Background()
	ErrTaskNotFound = errors.New("task not found")
)

const (
	TaskQueuePrefix = "task_queue"
	DelayedTasksKey = "delayed_tasks"
	DeadLetterQueue = "dead_letter_queue"
	JobIndexKey     = "job_index"
)

type Task struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Data        map[string]interface{} `json:"data"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Priority    int                    `json:"priority"`
	CreatedAt   time.Time              `json:"created_at"`
	ScheduledAt time.Time              `json:"scheduled_at,omitempty"`
	Status      string                 `json:"status"`
	Attempts    int                    `json:"attempts"`
	LastError   string                 `json:"last_error,omitempty"`
}

type RedisQueue struct {
	client *redis.Client
	logger Logger
}

func NewRedisQueue(client *redis.Client, logger Logger) *RedisQueue {
	return &RedisQueue{client: client, logger: logger}
}

func (q *RedisQueue) Publish(task *Task) error {
	task.CreatedAt = time.Now()
	task.Status = "pending"
	return q.publishToQueue(task, getQueueName(task.Priority))
}

func (q *RedisQueue) PublishDelayed(task *Task, delaySeconds int) error {
	task.CreatedAt = time.Now()
	task.ScheduledAt = time.Now().Add(time.Duration(delaySeconds) * time.Second)
	task.Status = "scheduled"

	taskJSON, err := json.Marshal(task)
	if err != nil {
		return err
	}

	err = q.client.ZAdd(ctx, DelayedTasksKey, &redis.Z{
		Score:  float64(task.ScheduledAt.Unix()),
		Member: string(taskJSON),
	}).Err()
	if err != nil {
		return err
	}

	q.logger.Info(fmt.Sprintf("Task %s scheduled for %s", task.ID, task.ScheduledAt.Format(time.RFC3339)))
	return nil
}

const delayedTasksBatchSize = 100

// ProcessDelayedTasks moves ready tasks from the delayed sorted set to regular queues.
func (q *RedisQueue) ProcessDelayedTasks() (int, error) {
	now := time.Now().Unix()

	// cap batch to avoid memory spikes
	tasks, err := q.client.ZRangeByScore(ctx, DelayedTasksKey, &redis.ZRangeBy{
		Min: "0", Max: fmt.Sprintf("%d", now),
		Offset: 0, Count: delayedTasksBatchSize,
	}).Result()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, taskJSON := range tasks {
		var task Task
		if err := json.Unmarshal([]byte(taskJSON), &task); err != nil {
			q.logger.Info(fmt.Sprintf("Error unmarshalling delayed task: %v", err))
			continue
		}

		task.Status = "pending"
		if err := q.publishToQueue(&task, getQueueName(task.Priority)); err != nil {
			q.logger.Info(fmt.Sprintf("Error publishing delayed task %s: %v", task.ID, err))
			continue
		}

		if err := q.client.ZRem(ctx, DelayedTasksKey, taskJSON).Err(); err != nil {
			q.logger.Info(fmt.Sprintf("Error removing task %s from delayed set: %v", task.ID, err))
			continue
		}

		count++
	}

	return count, nil
}

// Consume blocks up to 1s for the next task, checking high priority first.
// BRPop checks keys left-to-right, so high→low gives us priority ordering.
func (q *RedisQueue) Consume(ctx context.Context) (*Task, error) {
	result, err := q.client.BRPop(ctx, time.Second,
		getQueueName(PriorityHigh),
		getQueueName(PriorityNormal),
		getQueueName(PriorityLow),
	).Result()

	if err == redis.Nil {
		return nil, redis.Nil
	}
	if err != nil {
		return nil, err
	}

	var task Task
	if err := json.Unmarshal([]byte(result[1]), &task); err != nil {
		return nil, err
	}

	task.Status = "running"
	if err := q.UpdateStatus(&task); err != nil {
		q.logger.Info(fmt.Sprintf("Failed to update status for task %s: %v", task.ID, err))
	}

	return &task, nil
}

func (q *RedisQueue) MoveToDeadLetterQueue(task *Task, err error) error {
	task.Status = "failed"
	task.LastError = err.Error()

	taskJSON, jsonErr := json.Marshal(task)
	if jsonErr != nil {
		return jsonErr
	}

	return q.client.LPush(ctx, DeadLetterQueue, string(taskJSON)).Err()
}

func (q *RedisQueue) RetryTask(task *Task, err error) error {
	task.Attempts++
	task.Status = "retrying"
	task.LastError = err.Error()

	backoffSeconds := 1 << uint(task.Attempts)
	if backoffSeconds > 300 {
		backoffSeconds = 300
	}

	return q.PublishDelayed(task, backoffSeconds)
}

func (q *RedisQueue) UpdateStatus(task *Task) error {
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("task:%s", task.ID)
	return q.client.Set(ctx, key, string(taskJSON), 24*time.Hour).Err()
}

func (q *RedisQueue) GetTaskStatus(taskID string) (*Task, error) {
	key := fmt.Sprintf("task:%s", taskID)
	taskJSON, err := q.client.Get(ctx, key).Result()

	if err == redis.Nil {
		return nil, ErrTaskNotFound
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

func (q *RedisQueue) GetQueueStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	for priority := PriorityHigh; priority >= PriorityLow; priority-- {
		queueName := getQueueName(priority)
		count, err := q.client.LLen(ctx, queueName).Result()
		if err != nil {
			return nil, err
		}
		stats[queueName] = count
	}

	delayedCount, err := q.client.ZCard(ctx, DelayedTasksKey).Result()
	if err != nil {
		return nil, err
	}
	stats[DelayedTasksKey] = delayedCount

	deadLetterCount, err := q.client.LLen(ctx, DeadLetterQueue).Result()
	if err != nil {
		return nil, err
	}
	stats[DeadLetterQueue] = deadLetterCount

	return stats, nil
}

func (q *RedisQueue) RemoveFromQueue(taskID string) (bool, error) {
	for priority := PriorityHigh; priority >= PriorityLow; priority-- {
		queueName := getQueueName(priority)
		members, err := q.client.LRange(ctx, queueName, 0, -1).Result()
		if err != nil {
			return false, err
		}

		for _, member := range members {
			var t Task
			if err := json.Unmarshal([]byte(member), &t); err != nil {
				continue
			}
			if t.ID == taskID {
				removed, err := q.client.LRem(ctx, queueName, 1, member).Result()
				if err != nil {
					return false, err
				}
				return removed > 0, nil
			}
		}
	}
	return false, nil
}

func (q *RedisQueue) ListJobs(limit, offset int) ([]*Task, error) {
	ids, err := q.client.LRange(ctx, JobIndexKey, int64(offset), int64(offset+limit-1)).Result()
	if err != nil {
		return nil, err
	}

	tasks := make([]*Task, 0, len(ids))
	for _, id := range ids {
		task, err := q.GetTaskStatus(id)
		if err != nil {
			continue // skip expired/missing tasks
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (q *RedisQueue) Close() error {
	return q.client.Close()
}

func getQueueName(priority int) string {
	return fmt.Sprintf("%s:%d", TaskQueuePrefix, priority)
}

func (q *RedisQueue) publishToQueue(task *Task, queueName string) error {
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return err
	}

	if err = q.client.LPush(ctx, queueName, string(taskJSON)).Err(); err != nil {
		return err
	}

	// SetNX deduplicates retry re-queues in the index
	indexedKey := fmt.Sprintf("job_indexed:%s", task.ID)
	added, err := q.client.SetNX(ctx, indexedKey, "1", 24*time.Hour).Result()
	if err == nil && added {
		q.client.LPush(ctx, JobIndexKey, task.ID)
	}

	q.logger.Info(fmt.Sprintf("Task %s added to queue %s", task.ID, queueName))
	return nil
}

type RedisQueueFactory struct {
	logger Logger
}

func NewRedisQueueFactory(logger Logger) QueueFactory {
	return &RedisQueueFactory{logger: logger}
}

func (f *RedisQueueFactory) CreateQueue(config map[string]string) (Queue, error) {
	redisAddr := "localhost:6379"
	if addr, ok := config["addr"]; ok {
		redisAddr = addr
	}

	client := redis.NewClient(&redis.Options{Addr: redisAddr})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		f.logger.Info("Failed to connect to Redis: " + err.Error())
		return nil, err
	}

	f.logger.Info("Connected to Redis at " + redisAddr)
	return NewRedisQueueAdapter(NewRedisQueue(client, f.logger)), nil
}

func (f *RedisQueueFactory) Close() error {
	return nil
}
