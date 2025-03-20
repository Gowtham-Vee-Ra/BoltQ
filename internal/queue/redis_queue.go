package queue

import (
	"BoltQ/pkg/config"
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

type RedisQueue struct {
	client *redis.Client
}

func NewRedisQueue() *RedisQueue {
	// Get Redis address from environment with default value
	// When using WSL, you'll set this to your WSL IP address
	redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")

	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Test the connection
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		fmt.Printf("Failed to connect to Redis at %s: %v\n", redisAddr, err)
	} else {
		fmt.Printf("Connected to Redis: %s\n", pong)
	}

	return &RedisQueue{client: client}
}

func (q *RedisQueue) Publish(task string) error {
	err := q.client.LPush(ctx, "task_queue", task).Err()
	if err != nil {
		return err
	}

	fmt.Println("Task added:", task)
	return nil
}

func (q *RedisQueue) Consume() (string, error) {
	task, err := q.client.RPop(ctx, "task_queue").Result()
	if err != nil {
		return "", err
	}

	return task, nil
}
