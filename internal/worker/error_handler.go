// internal/worker/error_handler.go
package worker

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"

	"BoltQ/internal/queue"
	"BoltQ/pkg/logger"
	"BoltQ/pkg/metrics"
)

// ErrorCategory classifies the type of error for appropriate handling
type ErrorCategory int

const (
	// TransientError represents a temporary error that can be retried
	TransientError ErrorCategory = iota

	// DataError represents an error with the job data itself
	DataError

	// SystemError represents an error with the system
	SystemError

	// UnknownError represents an error that couldn't be classified
	UnknownError
)

// ErrorHandler manages error handling and retry logic
type ErrorHandler struct {
	queue   *queue.RedisQueue
	logger  *logger.Logger
	metrics *metrics.MetricsCollector
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(q *queue.RedisQueue, l *logger.Logger, m *metrics.MetricsCollector) *ErrorHandler {
	return &ErrorHandler{
		queue:   q,
		logger:  l,
		metrics: m,
	}
}

// HandleJobError processes an error from a job and determines the appropriate action
func (h *ErrorHandler) HandleJobError(task *queue.Task, err error) error {
	if err == nil {
		return nil
	}

	// Categorize the error
	category := h.categorizeError(err)
	h.metrics.IncrementErrorCounter(string(category))

	// Log error with proper context
	h.logger.Error(fmt.Sprintf("Task %s failed with error [%s]: %v",
		task.ID, categoryToString(category), err))

	// Handle based on category
	switch category {
	case TransientError:
		// Retry with exponential backoff if under max attempts
		if task.Attempts < getMaxAttempts(category) {
			return h.queue.RetryTask(task, err)
		}
		// Otherwise treat as permanent failure
		fallthrough

	case DataError:
		// Data errors are not retried, move to dead letter queue
		h.logger.Error(fmt.Sprintf("Moving task %s to dead letter queue due to data error", task.ID))
		return h.queue.MoveToDeadLetterQueue(task, err)

	case SystemError:
		// System errors have different max attempts and backoff strategy
		if task.Attempts < getMaxAttempts(category) {
			// Use a different backoff strategy for system errors
			return h.retryWithSystemErrorBackoff(task, err)
		}
		h.logger.Error(fmt.Sprintf("Moving task %s to dead letter queue after exhausting system error retries", task.ID))
		return h.queue.MoveToDeadLetterQueue(task, err)

	case UnknownError:
		// Unknown errors get default retry behavior
		if task.Attempts < getMaxAttempts(category) {
			return h.queue.RetryTask(task, err)
		}
		h.logger.Error(fmt.Sprintf("Moving task %s to dead letter queue after exhausting retries", task.ID))
		return h.queue.MoveToDeadLetterQueue(task, err)
	}

	return nil
}

// categorizeError determines what type of error occurred
func (h *ErrorHandler) categorizeError(err error) ErrorCategory {
	errMsg := err.Error()

	// Check for network and system errors (usually transient)
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return TransientError
	}

	// Check for specific system errors
	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return SystemError
	}

	// Check for Redis connection errors
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "network timeout") ||
		strings.Contains(errMsg, "connection reset by peer") {
		return SystemError
	}

	// Check for data validation errors
	if strings.Contains(errMsg, "validation failed") ||
		strings.Contains(errMsg, "invalid parameter") ||
		strings.Contains(errMsg, "not found") ||
		strings.Contains(errMsg, "bad request") {
		return DataError
	}

	// Default to unknown
	return UnknownError
}

// retryWithSystemErrorBackoff uses a custom backoff for system errors
func (h *ErrorHandler) retryWithSystemErrorBackoff(task *queue.Task, err error) error {
	task.Attempts++
	task.Status = "retrying"
	task.LastError = err.Error()

	// For system errors, we use a more aggressive linear backoff
	// starting with 5 seconds and increasing by 5 seconds each attempt
	backoffSeconds := 5 * task.Attempts

	// Cap at 2 minutes
	if backoffSeconds > 120 {
		backoffSeconds = 120
	}

	h.logger.Info(fmt.Sprintf("System error for task %s, attempt %d. Retrying in %d seconds",
		task.ID, task.Attempts, backoffSeconds))

	return h.queue.PublishDelayed(task, int(backoffSeconds))
}

// getMaxAttempts returns the maximum number of retry attempts based on error category
func getMaxAttempts(category ErrorCategory) int {
	switch category {
	case TransientError:
		return 5 // Transient errors get more retries
	case SystemError:
		return 10 // System errors get the most retries
	case DataError:
		return 0 // Data errors don't get retried
	default:
		return 3 // Default for unknown errors
	}
}

// categoryToString converts an error category to a human-readable string
func categoryToString(category ErrorCategory) string {
	switch category {
	case TransientError:
		return "TRANSIENT"
	case DataError:
		return "DATA"
	case SystemError:
		return "SYSTEM"
	default:
		return "UNKNOWN"
	}
}

// EnrichError adds context to an error
func EnrichError(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}

// IsRetryableError determines if an error can be retried
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific error types
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// Check for system errors
	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	// Check error message
	errMsg := err.Error()
	return strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "network timeout") ||
		strings.Contains(errMsg, "connection reset by peer") ||
		strings.Contains(errMsg, "temporary") ||
		strings.Contains(errMsg, "timeout")
}
