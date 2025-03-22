// internal/worker/processor.go
package worker

import (
	"encoding/json"
	"fmt"
	"time"

	"BoltQ/pkg/logger"
)

// JobProcessorFunc defines a function type for processing jobs
type JobProcessorFunc func(payload string) error

// defaultProcessors maps job types to their processor functions
var defaultProcessors = map[string]JobProcessorFunc{
	"email":        processEmailJob,
	"notification": processNotificationJob,
	"report":       processReportJob,
	"default":      processDefaultJob,
}

// ProcessTask processes a job payload
func ProcessTask(jobType, payload string) error {
	// Log the processing start
	logger.Info(fmt.Sprintf("Processing task type: %s", jobType))

	// Find the appropriate processor for this job type
	processor, exists := defaultProcessors[jobType]
	if !exists {
		// Use default processor if no specific one exists
		processor = defaultProcessors["default"]
	}

	// Process the job with timing
	startTime := time.Now()
	err := processor(payload)
	duration := time.Since(startTime)

	// Log the result
	if err != nil {
		logger.Error(fmt.Sprintf("Task processing failed. Type: %s, Duration: %s, Error: %v",
			jobType, duration.String(), err))
		return err
	}

	logger.Info(fmt.Sprintf("Task processed successfully. Type: %s, Duration: %s",
		jobType, duration.String()))

	return nil
}

// RegisterProcessor registers a custom processor for a job type
func RegisterProcessor(jobType string, processor JobProcessorFunc) {
	defaultProcessors[jobType] = processor
}

// processEmailJob processes email jobs
func processEmailJob(payload string) error {
	// Parse payload
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return fmt.Errorf("failed to parse email payload: %v", err)
	}

	// Simulate email sending
	time.Sleep(500 * time.Millisecond)

	// In a real implementation, this would connect to an email service
	return nil
}

// processNotificationJob processes notification jobs
func processNotificationJob(payload string) error {
	// Parse payload
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return fmt.Errorf("failed to parse notification payload: %v", err)
	}

	// Simulate notification sending
	time.Sleep(200 * time.Millisecond)

	// In a real implementation, this would connect to a notification service
	return nil
}

// processReportJob processes report generation jobs
func processReportJob(payload string) error {
	// Parse payload
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return fmt.Errorf("failed to parse report payload: %v", err)
	}

	// Simulate report generation (longer task)
	time.Sleep(1 * time.Second)

	// In a real implementation, this would generate and store a report
	return nil
}

// processDefaultJob is a fallback processor for unknown job types
func processDefaultJob(payload string) error {
	// For unknown job types, just log and pass through
	logger.Warn(fmt.Sprintf("Using default processor for unknown job type with payload: %s", payload))

	// Simulate generic processing
	time.Sleep(300 * time.Millisecond)

	return nil
}
