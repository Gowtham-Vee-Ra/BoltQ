// pkg/metrics/prometheus.go
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Job metrics
	JobsSubmitted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "boltq_jobs_submitted_total",
			Help: "The total number of submitted jobs",
		},
		[]string{"type", "priority"},
	)

	JobsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "boltq_jobs_processed_total",
			Help: "The total number of processed jobs",
		},
		[]string{"type", "status"},
	)

	JobsInQueue = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "boltq_jobs_in_queue",
			Help: "The number of jobs currently in each queue",
		},
		[]string{"queue", "priority"},
	)

	JobProcessingTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "boltq_job_processing_seconds",
			Help:    "Time taken to process jobs",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 10), // From 10ms to ~10s
		},
		[]string{"type"},
	)

	// Worker metrics
	WorkerPoolSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "boltq_worker_pool_size",
			Help: "The size of the worker pool",
		},
	)

	ActiveWorkers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "boltq_active_workers",
			Help: "The number of currently active workers",
		},
	)

	// Queue metrics
	RedisOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "boltq_redis_operations_total",
			Help: "The total number of Redis operations",
		},
		[]string{"operation", "status"},
	)

	RedisOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "boltq_redis_operation_seconds",
			Help:    "Duration of Redis operations",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // From 1ms to ~1s
		},
		[]string{"operation"},
	)
)

// SetupMetricsServer starts the HTTP server for Prometheus metrics
func SetupMetricsServer(addr string) {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		http.ListenAndServe(addr, nil)
	}()
}
