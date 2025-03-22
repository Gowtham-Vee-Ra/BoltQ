// cmd/api/main.go
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
	"BoltQ/internal/queue"
	"BoltQ/pkg/config"
	"BoltQ/pkg/logger"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	godotenv.Load()

	// Configure logging
	logger.Setup(os.Getenv("LOG_LEVEL"))
	msg := "Starting Job API Service..."
	logger.Info(msg)

	// Configure API port
	port := config.GetEnv("API_PORT", "8080")

	// Configure Redis
	redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")
	redisPass := config.GetEnv("REDIS_PASSWORD", "")
	redisDB, _ := strconv.Atoi(config.GetEnv("REDIS_DB", "0"))

	// Set up Redis queue - using the simple constructor which matches our implementation
	redisQueue := queue.NewRedisQueue(redisAddr, redisPass, redisDB)
	connectedMsg := fmt.Sprintf("Connected to Redis at %s", redisAddr)
	logger.Info(connectedMsg)

	// Create API handler
	apiHandler := api.NewAPI(redisQueue)

	// Set up router
	router := mux.NewRouter()
	apiHandler.SetupRoutes(router)

	// Set up HTTP server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Run server in a goroutine so it doesn't block shutdown handling
	go func() {
		listeningMsg := fmt.Sprintf("API server listening on port %s", port)
		logger.Info(listeningMsg)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errorMsg := fmt.Sprintf("Server error: %v", err)
			logger.Error(errorMsg)
			os.Exit(1)
		}
	}()

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a shutdown signal
	<-quit
	msgShutdown := "Shutting down API server..."
	logger.Info(msgShutdown)

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		errorMsg := fmt.Sprintf("Server forced to shutdown: %v", err)
		logger.Error(errorMsg)
	}

	// Close Redis connection
	if err := redisQueue.Close(); err != nil {
		errorMsg := fmt.Sprintf("Error closing Redis connection: %v", err)
		logger.Error(errorMsg)
	}

	msgComplete := "API server shutdown complete"
	logger.Info(msgComplete)
}
