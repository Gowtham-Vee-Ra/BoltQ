// internal/api/handler.go
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"BoltQ/internal/job"
	"BoltQ/internal/queue"
	"BoltQ/pkg/logger"
	"BoltQ/pkg/metrics"

	"github.com/google/uuid"
)

// Handler handles API requests
type Handler struct {
	queue  queue.Queue
	logger *logger.Logger
}

// NewHandler creates a new API handler
func NewHandler(q queue.Queue) *Handler {
	return &Handler{
		queue:  q,
		logger: logger.NewLogger("api-handler"),
	}
}

// SubmitJobRequest represents a job submission request
type SubmitJobRequest struct {
	Type     string                 `json:"type"`
	Priority string                 `json:"priority,omitempty"`
	Data     map[string]interface{} `json:"data"`
	Tags     []string               `json:"tags,omitempty"`
	Timeout  int                    `json:"timeout,omitempty"`
}

// JobResponse represents a job response
type JobResponse struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// writeJSONResponse writes a JSON response
func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeErrorResponse writes an error response
func writeErrorResponse(w http.ResponseWriter, status int, message string) {
	writeJSONResponse(w, status, map[string]string{"error": message})
}

// SubmitJobHandler handles job submission
func (h *Handler) SubmitJobHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		h.logger.Error("Invalid request method", map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
		})
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse the request body
	var req SubmitJobRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		h.logger.Error("Failed to decode request body", map[string]interface{}{
			"error": err.Error(),
		})
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.Type == "" {
		h.logger.Error("Missing job type in request")
		writeErrorResponse(w, http.StatusBadRequest, "Job type is required")
		return
	}

	// Create a new job
	jobID := uuid.New().String()

	// Determine priority
	priority := job.PriorityNormal
	if req.Priority != "" {
		switch req.Priority {
		case "low":
			priority = job.PriorityLow
		case "normal":
			priority = job.PriorityNormal
		case "high":
			priority = job.PriorityHigh
		case "critical":
			priority = job.PriorityCritical
		default:
			h.logger.Warn("Invalid priority in request, using normal", map[string]interface{}{
				"requested_priority": req.Priority,
			})
		}
	}

	// Set default timeout if not provided
	timeout := 300 // 5 minutes default
	if req.Timeout > 0 {
		timeout = req.Timeout
	}

	j := &job.Job{
		ID:        jobID,
		Type:      req.Type,
		Data:      req.Data,
		Status:    job.StatusPending,
		Priority:  priority,
		Tags:      req.Tags,
		CreatedAt: time.Now(),
		Timeout:   timeout,
	}

	// Publish job to queue
	err := h.queue.PublishJob(j)
	if err != nil {
		h.logger.Error("Failed to publish job", map[string]interface{}{
			"job_id": jobID,
			"type":   req.Type,
			"error":  err.Error(),
		})
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to submit job")
		return
	}

	// Log job submission
	h.logger.Info("Job submitted", map[string]interface{}{
		"job_id":   jobID,
		"type":     req.Type,
		"priority": string(priority),
	})

	// Record metrics
	metrics.JobsSubmitted.WithLabelValues(req.Type, string(priority)).Inc()

	// Return response
	writeJSONResponse(w, http.StatusAccepted, JobResponse{
		ID:        jobID,
		Type:      req.Type,
		Status:    string(job.StatusPending),
		CreatedAt: j.CreatedAt,
	})
}

// GetJobStatusHandler handles job status requests
func (h *Handler) GetJobStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept GET requests
	if r.Method != http.MethodGet {
		h.logger.Error("Invalid request method", map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
		})
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get job ID from query parameters
	jobID := r.URL.Query().Get("id")
	if jobID == "" {
		h.logger.Error("Missing job ID in request")
		writeErrorResponse(w, http.StatusBadRequest, "Job ID is required")
		return
	}

	// Get job status
	status, err := h.queue.GetJobStatus(jobID)
	if err != nil {
		h.logger.Error("Failed to get job status", map[string]interface{}{
			"job_id": jobID,
			"error":  err.Error(),
		})
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get job status")
		return
	}

	if status == job.StatusUnknown {
		h.logger.Warn("Job not found", map[string]interface{}{
			"job_id": jobID,
		})
		writeErrorResponse(w, http.StatusNotFound, "Job not found")
		return
	}

	h.logger.Info("Job status retrieved", map[string]interface{}{
		"job_id": jobID,
		"status": string(status),
	})

	// Return response
	writeJSONResponse(w, http.StatusOK, map[string]string{
		"id":     jobID,
		"status": string(status),
	})
}

// GetQueueStatsHandler handles queue statistics requests
func (h *Handler) GetQueueStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept GET requests
	if r.Method != http.MethodGet {
		h.logger.Error("Invalid request method", map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
		})
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get Redis queue pointer
	redisQueue, ok := h.queue.(*queue.RedisQueue)
	if !ok {
		h.logger.Error("Queue implementation does not support stats")
		writeErrorResponse(w, http.StatusInternalServerError, "Queue statistics not available")
		return
	}

	// Get queue statistics
	stats, err := redisQueue.GetQueueStats()
	if err != nil {
		h.logger.Error("Failed to get queue statistics", map[string]interface{}{
			"error": err.Error(),
		})
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get queue statistics")
		return
	}

	h.logger.Info("Queue statistics retrieved")

	// Return response
	writeJSONResponse(w, http.StatusOK, stats)
}
