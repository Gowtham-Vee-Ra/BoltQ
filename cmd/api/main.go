package main

import (
	"fmt"
	"log"
	"net/http"

	"BoltQ/internal/api"
	"BoltQ/internal/queue"
	"BoltQ/pkg/config"
	"BoltQ/pkg/logger"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found or error loading it")
	}

	// Initialize the Redis queue
	redisQueue := queue.NewRedisQueue()

	// Create API handlers with the queue
	jobHandler := api.NewJobHandler(redisQueue)

	// Set up routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("BoltQ API Running!"))
	})

	http.HandleFunc("/submit", jobHandler.SubmitJobHandler)
	http.HandleFunc("/status", jobHandler.JobStatusHandler)

	// Get port from environment or use default
	port := config.GetEnv("PORT", "8080")

	// Start the server
	serverAddr := fmt.Sprintf(":%s", port)
	msg := fmt.Sprintf("Starting Job API Service on port %s...", port)
	logger.Info(msg)

	log.Fatal(http.ListenAndServe(serverAddr, nil))
}
