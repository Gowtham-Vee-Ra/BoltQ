package worker

import (
	"encoding/json"
	"fmt"
	"time"

	"BoltQ/pkg/logger"
)

// Job represents a task to be processed
type Job struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Payload  map[string]interface{} `json:"payload"`
	Priority int                    `json:"priority,omitempty"`
}

// ProcessTask handles the execution of a task
func ProcessTask(taskJSON *string) {
	// Parse the job
	var job Job
	if err := json.Unmarshal([]byte(*taskJSON), &job); err != nil {
		errorMsg := fmt.Sprintf("Error parsing job: %v", err)
		logger.Error(errorMsg)
		return
	}

	// Log the start of processing
	startMsg := fmt.Sprintf("Started processing job: %s", job.ID)
	logger.Info(startMsg)

	// Process different job types
	switch job.Type {
	case "email":
		processEmailJob(job)
	case "notification":
		processNotificationJob(job)
	case "data_processing":
		processDataJob(job)
	default:
		// Process generic job type
		processGenericJob(job)
	}

	// Log completion
	completeMsg := fmt.Sprintf("Job completed: %s", job.ID)
	logger.Info(completeMsg)
}

// Process an email job
func processEmailJob(job Job) {
	recipient, _ := job.Payload["recipient"].(string)
	fmt.Printf("Sending email to %s\n", recipient)
	// Simulate work
	time.Sleep(1 * time.Second)
}

// Process a notification job
func processNotificationJob(job Job) {
	userID, _ := job.Payload["user_id"].(string)
	message, _ := job.Payload["message"].(string)
	fmt.Printf("Sending notification to user %s: %s\n", userID, message)
	// Simulate work
	time.Sleep(500 * time.Millisecond)
}

// Process a data processing job
func processDataJob(job Job) {
	fmt.Println("Processing data job")
	// Simulate heavier work
	time.Sleep(2 * time.Second)
}

// Process a generic job
func processGenericJob(job Job) {
	fmt.Printf("Processing generic job: %s\n", job.ID)
	// Simulate work
	time.Sleep(1 * time.Second)
}
