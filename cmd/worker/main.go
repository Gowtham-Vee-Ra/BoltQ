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
	"BoltQ/pkg/config"
	"BoltQ/pkg/logger"

	"github.com/joho/godotenv"
)

// Example job processor function
func processGenericJob(j *job.Job) error {
	msg := fmt.Sprintf("Processing job: %s with payload: %v", j.ID, j.Payload)
	logger.Info(&msg)

	// Simulate work
	time.Sleep(2 * time.Second)

	// For testing error handling, uncomment this to make some jobs fail
	// if rand.Intn(10) < 3 { // 30% chance of failure
	//     return fmt.Errorf("simulated random error")
	// }

	return nil
}

// Example email job processor
func processEmailJob(j *job.Job) error {
	recipient, ok := j.Payload["recipient"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid recipient")
	}

	subject, ok := j.Payload["subject"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid subject")
	}

	msg := fmt.Sprintf("Sending email to %s with subject: %s", recipient, subject)
	logger.Info(&msg)

	// Simulate sending email
	time.Sleep(1 * time.Second)

	return nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		// Only log as info, not error, since .env file is optional
		msg := "No .env file found, using environment variables"
		logger.Info(&msg)
	} else {
		msg := ".env file loaded successfully"
		logger.Info(&msg)
	}

	// Log startup
	startupMsg := "Worker service starting..."
	logger.Info(&startupMsg)

	// Load configuration
	redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")
	workerID := config.GetEnv("WORKER_ID", fmt.Sprintf("worker-%d", time.Now().UnixNano()))

	// Initialize Redis queue
	redisQueue := queue.NewRedisQueue(redisAddr)

	// Create worker
	w := worker.NewWorker(workerID, redisQueue)

	// Register job processors
	w.RegisterProcessor("generic", processGenericJob)
	w.RegisterProcessor("email", processEmailJob)

	// Start worker
	w.Start()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	<-sigChan

	// Shut down worker
	w.Stop()

	shutdownMsg := "Worker service shut down gracefully"
	logger.Info(&shutdownMsg)
}
