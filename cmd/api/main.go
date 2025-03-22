// cmd/api/main.go
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"BoltQ/internal/api"
	"BoltQ/internal/queue"
	"BoltQ/pkg/config"
	"BoltQ/pkg/logger"
	"BoltQ/pkg/metrics"
	"BoltQ/pkg/tracing"
)

func main() {
	log := logger.NewLogger("api-service")
	log.Info("Starting Job API Service")

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize tracing
	shutdownTracer, err := tracing.InitTracer(ctx, "boltq-api")
	if err != nil {
		log.Error("Failed to initialize tracer", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		defer shutdownTracer()
	}

	// Get configuration
	redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")
	apiPort := config.GetEnv("API_PORT", "8080")
	metricsPort := config.GetEnv("METRICS_PORT", "9090")

	// Setup Redis queue
	redisQueue := queue.NewRedisQueue(ctx, redisAddr)

	// Setup API handler
	handler := api.NewHandler(redisQueue)

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Job API Running!"))
	})
	mux.HandleFunc("/jobs/submit", handler.SubmitJobHandler)
	mux.HandleFunc("/jobs/status", handler.GetJobStatusHandler)
	mux.HandleFunc("/queue/stats", handler.GetQueueStatsHandler)

	// Setup metrics server
	metrics.SetupMetricsServer(":" + metricsPort)
	log.Info("Metrics server started", map[string]interface{}{
		"port": metricsPort,
	})

	// Start HTTP server
	server := &http.Server{
		Addr:    ":" + apiPort,
		Handler: mux,
	}

	// Start server in a goroutine
	go func() {
		log.Info("API server listening", map[string]interface{}{
			"port": apiPort,
		})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", map[string]interface{}{
				"error": err.Error(),
			})
			cancel()
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("Shutting down API service")

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", map[string]interface{}{
			"error": err.Error(),
		})
	}

	log.Info("API service stopped")
}
