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
	"BoltQ/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"
)

// JobProcessor defines a function to process a job
type JobProcessor func(context.Context, *job.Job) error

// ProcessorRegistry stores job processors by job type
type ProcessorRegistry map[string]JobProcessor

// WorkerPool manages a pool of workers for job processing
type WorkerPool struct {
	queue       queue.Queue
	processors  ProcessorRegistry
	workerCount int
	maxAttempts int
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	logger      *logger.Logger
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(
	ctx context.Context,
	q queue.Queue,
	workerCount int,
	maxAttempts int,
) *WorkerPool {
	if workerCount <= 0 {
		workerCount = 1
	}

	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	poolCtx, cancel := context.WithCancel(ctx)

	log := logger.NewLogger("worker-pool")

	pool := &WorkerPool{
		queue:       q,
		processors:  make(ProcessorRegistry),
		workerCount: workerCount,
		maxAttempts: maxAttempts,
		ctx:         poolCtx,
		cancel:      cancel,
		logger:      log,
	}

	metrics.WorkerPoolSize.Set(float64(workerCount))

	return pool
}

// RegisterProcessor registers a job processor for a specific job type
func (p *WorkerPool) RegisterProcessor(jobType string, processor JobProcessor) {
	p.processors[jobType] = processor
	p.logger.Info("Registered processor", map[string]interface{}{
		"job_type": jobType,
	})
}

// Start starts the worker pool
func (p *WorkerPool) Start() {
	p.logger.Info("Starting worker pool", map[string]interface{}{
		"worker_count": p.workerCount,
	})

	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.startWorker(i)
	}
}

// Stop stops the worker pool
func (p *WorkerPool) Stop() {
	p.logger.Info("Stopping worker pool")
	p.cancel()
	p.wg.Wait()
	p.logger.Info("Worker pool stopped")
}

// startWorker starts a worker routine
func (p *WorkerPool) startWorker(id int) {
	defer p.wg.Done()

	workerLogger := logger.NewLogger(fmt.Sprintf("worker-%d", id))
	workerLogger.Info("Worker started")

	for {
		select {
		case <-p.ctx.Done():
			workerLogger.Info("Worker stopping")
			return
		default:
			// Try to get a job from the queue
			j, err := p.queue.ConsumeJob()
			if err != nil {
				workerLogger.Error("Error consuming job", map[string]interface{}{
					"error": err.Error(),
				})
				time.Sleep(1 * time.Second)
				continue
			}

			if j == nil {
				// No job available
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Process the job
			p.processJob(j, workerLogger)
		}
	}
}

// processJob processes a single job
func (p *WorkerPool) processJob(j *job.Job, workerLogger *logger.Logger) {
	jobLogger := logger.NewLogger("job-processor")

	metrics.ActiveWorkers.Inc()
	defer metrics.ActiveWorkers.Dec()

	jobLogger.WithJob(j.ID, "Processing job", map[string]interface{}{
		"type":     j.Type,
		"priority": string(j.Priority),
		"attempts": j.Attempts,
	})

	// Find the appropriate processor
	processor, exists := p.processors[j.Type]
	if !exists {
		jobLogger.JobError(j.ID, "No processor registered for job type", map[string]interface{}{
			"type": j.Type,
		})

		// Move to dead letter queue if no processor is registered
		err := p.queue.MoveToDeadLetter(j)
		if err != nil {
			jobLogger.JobError(j.ID, "Failed to move job to dead letter queue", map[string]interface{}{
				"error": err.Error(),
			})
		}
		return
	}

	// Process the job with timing
	timer := prometheus.NewTimer(metrics.JobProcessingTime.WithLabelValues(j.Type))
	err := processor(p.ctx, j)
	timer.ObserveDuration()

	if err != nil {
		jobLogger.JobError(j.ID, "Job processing failed", map[string]interface{}{
			"error":    err.Error(),
			"attempts": j.Attempts,
		})

		// Check if we should retry
		if j.Attempts < p.maxAttempts {
			// Move to retry queue
			err = p.queue.RetryJob(j)
			if err != nil {
				jobLogger.JobError(j.ID, "Failed to move job to retry queue", map[string]interface{}{
					"error": err.Error(),
				})
			}
		} else {
			// Move to dead letter queue
			err = p.queue.MoveToDeadLetter(j)
			if err != nil {
				jobLogger.JobError(j.ID, "Failed to move job to dead letter queue", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
		return
	}

	// Job completed successfully
	jobLogger.WithJob(j.ID, "Job completed successfully", map[string]interface{}{
		"type":     j.Type,
		"priority": string(j.Priority),
	})

	// Update job status to completed
	err = p.queue.UpdateJobStatus(j.ID, job.StatusCompleted)
	if err != nil {
		jobLogger.JobError(j.ID, "Failed to update job status to completed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	metrics.JobsProcessed.WithLabelValues(j.Type, "completed").Inc()
}
