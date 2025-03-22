// internal/worker/worker.go
package worker

import (
	"fmt"
	"time"

	"BoltQ/internal/job"
	"BoltQ/internal/queue"
	"BoltQ/pkg/logger"
)

// Worker represents a job processor
type Worker struct {
	ID         string
	queue      *queue.RedisQueue
	processors map[string]JobProcessor
	shutdown   chan bool
}

// JobProcessor is a function that processes a specific type of job
type JobProcessor func(*job.Job) error

// NewWorker creates a new worker
func NewWorker(id string, q *queue.RedisQueue) *Worker {
	return &Worker{
		ID:         id,
		queue:      q,
		processors: make(map[string]JobProcessor),
		shutdown:   make(chan bool),
	}
}

// RegisterProcessor registers a processor for a specific job type
func (w *Worker) RegisterProcessor(jobType string, processor JobProcessor) {
	w.processors[jobType] = processor
}

// Start begins the worker processing loop
func (w *Worker) Start() {
	msg := fmt.Sprintf("Worker %s started", w.ID)
	logger.Info(msg)

	go w.processLoop()
}

// Stop signals the worker to shut down
func (w *Worker) Stop() {
	w.shutdown <- true
	msg := fmt.Sprintf("Worker %s stopped", w.ID)
	logger.Info(msg)
}

// processLoop continuously polls for jobs and processes them
func (w *Worker) processLoop() {
	for {
		select {
		case <-w.shutdown:
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
		errMsg := fmt.Sprintf("Error consuming job: %v", err)
		logger.Error(errMsg)
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
	startMsg := fmt.Sprintf("Worker %s processing job %s (type: %s)", w.ID, j.ID, j.Type)
	logger.Info(startMsg)

	// Mark job as running
	j.MarkRunning()
	j.WorkerID = w.ID
	err := w.queue.UpdateJobStatus(j)
	if err != nil {
		errMsg := fmt.Sprintf("Error updating job status: %v", err)
		logger.Error(errMsg)
	}

	// Find the processor for this job type
	processor, exists := w.processors[j.Type]
	if !exists {
		errMsg := fmt.Sprintf("No processor registered for job type: %s", j.Type)
		logger.Error(errMsg)
		j.MarkFailed(errMsg)
		w.handleFailedJob(j, fmt.Errorf(errMsg))
		return
	}

	// Process the job
	err = processor(j)
	if err != nil {
		errMsg := fmt.Sprintf("Job %s failed: %v", j.ID, err)
		logger.Error(errMsg)
		j.MarkFailed(err.Error())
		w.handleFailedJob(j, err)
		return
	}

	// Job succeeded
	j.MarkCompleted()
	err = w.queue.UpdateJobStatus(j)
	if err != nil {
		errMsg := fmt.Sprintf("Error updating job status: %v", err)
		logger.Error(errMsg)
	}

	completeMsg := fmt.Sprintf("Worker %s completed job %s", w.ID, j.ID)
	logger.Info(completeMsg)
}

// handleFailedJob determines what to do with a failed job
func (w *Worker) handleFailedJob(j *job.Job, err error) {
	if j.ShouldRetry() {
		retryMsg := fmt.Sprintf("Scheduling job %s for retry (%d/%d)", j.ID, j.Attempts, j.MaxAttempts)
		logger.Info(retryMsg)

		retryErr := w.queue.RetryJob(j)
		if retryErr != nil {
			errMsg := fmt.Sprintf("Error scheduling retry: %v", retryErr)
			logger.Error(errMsg)
		}
	} else {
		deadLetterMsg := fmt.Sprintf("Moving job %s to dead letter queue after %d failed attempts", j.ID, j.Attempts)
		logger.Info(deadLetterMsg)

		// Update status and move to dead letter queue
		j.MarkDeadLetter()
		dlqErr := w.queue.RetryJob(j)
		if dlqErr != nil {
			errMsg := fmt.Sprintf("Error moving to dead letter queue: %v", dlqErr)
			logger.Error(errMsg)
		}
	}
}

// RegisterDefaultProcessors registers basic processors for common job types
func (w *Worker) RegisterDefaultProcessors() {
	// Email processor
	w.RegisterProcessor("email", func(j *job.Job) error {
		logger.Info(fmt.Sprintf("Processing email job: %s", j.Payload))
		// Simulate email sending
		time.Sleep(500 * time.Millisecond)
		return nil
	})

	// Notification processor
	w.RegisterProcessor("notification", func(j *job.Job) error {
		logger.Info(fmt.Sprintf("Processing notification job: %s", j.Payload))
		// Simulate notification
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	// Generic processor for testing
	w.RegisterProcessor("test", func(j *job.Job) error {
		logger.Info(fmt.Sprintf("Processing test job: %s", j.Payload))
		// Simulate work
		time.Sleep(100 * time.Millisecond)
		return nil
	})
}
