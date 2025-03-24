// Updated factory.go without Kafka support
package queue

import (
	"fmt"
)

// QueueType represents supported queue backend types
type QueueType string

const (
	// QueueTypeRedis represents a Redis-backed queue
	QueueTypeRedis QueueType = "redis"
)

// QueueServiceFactory creates and manages queue instances
type QueueServiceFactory struct {
	factories map[QueueType]QueueFactory
	logger    Logger
}

// NewQueueServiceFactory creates a new queue service factory
func NewQueueServiceFactory(logger Logger) *QueueServiceFactory {
	return &QueueServiceFactory{
		factories: make(map[QueueType]QueueFactory),
		logger:    logger,
	}
}

// RegisterQueueFactory registers a queue factory for a specific type
func (f *QueueServiceFactory) RegisterQueueFactory(queueType QueueType, factory QueueFactory) {
	f.factories[queueType] = factory
}

// CreateQueue creates a new queue of the specified type
func (f *QueueServiceFactory) CreateQueue(queueType QueueType, config map[string]string) (Queue, error) {
	factory, ok := f.factories[queueType]
	if !ok {
		return nil, fmt.Errorf("unsupported queue type: %s", queueType)
	}

	return factory.CreateQueue(config)
}

// InitDefaultFactories initializes the default queue factories
func (f *QueueServiceFactory) InitDefaultFactories() {
	// Register Redis queue factory
	f.RegisterQueueFactory(QueueTypeRedis, NewRedisQueueFactory(f.logger))
}
