// internal/worker/pool.go
package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"BoltQ/internal/job"
	"BoltQ/internal/queue"
	"BoltQ/pkg/logger"
)

// WorkerPool manages a pool of workers for processing jobs
type WorkerPool struct {
	queue        *queue.RedisQueue
	numWorkers   int
	workerID     string
	maxAttempts  int
	shutdownChan chan struct{}
	wg           sync.WaitGroup
	jobChan      chan string
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(q *queue.RedisQueue, workerID string, numWorkers int, maxAttempts int) *WorkerPool {
	return &WorkerPool{
		queue:        q,
		numWorkers:   numWorkers,
		workerID:     workerID,
		maxAttempts:  maxAttempts,
		shutdownChan: make(chan struct{}),
		jobChan:      make(chan string, numWorkers*2), // Buffer to avoid blocking
	}
}

// Start initializes and starts the worker pool
func (wp *WorkerPool) Start(ctx context.Context) {
	msg := fmt.Sprintf("Starting worker pool with %d workers", wp.numWorkers)
	logger.Info(msg)

	// Start the job fetcher goroutine
	wp.wg.Add(1)
	go wp.fetchJobs(ctx)

	// Start worker goroutines
	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		workerNum := i + 1
		go wp.runWorker(ctx, workerNum)
	}
}

// fetchJobs continuously fetches jobs from the queue and sends them to workers
func (wp *WorkerPool) fetchJobs(ctx context.Context) {
	defer wp.wg.Done()

	for {
		select {
		case <-ctx.Done():
			close(wp.jobChan)
			return
		case <-wp.shutdownChan:
			close(wp.jobChan)
			return
		default:
			// First check the retry queue (higher priority)
			taskStr, err := wp.queue.ConsumeRetry()

			// If no retry tasks, check the main queue
			if err != nil || taskStr == "" {
				taskStr, err = wp.queue.Consume()
			}

			if err != nil {
				// No jobs available or error
				time.Sleep(100 * time.Millisecond) // Avoid busy waiting
				continue
			}

			if taskStr != "" {
				select {
				case wp.jobChan <- taskStr:
					// Successfully sent to a worker
				case <-ctx.Done():
					// Context was canceled while trying to send job
					// Re-queue the job we just got
					wp.queue.Publish(taskStr)
					close(wp.jobChan)
					return
				case <-wp.shutdownChan:
					// Shutdown requested while trying to send job
					// Re-queue the job we just got
					wp.queue.Publish(taskStr)
					close(wp.jobChan)
					return
				}
			}
		}
	}
}

// runWorker starts a worker that processes jobs from the job channel
func (wp *WorkerPool) runWorker(ctx context.Context, workerNum int) {
	defer wp.wg.Done()

	workerID := fmt.Sprintf("%s-worker-%d", wp.workerID, workerNum)
	msg := fmt.Sprintf("Worker %s started", workerID)
	logger.Info(msg)

	for taskStr := range wp.jobChan {
		// Parse job
		j, err := job.FromJSON(taskStr)
		if err != nil {
			msg := fmt.Sprintf("Worker %s: Failed to parse job: %v", workerID, err)
			logger.Error(msg)
			continue
		}

		// Update job status to running
		j.Status = job.StatusRunning
		j.StartedAt = time.Now()
		j.WorkerID = workerID

		// Save job status
		jobJSON, _ := j.ToJSON()
		wp.queue.SaveJobStatus(j.ID, jobJSON)

		logger.Info(fmt.Sprintf("Worker %s: Processing job: %s", workerID, j.ID))

		// Process the job
		err = ProcessTask(j.Type, j.Payload)

		// Update job status based on processing result
		if err != nil {
			j.Status = job.StatusFailed
			j.Error = err.Error()
			j.Attempts++

			logger.Error(fmt.Sprintf("Worker %s: Job failed: %s, error: %v, attempts: %d/%d",
				workerID, j.ID, err, j.Attempts, wp.maxAttempts))

			// Check if we should retry or move to dead letter
			if j.Attempts >= wp.maxAttempts {
				j.Status = job.StatusDeadLetter
				jobJSON, _ = j.ToJSON()
				wp.queue.SaveJobStatus(j.ID, jobJSON)
				wp.queue.PublishToDeadLetter(jobJSON)
				logger.Error(fmt.Sprintf("Worker %s: Job %s moved to dead letter queue after %d attempts",
					workerID, j.ID, j.Attempts))
			} else {
				j.Status = job.StatusRetrying
				j.NextRetryAt = time.Now().Add(time.Second * time.Duration(1<<j.Attempts)) // Exponential backoff
				jobJSON, _ = j.ToJSON()
				wp.queue.SaveJobStatus(j.ID, jobJSON)
				wp.queue.PublishToRetry(jobJSON)
				logger.Info(fmt.Sprintf("Worker %s: Job %s scheduled for retry in %v",
					workerID, j.ID, time.Second*time.Duration(1<<j.Attempts)))
			}
		} else {
			// Job completed successfully
			j.Status = job.StatusCompleted
			j.CompletedAt = time.Now()
			jobJSON, _ = j.ToJSON()
			wp.queue.SaveJobStatus(j.ID, jobJSON)
			logger.Info(fmt.Sprintf("Worker %s: Job completed successfully: %s", workerID, j.ID))
		}
	}

	logger.Info(fmt.Sprintf("Worker %s shutting down", workerID))
}

// Shutdown gracefully shuts down the worker pool
func (wp *WorkerPool) Shutdown() {
	logger.Info("Worker pool shutting down...")
	close(wp.shutdownChan)
	wp.wg.Wait()
	logger.Info("Worker pool shutdown complete")
}
