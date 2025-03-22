// cmd/test/main.go
package main

import (
	"context"
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
	"BoltQ/pkg/metrics"
	"BoltQ/pkg/tracing"
)

// SampleProcessor processes a sample job
func SampleProcessor(ctx context.Context, j *job.Job) error {
	log := logger.NewLogger("sample-processor")
	log.WithJob(j.ID, "Processing sample job", map[string]interface{}{
		"data": j.Data,
	})

	// Simulate processing
	time.Sleep(2 * time.Second)

	log.WithJob(j.ID, "Sample job processed successfully")
	return nil
}

// EmailProcessor processes an email job
func EmailProcessor(ctx context.Context, j *job.Job) error {
	log := logger.NewLogger("email-processor")
	log.WithJob(j.ID, "Sending email", map[string]interface{}{
		"data": j.Data,
	})

	// Simulate sending email
	time.Sleep(1 * time.Second)

	log.WithJob(j.ID, "Email sent successfully")
	return nil
}

// NotificationProcessor processes a notification job
func NotificationProcessor(ctx context.Context, j *job.Job) error {
	log := logger.NewLogger("notification-processor")
	log.WithJob(j.ID, "Sending notification", map[string]interface{}{
		"data": j.Data,
	})

	// Simulate sending notification
	time.Sleep(500 * time.Millisecond)

	log.WithJob(j.ID, "Notification sent successfully")
	return nil
}

func main() {
	// Initialize logger
	log := logger.NewLogger("test")
	log.Info("Starting BoltQ test program...")

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize tracing
	shutdownTracer, err := tracing.InitTracer(ctx, "boltq-test")
	if err != nil {
		log.Error("Failed to initialize tracer", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		defer shutdownTracer()
	}

	// Get configuration
	redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")
	metricsPort := config.GetEnv("METRICS_PORT", "9092")

	// Setup Redis queue
	redisQueue := queue.NewRedisQueue(ctx, redisAddr)

	// Setup metrics server
	metrics.SetupMetricsServer(":" + metricsPort)
	log.Info("Metrics server started", map[string]interface{}{
		"port": metricsPort,
	})

	// Create test jobs
	createTestJobs(ctx, redisQueue, log)

	// Create worker pool
	pool := worker.NewWorkerPool(ctx, redisQueue, 2, 3)

	// Register job processors
	pool.RegisterProcessor("sample", SampleProcessor)
	pool.RegisterProcessor("email", EmailProcessor)
	pool.RegisterProcessor("notification", NotificationProcessor)
	pool.RegisterProcessor("test", SampleProcessor) // Reuse sample processor for test jobs

	// Start the worker pool
	pool.Start()
	log.Info("Worker pool started", map[string]interface{}{
		"worker_count": 2,
	})

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for interrupt or timeout
	select {
	case <-sigChan:
		log.Info("Received shutdown signal")
	case <-time.After(30 * time.Second):
		log.Info("Test completed")
	}

	// Stop the worker pool
	log.Info("Stopping worker pool")
	pool.Stop()

	// Give time for cleanup
	time.Sleep(1 * time.Second)
	log.Info("Test program exiting")
}

func createTestJobs(ctx context.Context, q *queue.RedisQueue, log *logger.Logger) {
	// Create different job types with different priorities
	jobTypes := []string{"email", "notification", "test"}
	priorities := []job.Priority{job.PriorityLow, job.PriorityNormal, job.PriorityHigh, job.PriorityCritical}

	for i, jobType := range jobTypes {
		// Create a job
		j := &job.Job{
			ID:        fmt.Sprintf("test-job-%d", i+1),
			Type:      jobType,
			Priority:  priorities[i%len(priorities)],
			CreatedAt: time.Now(),
			Status:    job.StatusPending,
			Data: map[string]interface{}{
				"message": fmt.Sprintf("Test payload for %s job", jobType),
			},
		}

		// Publish to queue
		err := q.PublishJob(j)
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
				"priority": string(j.Priority),
			})
		}

		// Add a short delay between job creations
		time.Sleep(100 * time.Millisecond)
	}

	// Add one more job with Critical priority
	criticalJob := &job.Job{
		ID:        "critical-job",
		Type:      "notification",
		Priority:  job.PriorityCritical,
		CreatedAt: time.Now(),
		Status:    job.StatusPending,
		Data: map[string]interface{}{
			"message": "CRITICAL ALERT: This should be processed first!",
		},
	}

	err := q.PublishJob(criticalJob)
	if err != nil {
		log.Error("Failed to publish critical job", map[string]interface{}{
			"job_id": criticalJob.ID,
			"error":  err.Error(),
		})
	} else {
		log.Info("Published critical job", map[string]interface{}{
			"job_id": criticalJob.ID,
		})
	}
}
