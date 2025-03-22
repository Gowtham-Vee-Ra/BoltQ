// internal/queue/queue.go
package queue

// Queue defines the interface for job queues
type Queue interface {
	// Basic queue operations
	Publish(task string) error
	Consume() (string, error)

	// Retry and dead letter queue operations
	PublishToRetry(task string) error
	ConsumeRetry() (string, error)
	PublishToDeadLetter(task string) error

	// Job status operations
	SaveJobStatus(jobID, jobJSON string) error
	GetJobStatus(jobID string) (string, error)

	// Queue statistics
	GetQueueStats() (map[string]int64, error)

	// Distributed locking
	AcquireJobLock(jobID, workerID string) (bool, error)
	ReleaseJobLock(jobID, workerID string) (bool, error)

	// Cleanup
	Close() error
}
