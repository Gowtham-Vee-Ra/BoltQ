// internal/worker/processor.go
package worker

import (
	"context"
	"time"

	"BoltQ/internal/job"
	"BoltQ/pkg/logger"
	"BoltQ/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"
)

// ProcessorFunc defines a function that processes a job
type ProcessorFunc func(ctx context.Context, j *job.Job) error

// DefaultProcessors holds processor functions for common job types
var DefaultProcessors = make(map[string]ProcessorFunc)

// InitDefaultProcessors registers all default job processors
func InitDefaultProcessors() {
	DefaultProcessors["email"] = processEmailJob
	DefaultProcessors["notification"] = processNotificationJob
	DefaultProcessors["report"] = processReportJob
	DefaultProcessors["default"] = processDefaultJob
}

// processEmailJob processes email jobs
func processEmailJob(ctx context.Context, j *job.Job) error {
	log := logger.NewLogger("email-processor")

	log.WithJob(j.ID, "Processing email job", map[string]interface{}{
		"data": j.Data,
	})

	// Start timer for metrics
	timer := prometheus.NewTimer(metrics.JobProcessingTime.WithLabelValues("email"))
	defer timer.ObserveDuration()

	// Simulate email sending
	time.Sleep(500 * time.Millisecond)

	log.WithJob(j.ID, "Email job processed successfully")

	// In a real implementation, this would connect to an email service
	return nil
}

// processNotificationJob processes notification jobs
func processNotificationJob(ctx context.Context, j *job.Job) error {
	log := logger.NewLogger("notification-processor")

	log.WithJob(j.ID, "Processing notification job", map[string]interface{}{
		"data": j.Data,
	})

	// Start timer for metrics
	timer := prometheus.NewTimer(metrics.JobProcessingTime.WithLabelValues("notification"))
	defer timer.ObserveDuration()

	// Simulate notification sending
	time.Sleep(200 * time.Millisecond)

	log.WithJob(j.ID, "Notification job processed successfully")

	// In a real implementation, this would connect to a notification service
	return nil
}

// processReportJob processes report generation jobs
func processReportJob(ctx context.Context, j *job.Job) error {
	log := logger.NewLogger("report-processor")

	log.WithJob(j.ID, "Processing report job", map[string]interface{}{
		"data": j.Data,
	})

	// Start timer for metrics
	timer := prometheus.NewTimer(metrics.JobProcessingTime.WithLabelValues("report"))
	defer timer.ObserveDuration()

	// Simulate report generation (longer task)
	time.Sleep(1 * time.Second)

	log.WithJob(j.ID, "Report job processed successfully")

	// In a real implementation, this would generate and store a report
	return nil
}

// processDefaultJob is a fallback processor for unknown job types
func processDefaultJob(ctx context.Context, j *job.Job) error {
	log := logger.NewLogger("default-processor")

	log.Warn("Using default processor for unknown job type", map[string]interface{}{
		"job_id": j.ID,
		"type":   j.Type,
		"data":   j.Data,
	})

	// Start timer for metrics
	timer := prometheus.NewTimer(metrics.JobProcessingTime.WithLabelValues("default"))
	defer timer.ObserveDuration()

	// Simulate generic processing
	time.Sleep(300 * time.Millisecond)

	log.WithJob(j.ID, "Default processing completed")

	return nil
}

// GetProcessorFunc returns an appropriate processor function for a job type
func GetProcessorFunc(jobType string) ProcessorFunc {
	// Initialize processors if not already done
	if len(DefaultProcessors) == 0 {
		InitDefaultProcessors()
	}

	// Look up the processor for this job type
	processor, exists := DefaultProcessors[jobType]
	if !exists {
		// Fall back to default processor
		return DefaultProcessors["default"]
	}

	return processor
}

// RegisterCustomProcessor adds a custom processor function for a job type
func RegisterCustomProcessor(jobType string, processor ProcessorFunc) {
	// Initialize processors if not already done
	if len(DefaultProcessors) == 0 {
		InitDefaultProcessors()
	}

	DefaultProcessors[jobType] = processor
}
