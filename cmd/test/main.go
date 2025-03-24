// cmd/test/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"BoltQ/internal/queue"
	"BoltQ/internal/worker"
	"BoltQ/pkg/config"
	"BoltQ/pkg/logger"

	"github.com/go-redis/redis/v8"
)

// LoggerInterface defines a common interface that works with our logger implementation
type LoggerInterface interface {
	Info(msg string, fields ...map[string]interface{})
	Error(msg string, fields ...map[string]interface{})
	Debug(msg string, fields ...map[string]interface{})
}

// Define job processors that work with queue.Job instead of job.Job
func sampleProcessor(ctx context.Context, j *queue.Job) error {
	log := logger.NewLogger("sample-processor")
	log.Info("Processing sample job", map[string]interface{}{
		"job_id":  j.ID,
		"payload": j.Payload,
	})

	// Simulate processing
	time.Sleep(2 * time.Second)

	log.Info("Sample job processed successfully", map[string]interface{}{
		"job_id": j.ID,
	})
	return nil
}

func emailProcessor(ctx context.Context, j *queue.Job) error {
	log := logger.NewLogger("email-processor")

	recipient, ok := j.Payload["recipient"].(string)
	if !ok {
		return fmt.Errorf("invalid recipient")
	}

	subject, ok := j.Payload["subject"].(string)
	if !ok {
		return fmt.Errorf("invalid subject")
	}

	log.Info("Sending email", map[string]interface{}{
		"job_id":    j.ID,
		"recipient": recipient,
		"subject":   subject,
	})

	// Simulate sending email
	time.Sleep(1 * time.Second)

	log.Info("Email sent successfully", map[string]interface{}{
		"job_id": j.ID,
	})
	return nil
}

func notificationProcessor(ctx context.Context, j *queue.Job) error {
	log := logger.NewLogger("notification-processor")

	message, ok := j.Payload["message"].(string)
	if !ok {
		return fmt.Errorf("invalid message")
	}

	log.Info("Sending notification", map[string]interface{}{
		"job_id":  j.ID,
		"message": message,
	})

	// Simulate sending notification
	time.Sleep(500 * time.Millisecond)

	log.Info("Notification sent successfully", map[string]interface{}{
		"job_id": j.ID,
	})
	return nil
}

func main() {
	// Initialize logger
	log := logger.NewLogger("test")
	log.Info("Starting BoltQ test program...", nil)

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get configuration
	redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")
	numWorkers := 2

	// Setup Redis client
	redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Create queue using factory approach
	queueServiceFactory := queue.NewQueueServiceFactory(log)
	queueServiceFactory.InitDefaultFactories()

	q, err := queueServiceFactory.CreateQueue(queue.QueueTypeRedis, map[string]string{
		"addr": redisAddr,
	})

	if err != nil {
		log.Error("Failed to create queue", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}
	defer q.Close()

	// Create test jobs - use explicit typing for the logger argument
	createTestJobs(ctx, q, LoggerInterface(log))

	// Create workers
	workers := make([]*worker.Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		// Use explicit typing for the logger argument
		workers[i] = worker.NewWorker(fmt.Sprintf("worker-%d", i), q, *log)

		// Register job processors for each worker
		workers[i].RegisterProcessor("sample", sampleProcessor)
		workers[i].RegisterProcessor("email", emailProcessor)
		workers[i].RegisterProcessor("notification", notificationProcessor)
		workers[i].RegisterProcessor("test", sampleProcessor) // Reuse sample processor for test jobs

		// Start the worker
		workers[i].Start()
	}

	log.Info("Workers started", map[string]interface{}{
		"worker_count": numWorkers,
	})

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for interrupt or timeout
	select {
	case <-sigChan:
		log.Info("Received shutdown signal", nil)
	case <-time.After(30 * time.Second):
		log.Info("Test completed", nil)
	}

	// Stop the workers
	log.Info("Stopping workers", nil)
	for _, w := range workers {
		w.Stop()
	}

	// Give time for cleanup
	time.Sleep(1 * time.Second)
	log.Info("Test program exiting", nil)
}

func createTestJobs(ctx context.Context, q queue.Queue, log LoggerInterface) {
	// Create different job types with different priorities
	jobTypes := []string{"email", "notification", "test"}
	priorities := []int{queue.PriorityLow, queue.PriorityNormal, queue.PriorityHigh}

	for i, jobType := range jobTypes {
		// Create a job
		j := &queue.Job{
			ID:        fmt.Sprintf("test-job-%d", i+1),
			Type:      jobType,
			Priority:  priorities[i%len(priorities)],
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Status:    queue.StatusPending,
			Payload: map[string]interface{}{
				"message":   fmt.Sprintf("Test payload for %s job", jobType),
				"recipient": "user@example.com",
				"subject":   fmt.Sprintf("Test %s", jobType),
			},
		}

		// Publish to queue
		err := q.Publish(ctx, j)
		if err != nil {
			log.Error("Failed to publish job", map[string]interface{}{
				"job_id": j.ID,
				"type":   j.Type,
				"error":  err.Error(),
			})
		} else {
			log.Info("Published job", map[string]interface{}{
				"job_id":   j.ID,
				"type":     j.Type,
				"priority": j.Priority,
			})
		}

		// Add a short delay between job creations
		time.Sleep(100 * time.Millisecond)
	}

	// Add one more job with high priority
	criticalJob := &queue.Job{
		ID:        "critical-job",
		Type:      "notification",
		Priority:  queue.PriorityHigh,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    queue.StatusPending,
		Payload: map[string]interface{}{
			"message": "HIGH PRIORITY ALERT: This should be processed first!",
		},
	}

	err := q.Publish(ctx, criticalJob)
	if err != nil {
		log.Error("Failed to publish high priority job", map[string]interface{}{
			"job_id": criticalJob.ID,
			"error":  err.Error(),
		})
	} else {
		log.Info("Published high priority job", map[string]interface{}{
			"job_id": criticalJob.ID,
		})
	}

	// Add a job with delay
	delayedJob := &queue.Job{
		ID:        "delayed-job",
		Type:      "email",
		Priority:  queue.PriorityNormal,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    queue.StatusPending,
		Payload: map[string]interface{}{
			"message":   "This is a delayed job that should execute after 5 seconds",
			"recipient": "delayed@example.com",
			"subject":   "Delayed Test Email",
		},
	}

	err = q.PublishDelayed(ctx, delayedJob, 5*time.Second)
	if err != nil {
		log.Error("Failed to publish delayed job", map[string]interface{}{
			"job_id": delayedJob.ID,
			"error":  err.Error(),
		})
	} else {
		log.Info("Published delayed job", map[string]interface{}{
			"job_id":       delayedJob.ID,
			"scheduled_at": time.Now().Add(5 * time.Second),
		})
	}
}
