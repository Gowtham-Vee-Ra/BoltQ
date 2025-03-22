// cmd/test/main.go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"BoltQ/internal/job"
	"BoltQ/internal/queue"
	"BoltQ/internal/worker"
	"BoltQ/pkg/logger"
)

func main() {
	// Initialize logger
	logger.Setup("INFO")
	logger.Info("Starting BoltQ test program...")

	// Connect to Redis
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisQueue := queue.NewRedisQueue(redisAddr, "", 0)
	defer redisQueue.Close()

	// Create some test jobs
	createTestJobs(redisQueue)

	// Create and start a worker
	worker1 := worker.NewWorker("test-worker-1", redisQueue)
	worker1.RegisterDefaultProcessors()
	worker1.Start()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Keep the program running until interrupted
	select {
	case <-sigChan:
		logger.Info("Received shutdown signal")
		worker1.Stop()
	case <-time.After(30 * time.Second):
		logger.Info("Test completed")
		worker1.Stop()
	}

	// Give the worker time to shut down
	time.Sleep(1 * time.Second)
	logger.Info("Test program exiting")
}

func createTestJobs(q *queue.RedisQueue) {
	// Create a few test jobs of different types
	jobs := []*job.Job{
		job.NewJob("email", "Send welcome email to user@example.com"),
		job.NewJob("notification", "System alert: CPU usage high"),
		job.NewJob("test", "Test job payload"),
	}

	// Set different priorities
	jobs[0].SetPriority(job.PriorityNormal)
	jobs[1].SetPriority(job.PriorityHigh)
	jobs[2].SetPriority(job.PriorityLow)

	// Add to the queue
	for _, j := range jobs {
		jobJSON, _ := j.ToJSON()
		err := q.Publish(jobJSON)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to publish job: %v", err))
		} else {
			logger.Info(fmt.Sprintf("Published job: %s (type: %s, priority: %d)", j.ID, j.Type, j.Priority))
		}
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
