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

type ErrorCategory int

const (
	TransientError ErrorCategory = iota
	DataError
	SystemError
	UnknownError
)

type ErrorHandler struct {
	queue   *queue.RedisQueue
	logger  *logger.Logger
	metrics *metrics.MetricsCollector
}

func NewErrorHandler(q *queue.RedisQueue, l *logger.Logger, m *metrics.MetricsCollector) *ErrorHandler {
	return &ErrorHandler{queue: q, logger: l, metrics: m}
}

func (h *ErrorHandler) HandleJobError(task *queue.Task, err error) error {
	if err == nil {
		return nil
	}

	category := h.categorizeError(err)
	h.metrics.IncrementErrorCounter(categoryToString(category))
	h.logger.Error(fmt.Sprintf("Task %s failed with error [%s]: %v", task.ID, categoryToString(category), err))

	switch category {
	case TransientError:
		if task.Attempts < getMaxAttempts(category) {
			return h.queue.RetryTask(task, err)
		}
		fallthrough

	case DataError:
		h.logger.Error(fmt.Sprintf("Moving task %s to dead letter queue due to data error", task.ID))
		return h.queue.MoveToDeadLetterQueue(task, err)

	case SystemError:
		if task.Attempts < getMaxAttempts(category) {
			return h.retryWithSystemErrorBackoff(task, err)
		}
		h.logger.Error(fmt.Sprintf("Moving task %s to dead letter queue after exhausting system error retries", task.ID))
		return h.queue.MoveToDeadLetterQueue(task, err)

	case UnknownError:
		if task.Attempts < getMaxAttempts(category) {
			return h.queue.RetryTask(task, err)
		}
		h.logger.Error(fmt.Sprintf("Moving task %s to dead letter queue after exhausting retries", task.ID))
		return h.queue.MoveToDeadLetterQueue(task, err)
	}

	return nil
}

func (h *ErrorHandler) categorizeError(err error) ErrorCategory {
	errMsg := err.Error()

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return TransientError
	}

	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return SystemError
	}

	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "network timeout") ||
		strings.Contains(errMsg, "connection reset by peer") {
		return SystemError
	}

	if strings.Contains(errMsg, "validation failed") ||
		strings.Contains(errMsg, "invalid parameter") ||
		strings.Contains(errMsg, "not found") ||
		strings.Contains(errMsg, "bad request") {
		return DataError
	}

	return UnknownError
}

func (h *ErrorHandler) retryWithSystemErrorBackoff(task *queue.Task, err error) error {
	task.Attempts++
	task.Status = "retrying"
	task.LastError = err.Error()

	backoffSeconds := 5 * task.Attempts
	if backoffSeconds > 120 {
		backoffSeconds = 120
	}

	h.logger.Info(fmt.Sprintf("System error for task %s, attempt %d. Retrying in %d seconds",
		task.ID, task.Attempts, backoffSeconds))

	return h.queue.PublishDelayed(task, int(backoffSeconds))
}

func getMaxAttempts(category ErrorCategory) int {
	switch category {
	case TransientError:
		return 5
	case SystemError:
		return 10
	case DataError:
		return 0
	default:
		return 3
	}
}

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

func EnrichError(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}

func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	errMsg := err.Error()
	return strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "network timeout") ||
		strings.Contains(errMsg, "connection reset by peer") ||
		strings.Contains(errMsg, "temporary") ||
		strings.Contains(errMsg, "timeout")
}
