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
)

// JobProcessor is a function that processes a task
type JobProcessor func(ctx context.Context, task *queue.Task) (map[string]interface{}, error)

// WorkerPool manages a pool of worker goroutines
type WorkerPool struct {
	queue           *queue.RedisQueue
	logger          *logger.Logger
	metrics         *metrics.MetricsCollector
	processors      map[string]JobProcessor
	errorHandler    *ErrorHandler
	workflowManager *job.WorkflowManager
	websocket       WebSocketPublisher
	numWorkers      int
	pollingInterval time.Duration
	wg              sync.WaitGroup
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	activeWorkers   int32 // Atomic counter for active workers
}

// WebSocketPublisher interface for publishing updates
type WebSocketPublisher interface {
	PublishJobUpdate(jobID, status string, data map[string]interface{}) error
	PublishWorkflowUpdate(workflowID string, status job.WorkflowStatus, data map[string]interface{}) error
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(
	queue *queue.RedisQueue,
	logger *logger.Logger,
	metrics *metrics.MetricsCollector,
	errorHandler *ErrorHandler,
	workflowManager *job.WorkflowManager,
	websocket WebSocketPublisher,
	numWorkers int,
	pollingInterval time.Duration,
) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		queue:           queue,
		logger:          logger,
		metrics:         metrics,
		processors:      make(map[string]JobProcessor),
		errorHandler:    errorHandler,
		workflowManager: workflowManager,
		websocket:       websocket,
		numWorkers:      numWorkers,
		pollingInterval: pollingInterval,
		ctx:             ctx,
		cancel:          cancel,
	}
}

// RegisterProcessor registers a processor for a specific job type
func (p *WorkerPool) RegisterProcessor(jobType string, processor JobProcessor) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.processors[jobType] = processor
	p.logger.Info(fmt.Sprintf("Registered processor for job type: %s", jobType))
}

// HasProcessorFor checks if a processor is registered for a job type
func (p *WorkerPool) HasProcessorFor(jobType string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	_, exists := p.processors[jobType]
	return exists
}

// Start starts the worker pool
func (p *WorkerPool) Start() {
	p.logger.Info(fmt.Sprintf("Starting worker pool with %d workers", p.numWorkers))

	// Start task workers
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.startWorker(i)
	}

	// Start workflow processor
	p.wg.Add(1)
	go p.startWorkflowProcessor()

	p.logger.Info("Worker pool started")
}

// Stop gracefully stops the worker pool
func (p *WorkerPool) Stop() {
	p.logger.Info("Stopping worker pool...")
	p.cancel()
	p.wg.Wait()
	p.logger.Info("Worker pool stopped")
}

// startWorker starts a worker goroutine
func (p *WorkerPool) startWorker(id int) {
	defer p.wg.Done()

	workerID := fmt.Sprintf("worker-%d", id)
	p.logger.Info(fmt.Sprintf("Worker %s started", workerID))

	for {
		select {
		case <-p.ctx.Done():
			p.logger.Info(fmt.Sprintf("Worker %s shutting down", workerID))
			return
		default:
			p.processNextTask(workerID)

			// Sleep briefly before next poll to avoid hammering Redis
			time.Sleep(p.pollingInterval)
		}
	}
}

// processNextTask processes the next task from the queue
func (p *WorkerPool) processNextTask(workerID string) {
	// Get next task from queue
	task, err := p.queue.Consume()

	if err != nil {
		// No tasks available
		return
	}

	// Update metrics
	p.metrics.IncrementActiveWorkers(1)
	defer p.metrics.IncrementActiveWorkers(-1)

	p.logger.Info(fmt.Sprintf("Worker %s processing task %s of type %s", workerID, task.ID, task.Type))

	// Get processor for this job type
	p.mu.RLock()
	processor, exists := p.processors[task.Type]
	p.mu.RUnlock()

	if !exists {
		err := fmt.Errorf("no processor registered for job type: %s", task.Type)
		p.logger.Error(err.Error())

		// Handle error (move to dead letter queue)
		p.errorHandler.HandleJobError(task, err)

		// Publish update
		p.websocket.PublishJobUpdate(task.ID, "failed", map[string]interface{}{
			"error": err.Error(),
		})

		return
	}

	// Create task context with timeout
	processingCtx, cancel := context.WithTimeout(p.ctx, 5*time.Minute)
	defer cancel()

	// Record start time for metrics
	startTime := time.Now()

	// Process the task
	result, err := processor(processingCtx, task)

	// Record metrics
	processingTime := time.Since(startTime).Seconds()
	p.metrics.RecordJobProcessingTime(task.Type, processingTime)

	if err != nil {
		p.logger.Error(fmt.Sprintf("Error processing task %s: %v", task.ID, err))

		// Handle the error with appropriate retry/dead letter strategy
		p.errorHandler.HandleJobError(task, err)

		// Publish update
		p.websocket.PublishJobUpdate(task.ID, "failed", map[string]interface{}{
			"error": err.Error(),
		})

		return
	}

	// Task completed successfully
	task.Status = "completed"

	if result != nil {
		// Convert result to JSON string for storage in Redis
		task.Data["result"] = result
	}

	// Update task status
	if err := p.queue.UpdateStatus(task); err != nil {
		p.logger.Error(fmt.Sprintf("Error updating task status: %v", err))
	}

	// Increment completed counter
	p.metrics.IncrementJobCounter("completed")

	// Publish update
	p.websocket.PublishJobUpdate(task.ID, "completed", map[string]interface{}{
		"result": result,
	})

	p.logger.Info(fmt.Sprintf("Worker %s completed task %s in %.2f seconds",
		workerID, task.ID, processingTime))
}

// startWorkflowProcessor starts the workflow processor
func (p *WorkerPool) startWorkflowProcessor() {
	defer p.wg.Done()

	p.logger.Info("Workflow processor started")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			p.logger.Info("Workflow processor shutting down")
			return

		case <-ticker.C:
			p.processNextWorkflow()
		}
	}
}

// processNextWorkflow processes the next workflow from the queue
func (p *WorkerPool) processNextWorkflow() {
	// Get next workflow
	workflow, err := p.workflowManager.GetNextWorkflow()

	if err != nil {
		p.logger.Error(fmt.Sprintf("Error getting next workflow: %v", err))
		return
	}

	if workflow == nil {
		// No workflows available
		return
	}

	// Update workflow status to running if it's pending
	if workflow.Status == job.WorkflowStatusPending {
		now := time.Now()
		workflow.Status = job.WorkflowStatusRunning
		workflow.StartedAt = &now

		if err := p.workflowManager.SaveWorkflow(workflow); err != nil {
			p.logger.Error(fmt.Sprintf("Error updating workflow status: %v", err))
			return
		}

		// Publish update
		p.websocket.PublishWorkflowUpdate(workflow.ID, workflow.Status, nil)
	}

	// Get all ready steps
	readySteps := workflow.GetReadySteps()

	if len(readySteps) == 0 {
		// No steps ready to process
		// Check if all steps are complete or if there was a failure
		allComplete := true
		hasFailed := false

		for _, step := range workflow.Steps {
			if step.Status == job.StepStatusPending {
				allComplete = false
			} else if step.Status == job.StepStatusFailed {
				hasFailed = true
			}
		}

		if allComplete || hasFailed {
			// Workflow is complete or has failed
			now := time.Now()
			workflow.FinishedAt = &now

			if hasFailed {
				workflow.Status = job.WorkflowStatusFailed
			} else {
				workflow.Status = job.WorkflowStatusCompleted
			}

			if err := p.workflowManager.SaveWorkflow(workflow); err != nil {
				p.logger.Error(fmt.Sprintf("Error updating workflow status: %v", err))
				return
			}

			// Publish update
			p.websocket.PublishWorkflowUpdate(workflow.ID, workflow.Status, nil)
		}

		return
	}

	// Process each ready step
	for _, step := range readySteps {
		// Create a task for the step
		task := &queue.Task{
			ID:        step.ID,
			Type:      step.JobType,
			Data:      step.Params,
			Priority:  1, // Use normal priority
			CreatedAt: time.Now(),
			Status:    "pending",
		}

		// Include workflow context
		task.Data["workflow_id"] = workflow.ID
		task.Data["workflow_step_id"] = step.ID

		// Update step status
		step.Status = job.StepStatusRunning
		if err := workflow.UpdateStepStatus(step.ID, job.StepStatusRunning, "", nil); err != nil {
			p.logger.Error(fmt.Sprintf("Error updating step status: %v", err))
			continue
		}

		// Save workflow
		if err := p.workflowManager.SaveWorkflow(workflow); err != nil {
			p.logger.Error(fmt.Sprintf("Error saving workflow: %v", err))
			continue
		}

		// Publish step to queue
		if err := p.queue.Publish(task); err != nil {
			p.logger.Error(fmt.Sprintf("Error publishing step task: %v", err))

			// Update step status as failed
			workflow.UpdateStepStatus(step.ID, job.StepStatusFailed, err.Error(), nil)
			p.workflowManager.SaveWorkflow(workflow)
			continue
		}

		p.logger.Info(fmt.Sprintf("Started workflow step %s of type %s for workflow %s",
			step.ID, step.JobType, workflow.ID))
	}
}
