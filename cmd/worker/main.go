package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"BoltQ/internal/queue"
	"BoltQ/internal/worker"
	"BoltQ/pkg/config"
	"BoltQ/pkg/logger"

	"github.com/joho/godotenv"
)

// Job represents a task to be processed
type Job struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Payload  map[string]interface{} `json:"payload"`
	Priority int                    `json:"priority,omitempty"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found or error loading it")
	}

	fmt.Println("Worker service started...")

	// Initialize the Redis queue
	redisQueue := queue.NewRedisQueue()

	// Create a channel to handle graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Set polling interval from environment or use default
	pollingInterval := config.GetEnv("POLLING_INTERVAL", "1")
	interval, err := time.ParseDuration(pollingInterval + "s")
	if err != nil {
		interval = 1 * time.Second
	}

	// Start the worker loop
	go startWorker(redisQueue, interval)

	// Wait for shutdown signal
	<-shutdown
	fmt.Println("Shutting down worker...")
}

func startWorker(q *queue.RedisQueue, interval time.Duration) {
	for {
		// Try to get a job from the queue
		jobJSON, err := q.Consume()
		if err != nil {
			// If no job is available, wait before polling again
			if err.Error() == "redis: nil" {
				time.Sleep(interval)
				continue
			}

			// Log other errors
			errorMsg := fmt.Sprintf("Error consuming from queue: %v", err)
			logger.Error(errorMsg)
			time.Sleep(interval)
			continue
		}

		// Parse the job
		var job Job
		if err := json.Unmarshal([]byte(jobJSON), &job); err != nil {
			errorMsg := fmt.Sprintf("Error parsing job: %v", err)
			logger.Error(errorMsg)
			continue
		}

		// Log job processing
		infoMsg := fmt.Sprintf("Processing job: %s, type: %s", job.ID, job.Type)
		logger.Info(infoMsg)

		// Process the job
		worker.ProcessTask(&jobJSON)
	}
}
