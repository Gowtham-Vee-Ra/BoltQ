// cmd/worker/main.go
package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"strconv"
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

	// Create span for job processing
	ctx, span := tracing.StartSpan(ctx, "process_sample_job")
	defer span.End()

	// Simulate processing
	time.Sleep(2 * time.Second)

	log.WithJob(j.ID, "Sample job processed successfully")
	return nil
}

func main() {
	log := logger.NewLogger("worker-service")
	log.Info("Starting Worker Service")

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize tracing
	shutdownTracer, err := tracing.InitTracer(ctx, "boltq-worker")
	if err != nil {
		log.Error("Failed to initialize tracer", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		defer shutdownTracer()
	}

	// Get configuration
	redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")
	metricsPort := config.GetEnv("METRICS_PORT", "9091")

	// Get worker count from environment or use CPU count
	workerCountStr := config.GetEnv("NUM_WORKERS", "")
	workerCount := runtime.NumCPU()
	if workerCountStr != "" {
		if count, err := strconv.Atoi(workerCountStr); err == nil && count > 0 {
			workerCount = count
		}
	}

	// Get max attempts from environment
	maxAttemptsStr := config.GetEnv("MAX_ATTEMPTS", "3")
	maxAttempts := 3
	if count, err := strconv.Atoi(maxAttemptsStr); err == nil && count > 0 {
		maxAttempts = count
	}

	// Setup Redis queue
	redisQueue := queue.NewRedisQueue(ctx, redisAddr)

	// Setup metrics server
	metrics.SetupMetricsServer(":" + metricsPort)
	log.Info("Metrics server started", map[string]interface{}{
		"port": metricsPort,
	})

	// Create worker pool
	pool := worker.NewWorkerPool(ctx, redisQueue, workerCount, maxAttempts)

	// Register job processors
	pool.RegisterProcessor("sample", SampleProcessor)

	// Start the worker pool
	pool.Start()

	log.Info("Worker pool started", map[string]interface{}{
		"worker_count": workerCount,
		"max_attempts": maxAttempts,
	})

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("Shutting down worker service")

	// Stop the worker pool
	pool.Stop()

	log.Info("Worker service stopped")
}
