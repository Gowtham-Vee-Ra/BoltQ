package worker

import (
	"sync"
	"time"

	"BoltQ/internal/queue"
	"BoltQ/pkg/logger"
	"BoltQ/pkg/metrics"
)

// DelayedJobProcessor is responsible for moving ready delayed jobs to regular queues
type DelayedJobProcessor struct {
	queue        *queue.RedisQueue
	logger       *logger.Logger
	metrics      *metrics.MetricsCollector
	ticker       *time.Ticker
	stopChan     chan struct{}
	wg           sync.WaitGroup
	processCount int64
}

// NewDelayedJobProcessor creates a new processor for delayed jobs
func NewDelayedJobProcessor(queue *queue.RedisQueue, logger *logger.Logger, metrics *metrics.MetricsCollector) *DelayedJobProcessor {
	return &DelayedJobProcessor{
		queue:    queue,
		logger:   logger,
		metrics:  metrics,
		stopChan: make(chan struct{}),
	}
}

// Start begins the processing of delayed jobs at regular intervals
func (p *DelayedJobProcessor) Start(interval time.Duration) {
	p.ticker = time.NewTicker(interval)
	p.wg.Add(1)

	go func() {
		defer p.wg.Done()

		for {
			select {
			case <-p.ticker.C:
				p.processDelayedJobs()
			case <-p.stopChan:
				p.ticker.Stop()
				return
			}
		}
	}()

	p.logger.Info("Delayed job processor started")
}

// Stop gracefully stops the processor
func (p *DelayedJobProcessor) Stop() {
	close(p.stopChan)
	p.wg.Wait()
	p.logger.Info("Delayed job processor stopped")
}

// processDelayedJobs moves ready jobs from delayed queue to regular queues
func (p *DelayedJobProcessor) processDelayedJobs() {
	startTime := time.Now()

	// Record metrics for monitoring
	defer func() {
		processingTime := time.Since(startTime).Seconds()
		p.metrics.RecordDelayedJobProcessorRun(processingTime)
	}()

	// Process all jobs that are ready
	count, err := p.queue.ProcessDelayedTasks()
	if err != nil {
		p.logger.Error("Error processing delayed tasks: " + err.Error())
		return
	}

	if count > 0 {
		p.processCount += int64(count)
		p.metrics.RecordDelayedJobsProcessed(count)
		p.logger.Info("Processed " + string(count) + " delayed tasks")
	}
}

// GetProcessCount returns the total number of processed delayed jobs
func (p *DelayedJobProcessor) GetProcessCount() int64 {
	return p.processCount
}
