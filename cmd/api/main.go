// cmd/api/main.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"BoltQ/internal/api"
	"BoltQ/internal/job"
	"BoltQ/internal/queue"
	"BoltQ/pkg/config"
	"BoltQ/pkg/logger"
	"BoltQ/pkg/metrics"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Initialize logger
	log := logger.NewLogger("api")
	log.Info("Starting BoltQ API Service...")

	// Load configuration
	apiPort := config.GetEnv("API_PORT", "8080")
	metricsPort := config.GetEnv("METRICS_PORT", "9090")
	redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")

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
	metricsCollector := metrics.NewMetricsCollector("api")

	// Initialize queue
	redisQueue := queue.NewRedisQueue(redisClient, log)

	// Initialize workflow manager
	workflowManager := job.NewWorkflowManager(redisClient, log)

	// Initialize WebSocket manager
	websocketManager := api.NewWebSocketManager(redisClient, log)
	websocketManager.Start()

	// Initialize API handler
	apiHandler := api.NewHandler(redisQueue, log, metricsCollector, workflowManager)

	// Create router
	router := mux.NewRouter()

	// Register API routes
	apiHandler.RegisterRoutes(router)

	// Register WebSocket route
	router.HandleFunc("/ws/jobs", websocketManager.HandleJobUpdatesWebSocket)

	// API server with graceful shutdown
	apiServer := &http.Server{
		Addr:         ":" + apiPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Metrics server
	metricsRouter := mux.NewRouter()
	metricsRouter.Handle("/metrics", promhttp.Handler())

	metricsServer := &http.Server{
		Addr:    ":" + metricsPort,
		Handler: metricsRouter,
	}

	// Run API server in goroutine
	go func() {
		log.Info(fmt.Sprintf("API server listening on port %s", apiPort))
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(fmt.Sprintf("Error starting API server: %v", err))
			os.Exit(1)
		}
	}()

	// Run metrics server in goroutine
	go func() {
		log.Info(fmt.Sprintf("Metrics server listening on port %s", metricsPort))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(fmt.Sprintf("Error starting metrics server: %v", err))
		}
	}()

	// Wait for interrupt signal to gracefully shut down the servers
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info("Shutting down servers...")

	// Create shutdown contexts with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown API server
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		log.Error(fmt.Sprintf("API server shutdown error: %v", err))
	}

	// Shutdown metrics server
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Error(fmt.Sprintf("Metrics server shutdown error: %v", err))
	}

	// Stop WebSocket manager
	websocketManager.Stop()

	log.Info("Servers stopped")
}
