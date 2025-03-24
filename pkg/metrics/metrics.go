// pkg/metrics/metrics.go
package metrics

import (
	"fmt"
	"sync/atomic"
)

// MetricsCollector handles Prometheus metrics collection
// This is a wrapper around the global Prometheus metrics defined in prometheus.go
type MetricsCollector struct {
	namespace          string
	activeWorkersCount int32 // atomic counter
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(namespace string) *MetricsCollector {
	// Create a metrics collector that uses the global metrics
	mc := &MetricsCollector{
		namespace: namespace,
	}

	// Set initial values for relevant gauges
	WorkerPoolSize.Set(0)
	ActiveWorkers.Set(0)

	return mc
}

// IncrementJobCounter increments the job counter for a status
func (mc *MetricsCollector) IncrementJobCounter(status string) {
	JobsProcessed.WithLabelValues("all", status).Inc()
}

// RecordJobProcessingTime records the time taken to process a job
func (mc *MetricsCollector) RecordJobProcessingTime(jobType string, seconds float64) {
	JobProcessingTime.WithLabelValues(jobType).Observe(seconds)
}

// SetQueueDepth sets the queue depth for a queue
func (mc *MetricsCollector) SetQueueDepth(queue string, depth float64) {
	JobsInQueue.WithLabelValues(queue, "all").Set(depth)
}

// IncrementErrorCounter increments the error counter for a type
func (mc *MetricsCollector) IncrementErrorCounter(errorType string) {
	// Use Redis operation metrics for errors
	RedisOperations.WithLabelValues("error", errorType).Inc()
}

// IncrementActiveWorkers increments or decrements the active workers count
func (mc *MetricsCollector) IncrementActiveWorkers(delta int) {
	newCount := atomic.AddInt32(&mc.activeWorkersCount, int32(delta))
	ActiveWorkers.Set(float64(newCount))
}

// RecordDelayedJobsProcessed records the number of delayed jobs processed
func (mc *MetricsCollector) RecordDelayedJobsProcessed(count int) {
	JobsProcessed.WithLabelValues("delayed", "processed").Add(float64(count))
}

// RecordDelayedJobProcessorRun records the time taken for a delayed job processor run
func (mc *MetricsCollector) RecordDelayedJobProcessorRun(seconds float64) {
	// Use Redis operation metrics for this, since we don't have a dedicated metric
	RedisOperationDuration.WithLabelValues("delayed_processor").Observe(seconds)
}

// RecordAPIRequestDuration records the time taken to process an API request
func (mc *MetricsCollector) RecordAPIRequestDuration(endpoint string, seconds float64) {
	// For API requests, we'll use the Redis operation metrics
	RedisOperationDuration.WithLabelValues(fmt.Sprintf("api_%s", endpoint)).Observe(seconds)
}
