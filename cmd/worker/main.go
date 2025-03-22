// cmd/worker/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"BoltQ/internal/queue"
	"BoltQ/internal/worker"
	"BoltQ/pkg/config"
	"BoltQ/pkg/logger"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	godotenv.Load()

	// Configure logging
	logger.Setup(os.Getenv("LOG_LEVEL"))
	msg := "Worker service starting..."
	logger.Info(msg)

	// Generate worker ID if not provided
	workerID := os.Getenv("WORKER_ID")
	if workerID == "" {
		workerID = fmt.Sprintf("worker-%s", uuid.New().String()[:8])
	}
	msg = fmt.Sprintf("Worker ID: %s", workerID)
	logger.Info(msg)

	// Determine number of workers (default to number of CPUs)
	numWorkersStr := config.GetEnv("NUM_WORKERS", strconv.Itoa(runtime.NumCPU()))
	numWorkers, err := strconv.Atoi(numWorkersStr)
	if err != nil || numWorkers < 1 {
		numWorkers = runtime.NumCPU()
	}
	msg = fmt.Sprintf("Number of workers: %d", numWorkers)
	logger.Info(msg)

	// Configure max attempts
	maxAttemptsStr := config.GetEnv("MAX_ATTEMPTS", "3")
	maxAttempts, err := strconv.Atoi(maxAttemptsStr)
	if err != nil || maxAttempts < 1 {
		maxAttempts = 3
	}

	// Configure Redis
	redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")
	redisPass := config.GetEnv("REDIS_PASSWORD", "")
	redisDB, _ := strconv.Atoi(config.GetEnv("REDIS_DB", "0"))

	// Set up Redis queue - using the simple constructor
	redisQueue := queue.NewRedisQueue(redisAddr, redisPass, redisDB)
	msg = fmt.Sprintf("Connected to Redis at %s", redisAddr)
	logger.Info(msg)

	// Create worker pool
	workerPool := worker.NewWorkerPool(redisQueue, workerID, numWorkers, maxAttempts)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		msg := fmt.Sprintf("Received shutdown signal: %v", sig)
		logger.Info(msg)
		cancel()
	}()

	// Start the worker pool
	workerPool.Start(ctx)
	msg = "Worker pool started, waiting for jobs..."
	logger.Info(msg)

	// Wait for context cancellation (shutdown signal)
	<-ctx.Done()

	msg = "Shutting down worker service..."
	logger.Info(msg)

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown worker pool
	workerPool.Shutdown()

	// Close Redis connection
	if err := redisQueue.Close(); err != nil {
		errMsg := fmt.Sprintf("Error closing Redis connection: %v", err)
		logger.Error(errMsg)
	}

	<-shutdownCtx.Done()
	msg = "Worker service shutdown complete"
	logger.Info(msg)
}
