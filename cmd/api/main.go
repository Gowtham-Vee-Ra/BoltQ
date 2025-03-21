package main

import (
	"fmt"
	"net/http"

	"BoltQ/internal/api"
	"BoltQ/internal/queue"
	"BoltQ/pkg/config"
	"BoltQ/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
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
	startupMsg := "Starting Job API Service..."
	logger.Info(&startupMsg)

	// Load configuration
	redisAddr := config.GetEnv("REDIS_ADDR", "localhost:6379")
	apiPort := config.GetEnv("API_PORT", "8080")

	// Initialize Redis queue
	redisQueue := queue.NewRedisQueue(redisAddr)

	// Initialize job handler
	jobHandler := api.NewJobHandler(redisQueue)

	// Set up HTTP routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("BoltQ API Running!"))
	})

	// Job submission endpoint
	http.HandleFunc("/api/jobs", jobHandler.SubmitJobHandler)

	// Job status endpoint
	http.HandleFunc("/api/jobs/status", jobHandler.GetJobStatusHandler)

	// Queue stats endpoint
	http.HandleFunc("/api/stats", jobHandler.GetQueueStatsHandler)

	// Start the server
	serverMsg := fmt.Sprintf("API server listening on port %s", apiPort)
	logger.Info(&serverMsg)

	err = http.ListenAndServe(":"+apiPort, nil)
	if err != nil {
		errMsg := fmt.Sprintf("Error starting server: %v", err)
		logger.Error(&errMsg)
		panic(err)
	}
}
