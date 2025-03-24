package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"BoltQ/internal/api"
	"BoltQ/internal/job"
	"BoltQ/internal/queue"
	"BoltQ/internal/worker"
	"BoltQ/pkg/config"
	"BoltQ/pkg/logger"
	"BoltQ/pkg/metrics"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Initialize logger
	log := logger.NewLogger("worker")
	log.Info("Starting BoltQ Worker Service...")

	// Load configuration
	numWorkersStr := config.GetEnv("NUM_WORKERS", "4")
	metricsPort := config.GetEnv("METRICS_PORT", "9091")
	redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")

	// Parse number of workers
	numWorkers, err := strconv.Atoi(numWorkersStr)
	if err != nil {
		log.Error(fmt.Sprintf("Invalid NUM_WORKERS value: %v", err))
		numWorkers = 4
	}

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Ping Redis to make sure it's available
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Error(fmt.Sprintf("Failed to connect to Redis: %v", err))
		os.Exit(1)
	}
	log.Info(fmt.Sprintf("Connected to Redis at %s", redisAddr))

	// Initialize metrics collector
	metricsCollector := metrics.NewMetricsCollector("worker")

	// Initialize queue
	redisQueue := queue.NewRedisQueue(redisClient, log)

	// Initialize workflow manager
	workflowManager := job.NewWorkflowManager(redisClient, log)

	// Initialize WebSocket handler for publishing job updates
	websocketManager := api.NewWebSocketManager(redisClient, log)

	// Initialize error handler
	errorHandler := worker.NewErrorHandler(redisQueue, log, metricsCollector)

	// Initialize worker pool
	workerPool := worker.NewWorkerPool(
		redisQueue,
		log,
		metricsCollector,
		errorHandler,
		workflowManager,
		websocketManager,
		numWorkers,
		100*time.Millisecond,
	)

	// Register job processors
	registerJobProcessors(workerPool)

	// Initialize delayed job processor
	delayedProcessor := worker.NewDelayedJobProcessor(redisQueue, log, metricsCollector)

	// Metrics server
	metricsRouter := mux.NewRouter()
	metricsRouter.Handle("/metrics", promhttp.Handler())
	metricsRouter.HandleFunc("/health", healthCheckHandler)

	metricsServer := &http.Server{
		Addr:    ":" + metricsPort,
		Handler: metricsRouter,
	}

	// Start delayed job processor
	delayedProcessor.Start(5 * time.Second)

	// Start worker pool
	workerPool.Start()

	// Run metrics server in goroutine
	go func() {
		log.Info(fmt.Sprintf("Metrics server listening on port %s", metricsPort))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(fmt.Sprintf("Error starting metrics server: %v", err))
		}
	}()

	// Wait for interrupt signal to gracefully shut down
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info("Shutting down...")

	// Stop the worker pool
	workerPool.Stop()

	// Stop the delayed job processor
	delayedProcessor.Stop()

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown metrics server
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Error(fmt.Sprintf("Metrics server shutdown error: %v", err))
	}

	log.Info("Worker service stopped")
}

// Health check handler
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Register job processors
func registerJobProcessors(workerPool *worker.WorkerPool) {
	// Example processor for "echo" jobs
	workerPool.RegisterProcessor("echo", func(ctx context.Context, task *queue.Task) (map[string]interface{}, error) {
		// Simulate some work
		time.Sleep(1 * time.Second)

		// Echo back the input data
		result := map[string]interface{}{
			"echo":      task.Data,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		return result, nil
	})

	// Example processor for "sleep" jobs
	workerPool.RegisterProcessor("sleep", func(ctx context.Context, task *queue.Task) (map[string]interface{}, error) {
		// Get sleep duration from task data
		var seconds float64 = 5 // Default to 5 seconds

		if durationVal, ok := task.Data["seconds"]; ok {
			switch v := durationVal.(type) {
			case float64:
				seconds = v
			case int:
				seconds = float64(v)
			case string:
				parsedSeconds, err := strconv.ParseFloat(v, 64)
				if err == nil {
					seconds = parsedSeconds
				}
			}
		}

		// Limit max sleep time
		if seconds > 60 {
			seconds = 60
		}

		// Sleep for the specified duration
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(seconds * float64(time.Second))):
			// Return result
			return map[string]interface{}{
				"slept_for":    seconds,
				"completed_at": time.Now().Format(time.RFC3339),
			}, nil
		}
	})

	// Add more job processors as needed
}
