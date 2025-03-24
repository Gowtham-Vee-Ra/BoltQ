// internal/worker/worker.go
package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"BoltQ/internal/queue"
	"BoltQ/pkg/logger"
)

// Worker represents a job processor
type Worker struct {
	ID         string
	queue      queue.Queue
	processors map[string]func(ctx context.Context, j *queue.Job) error
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	logger     logger.Logger
}

// NewWorker creates a new worker
func NewWorker(id string, q queue.Queue, log logger.Logger) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	return &Worker{
		ID:         id,
		queue:      q,
		processors: make(map[string]func(ctx context.Context, j *queue.Job) error),
		ctx:        ctx,
		cancel:     cancel,
		logger:     log,
	}
}

// RegisterProcessor registers a processor for a specific job type
func (w *Worker) RegisterProcessor(jobType string, processor func(ctx context.Context, j *queue.Job) error) {
	w.processors[jobType] = processor
	w.logger.Info(fmt.Sprintf("Registered processor for job type: %s", jobType), nil)
}

// Start begins the worker processing loop
func (w *Worker) Start() {
	w.logger.Info("Worker started", nil)

	w.wg.Add(1)
	go w.processLoop()
}

// Stop signals the worker to shut down
func (w *Worker) Stop() {
	w.logger.Info("Stopping worker", nil)
	w.cancel()
	w.wg.Wait()
	w.logger.Info("Worker stopped", nil)
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
	ctx, cancel := context.WithTimeout(w.ctx, 30*time.Second)
	defer cancel()

	j, err := w.queue.Consume(ctx)
	if err != nil {
		// Skip logging if no jobs are available (common case)
		if err.Error() != "no jobs available" {
			w.logger.Error("Error consuming job", map[string]interface{}{
				"error": err.Error(),
			})
		}
		return
	}

	if j == nil {
		// No job available
		return
	}

	w.processJob(ctx, j)
}

// processJob handles the processing of a job
func (w *Worker) processJob(ctx context.Context, j *queue.Job) {
	// Log processing start
	w.logger.Info(fmt.Sprintf("Processing job: %s", j.ID), map[string]interface{}{
		"type":     j.Type,
		"priority": j.Priority,
		"worker":   w.ID,
	})

	// Find the processor for this job type
	processor, exists := w.processors[j.Type]
	if !exists {
		w.logger.Error(fmt.Sprintf("No processor registered for job type: %s", j.Type), map[string]interface{}{
			"job_id": j.ID,
		})

		// Update status to failed - we don't have MoveToDeadLetter
		err := w.queue.UpdateStatus(ctx, j.ID, queue.StatusFailed, fmt.Errorf("no processor registered"))
		if err != nil {
			w.logger.Error("Failed to update job status", map[string]interface{}{
				"error":  err.Error(),
				"job_id": j.ID,
			})
		}
		return
	}

	// Process the job
	err := processor(ctx, j)
	if err != nil {
		w.logger.Error(fmt.Sprintf("Job processing failed: %s", j.ID), map[string]interface{}{
			"error":    err.Error(),
			"attempts": j.Attempts,
		})

		// Check if we should retry
		if j.Attempts < 3 { // Default max attempts
			// Instead of custom retry, update status to "retrying"
			err = w.queue.UpdateStatus(ctx, j.ID, queue.StatusRetrying, err)
			if err != nil {
				w.logger.Error("Failed to update job status to retrying", map[string]interface{}{
					"error":  err.Error(),
					"job_id": j.ID,
				})
			}
		} else {
			// Update status to failed
			err = w.queue.UpdateStatus(ctx, j.ID, queue.StatusFailed, err)
			if err != nil {
				w.logger.Error("Failed to update job status to failed", map[string]interface{}{
					"error":  err.Error(),
					"job_id": j.ID,
				})
			}
		}
		return
	}

	// Job completed successfully
	w.logger.Info(fmt.Sprintf("Job completed successfully: %s", j.ID), nil)

	// Update job status to completed
	err = w.queue.UpdateStatus(ctx, j.ID, queue.StatusCompleted, nil)
	if err != nil {
		w.logger.Error("Failed to update job status to completed", map[string]interface{}{
			"error":  err.Error(),
			"job_id": j.ID,
		})
	}
}

// RegisterDefaultProcessors registers basic processors for common job types
func (w *Worker) RegisterDefaultProcessors() {
	// Email processor
	w.RegisterProcessor("email", func(ctx context.Context, j *queue.Job) error {
		recipient, ok := j.Payload["recipient"].(string)
		if !ok {
			return fmt.Errorf("invalid recipient")
		}

		subject, ok := j.Payload["subject"].(string)
		if !ok {
			return fmt.Errorf("invalid subject")
		}

		w.logger.Info("Processing email job", map[string]interface{}{
			"job_id":    j.ID,
			"recipient": recipient,
			"subject":   subject,
		})

		// Simulate email sending
		time.Sleep(500 * time.Millisecond)

		w.logger.Info("Email job processed successfully", map[string]interface{}{
			"job_id": j.ID,
		})
		return nil
	})

	// Notification processor
	w.RegisterProcessor("notification", func(ctx context.Context, j *queue.Job) error {
		w.logger.Info("Processing notification job", map[string]interface{}{
			"job_id": j.ID,
		})

		// Simulate notification sending
		time.Sleep(200 * time.Millisecond)

		w.logger.Info("Notification job processed successfully", map[string]interface{}{
			"job_id": j.ID,
		})
		return nil
	})

	// Test processor
	w.RegisterProcessor("test", func(ctx context.Context, j *queue.Job) error {
		w.logger.Info("Processing test job", map[string]interface{}{
			"job_id": j.ID,
		})

		// Simulate work
		time.Sleep(100 * time.Millisecond)

		w.logger.Info("Test job processed successfully", map[string]interface{}{
			"job_id": j.ID,
		})
		return nil
	})
}
