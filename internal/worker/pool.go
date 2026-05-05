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

type JobProcessor func(ctx context.Context, task *queue.Task) (map[string]interface{}, error)

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
}

type WebSocketPublisher interface {
	PublishJobUpdate(jobID, status string, data map[string]interface{}) error
	PublishWorkflowUpdate(workflowID string, status job.WorkflowStatus, data map[string]interface{}) error
}

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

func (p *WorkerPool) RegisterProcessor(jobType string, processor JobProcessor) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.processors[jobType] = processor
	p.logger.Info(fmt.Sprintf("Registered processor for job type: %s", jobType))
}

func (p *WorkerPool) HasProcessorFor(jobType string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, exists := p.processors[jobType]
	return exists
}

func (p *WorkerPool) Start() {
	p.logger.Info(fmt.Sprintf("Starting worker pool with %d workers", p.numWorkers))

	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.startWorker(i)
	}

	p.wg.Add(1)
	go p.startWorkflowProcessor()

	p.logger.Info("Worker pool started")
}

func (p *WorkerPool) Stop() {
	p.logger.Info("Stopping worker pool...")
	p.cancel()
	p.wg.Wait()
	p.logger.Info("Worker pool stopped")
}

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
			// BRPop inside processNextTask blocks up to 1s — no sleep needed
			p.processNextTask(workerID)
		}
	}
}

func (p *WorkerPool) processNextTask(workerID string) {
	task, err := p.queue.Consume(p.ctx)
	if err != nil {
		return
	}

	// re-read status — task may have been cancelled after enqueue
	if current, err := p.queue.GetTaskStatus(task.ID); err == nil && current.Status == "cancelled" {
		p.logger.Info(fmt.Sprintf("Worker %s skipping cancelled task %s", workerID, task.ID))
		return
	}

	p.metrics.IncrementActiveWorkers(1)
	defer p.metrics.IncrementActiveWorkers(-1)

	p.logger.Info(fmt.Sprintf("Worker %s processing task %s of type %s", workerID, task.ID, task.Type))

	p.mu.RLock()
	processor, exists := p.processors[task.Type]
	p.mu.RUnlock()

	if !exists {
		err := fmt.Errorf("no processor registered for job type: %s", task.Type)
		p.logger.Error(err.Error())
		p.errorHandler.HandleJobError(task, err)
		p.websocket.PublishJobUpdate(task.ID, "failed", map[string]interface{}{"error": err.Error()})
		return
	}

	processingCtx, cancel := context.WithTimeout(p.ctx, 5*time.Minute)
	defer cancel()

	startTime := time.Now()
	result, err := processor(processingCtx, task)
	processingTime := time.Since(startTime).Seconds()
	p.metrics.RecordJobProcessingTime(task.Type, processingTime)

	if err != nil {
		p.logger.Error(fmt.Sprintf("Error processing task %s: %v", task.ID, err))
		p.errorHandler.HandleJobError(task, err)
		p.websocket.PublishJobUpdate(task.ID, "failed", map[string]interface{}{"error": err.Error()})
		return
	}

	task.Status = "completed"
	if result != nil {
		task.Result = result
	}

	if err := p.queue.UpdateStatus(task); err != nil {
		p.logger.Error(fmt.Sprintf("Error updating task status: %v", err))
	}

	if wfID, ok := task.Data["workflow_id"].(string); ok && wfID != "" {
		stepID, _ := task.Data["workflow_step_id"].(string)
		if err := p.workflowManager.CompleteWorkflowStep(wfID, stepID, result); err != nil {
			p.logger.Error(fmt.Sprintf("Error completing workflow step %s: %v", stepID, err))
		}
	}

	p.metrics.IncrementJobCounter("completed")
	p.websocket.PublishJobUpdate(task.ID, "completed", map[string]interface{}{"result": result})
	p.logger.Info(fmt.Sprintf("Worker %s completed task %s in %.2f seconds", workerID, task.ID, processingTime))
}

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

func (p *WorkerPool) processNextWorkflow() {
	workflow, err := p.workflowManager.GetNextWorkflow()
	if err != nil {
		p.logger.Error(fmt.Sprintf("Error getting next workflow: %v", err))
		return
	}
	if workflow == nil {
		return
	}

	if workflow.Status == job.WorkflowStatusPending {
		now := time.Now()
		workflow.Status = job.WorkflowStatusRunning
		workflow.StartedAt = &now

		if err := p.workflowManager.SaveWorkflow(workflow); err != nil {
			p.logger.Error(fmt.Sprintf("Error updating workflow status: %v", err))
			return
		}
		p.websocket.PublishWorkflowUpdate(workflow.ID, workflow.Status, nil)
	}

	readySteps := workflow.GetReadySteps()

	if len(readySteps) == 0 {
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
			p.websocket.PublishWorkflowUpdate(workflow.ID, workflow.Status, nil)
		}

		return
	}

	for _, step := range readySteps {
		task := &queue.Task{
			ID:        step.ID,
			Type:      step.JobType,
			Data:      step.Params,
			Priority:  1,
			CreatedAt: time.Now(),
			Status:    "pending",
		}

		task.Data["workflow_id"] = workflow.ID
		task.Data["workflow_step_id"] = step.ID

		step.Status = job.StepStatusRunning
		if err := workflow.UpdateStepStatus(step.ID, job.StepStatusRunning, "", nil); err != nil {
			p.logger.Error(fmt.Sprintf("Error updating step status: %v", err))
			continue
		}

		if err := p.workflowManager.SaveWorkflow(workflow); err != nil {
			p.logger.Error(fmt.Sprintf("Error saving workflow: %v", err))
			continue
		}

		if err := p.queue.Publish(task); err != nil {
			p.logger.Error(fmt.Sprintf("Error publishing step task: %v", err))
			workflow.UpdateStepStatus(step.ID, job.StepStatusFailed, err.Error(), nil)
			p.workflowManager.SaveWorkflow(workflow)
			continue
		}

		p.logger.Info(fmt.Sprintf("Started workflow step %s of type %s for workflow %s",
			step.ID, step.JobType, workflow.ID))
	}
}
