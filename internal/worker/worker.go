// internal/worker/worker.go
package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"BoltQ/internal/job"
	"BoltQ/internal/queue"
	"BoltQ/pkg/logger"
	"BoltQ/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"
)

// Worker represents a job processor
type Worker struct {
	ID         string
	queue      queue.Queue
	processors map[string]func(ctx context.Context, j *job.Job) error // Using anonymous function type instead
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	logger     *logger.Logger
}

// NewWorker creates a new worker
func NewWorker(id string, q queue.Queue) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	return &Worker{
		ID:         id,
		queue:      q,
		processors: make(map[string]func(ctx context.Context, j *job.Job) error),
		ctx:        ctx,
		cancel:     cancel,
		logger:     logger.NewLogger(fmt.Sprintf("worker-%s", id)),
	}
}

// RegisterProcessor registers a processor for a specific job type
func (w *Worker) RegisterProcessor(jobType string, processor func(ctx context.Context, j *job.Job) error) {
	w.processors[jobType] = processor
	w.logger.Info(fmt.Sprintf("Registered processor for job type: %s", jobType))
}

// Start begins the worker processing loop
func (w *Worker) Start() {
	w.logger.Info("Worker started")

	w.wg.Add(1)
	go w.processLoop()
}

// Stop signals the worker to shut down
func (w *Worker) Stop() {
	w.logger.Info("Stopping worker")
	w.cancel()
	w.wg.Wait()
	w.logger.Info("Worker stopped")
}

// processLoop continuously polls for jobs and processes them
func (w *Worker) processLoop() {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			w.processSingleJob()
			time.Sleep(100 * time.Millisecond) // Small delay to prevent CPU spinning
		}
	}
}

// processSingleJob processes a single job from the queue
func (w *Worker) processSingleJob() {
	j, err := w.queue.ConsumeJob()
	if err != nil {
		w.logger.Error("Error consuming job", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	if j == nil {
		// No job available
		return
	}

	w.processJob(j)
}

// processJob handles the processing of a job
func (w *Worker) processJob(j *job.Job) {
	// Start metrics tracking
	metrics.ActiveWorkers.Inc()
	defer metrics.ActiveWorkers.Dec()

	// Create process timer
	timer := prometheus.NewTimer(metrics.JobProcessingTime.WithLabelValues(j.Type))
	defer timer.ObserveDuration()

	w.logger.WithJob(j.ID, "Processing job", map[string]interface{}{
		"type":     j.Type,
		"priority": string(j.Priority),
		"worker":   w.ID,
	})

	// Find the processor for this job type
	processor, exists := w.processors[j.Type]
	if !exists {
		w.logger.JobError(j.ID, "No processor registered for job type", map[string]interface{}{
			"type": j.Type,
		})

		// Move to dead letter queue
		err := w.queue.MoveToDeadLetter(j)
		if err != nil {
			w.logger.JobError(j.ID, "Failed to move job to dead letter queue", map[string]interface{}{
				"error": err.Error(),
			})
		}
		return
	}

	// Process the job
	err := processor(w.ctx, j)
	if err != nil {
		w.logger.JobError(j.ID, "Job processing failed", map[string]interface{}{
			"error":    err.Error(),
			"attempts": j.Attempts,
		})

		// Check if we should retry
		if j.Attempts < 3 { // Default max attempts
			// Move to retry queue
			err = w.queue.RetryJob(j)
			if err != nil {
				w.logger.JobError(j.ID, "Failed to move job to retry queue", map[string]interface{}{
					"error": err.Error(),
				})
			}
		} else {
			// Move to dead letter queue
			err = w.queue.MoveToDeadLetter(j)
			if err != nil {
				w.logger.JobError(j.ID, "Failed to move job to dead letter queue", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
		return
	}

	// Job completed successfully
	w.logger.WithJob(j.ID, "Job completed successfully")

	// Update job status to completed
	err = w.queue.UpdateJobStatus(j.ID, job.StatusCompleted)
	if err != nil {
		w.logger.JobError(j.ID, "Failed to update job status to completed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	metrics.JobsProcessed.WithLabelValues(j.Type, "completed").Inc()
}

// RegisterDefaultProcessors registers basic processors for common job types
func (w *Worker) RegisterDefaultProcessors() {
	// Email processor
	w.RegisterProcessor("email", func(ctx context.Context, j *job.Job) error {
		log := logger.NewLogger("email-processor")
		log.WithJob(j.ID, "Processing email job", map[string]interface{}{
			"data": j.Data,
		})

		// Simulate email sending
		time.Sleep(500 * time.Millisecond)

		log.WithJob(j.ID, "Email job processed successfully")
		return nil
	})

	// Notification processor
	w.RegisterProcessor("notification", func(ctx context.Context, j *job.Job) error {
		log := logger.NewLogger("notification-processor")
		log.WithJob(j.ID, "Processing notification job", map[string]interface{}{
			"data": j.Data,
		})

		// Simulate notification sending
		time.Sleep(200 * time.Millisecond)

		log.WithJob(j.ID, "Notification job processed successfully")
		return nil
	})

	// Test processor
	w.RegisterProcessor("test", func(ctx context.Context, j *job.Job) error {
		log := logger.NewLogger("test-processor")
		log.WithJob(j.ID, "Processing test job", map[string]interface{}{
			"data": j.Data,
		})

		// Simulate work
		time.Sleep(100 * time.Millisecond)

		log.WithJob(j.ID, "Test job processed successfully")
		return nil
	})
}
