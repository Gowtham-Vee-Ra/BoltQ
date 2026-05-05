package worker

import (
	"fmt"
	"sync"
	"time"

	"BoltQ/internal/queue"
	"BoltQ/pkg/logger"
	"BoltQ/pkg/metrics"
)

type DelayedJobProcessor struct {
	queue        *queue.RedisQueue
	logger       *logger.Logger
	metrics      *metrics.MetricsCollector
	ticker       *time.Ticker
	stopChan     chan struct{}
	wg           sync.WaitGroup
	processCount int64
}

func NewDelayedJobProcessor(queue *queue.RedisQueue, logger *logger.Logger, metrics *metrics.MetricsCollector) *DelayedJobProcessor {
	return &DelayedJobProcessor{
		queue:    queue,
		logger:   logger,
		metrics:  metrics,
		stopChan: make(chan struct{}),
	}
}

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

func (p *DelayedJobProcessor) Stop() {
	close(p.stopChan)
	p.wg.Wait()
	p.logger.Info("Delayed job processor stopped")
}

func (p *DelayedJobProcessor) processDelayedJobs() {
	startTime := time.Now()
	defer func() {
		p.metrics.RecordDelayedJobProcessorRun(time.Since(startTime).Seconds())
	}()

	count, err := p.queue.ProcessDelayedTasks()
	if err != nil {
		p.logger.Error("Error processing delayed tasks: " + err.Error())
		return
	}

	if count > 0 {
		p.processCount += int64(count)
		p.metrics.RecordDelayedJobsProcessed(count)
		p.logger.Info(fmt.Sprintf("Processed %d delayed tasks", count))
	}
}

func (p *DelayedJobProcessor) GetProcessCount() int64 {
	return p.processCount
}
