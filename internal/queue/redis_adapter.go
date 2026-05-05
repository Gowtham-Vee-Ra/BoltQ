package queue

import (
	"context"
	"time"
)

type RedisQueueAdapter struct {
	redisQueue *RedisQueue
}

func NewRedisQueueAdapter(redisQueue *RedisQueue) Queue {
	return &RedisQueueAdapter{redisQueue: redisQueue}
}

func (a *RedisQueueAdapter) Publish(ctx context.Context, job *Job) error {
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

func (a *RedisQueueAdapter) PublishDelayed(ctx context.Context, job *Job, delay time.Duration) error {
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
	return a.redisQueue.PublishDelayed(task, int(delay.Seconds()))
}

func (a *RedisQueueAdapter) Consume(ctx context.Context) (*Job, error) {
	task, err := a.redisQueue.Consume(ctx)
	if err != nil {
		return nil, err
	}

	return &Job{
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
	}, nil
}

func (a *RedisQueueAdapter) UpdateStatus(ctx context.Context, jobID string, status JobStatus, err error) error {
	task, getErr := a.redisQueue.GetTaskStatus(jobID)
	if getErr != nil {
		return getErr
	}

	task.Status = string(status)
	if err != nil {
		task.LastError = err.Error()
	}
	return a.redisQueue.UpdateStatus(task)
}

func (a *RedisQueueAdapter) GetJob(ctx context.Context, jobID string) (*Job, error) {
	task, err := a.redisQueue.GetTaskStatus(jobID)
	if err != nil {
		return nil, err
	}

	return &Job{
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
	}, nil
}

func (a *RedisQueueAdapter) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return a.redisQueue.GetQueueStats()
}

func (a *RedisQueueAdapter) Close() error {
	return a.redisQueue.Close()
}
