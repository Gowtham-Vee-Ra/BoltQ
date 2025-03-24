// internal/api/handler.go
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"BoltQ/internal/job"
	"BoltQ/internal/queue"
	"BoltQ/pkg/logger"
	"BoltQ/pkg/metrics"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Handler handles HTTP requests for the API
type Handler struct {
	queue           *queue.RedisQueue
	logger          *logger.Logger
	metrics         *metrics.MetricsCollector
	workflowManager *job.WorkflowManager
}

// NewHandler creates a new API handler
func NewHandler(queue *queue.RedisQueue, logger *logger.Logger, metrics *metrics.MetricsCollector,
	workflowManager *job.WorkflowManager) *Handler {
	return &Handler{
		queue:           queue,
		logger:          logger,
		metrics:         metrics,
		workflowManager: workflowManager,
	}
}

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SubmitJobRequest represents a job submission request
type SubmitJobRequest struct {
	Type         string                 `json:"type"`
	Data         map[string]interface{} `json:"data"`
	Priority     int                    `json:"priority,omitempty"`
	DelaySeconds int                    `json:"delay_seconds,omitempty"`
}

// RegisterRoutes sets up the API routes
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Job endpoints
	r.HandleFunc("/api/v1/jobs", h.SubmitJobHandler).Methods("POST")
	r.HandleFunc("/api/v1/jobs/{id}", h.GetJobStatusHandler).Methods("GET")
	r.HandleFunc("/api/v1/jobs/{id}/cancel", h.CancelJobHandler).Methods("POST")

	// Queue endpoints
	r.HandleFunc("/api/v1/queues/stats", h.GetQueueStatsHandler).Methods("GET")

	// Workflow endpoints
	r.HandleFunc("/api/v1/workflows", h.CreateWorkflowHandler).Methods("POST")
	r.HandleFunc("/api/v1/workflows", h.ListWorkflowsHandler).Methods("GET")
	r.HandleFunc("/api/v1/workflows/{id}", h.GetWorkflowHandler).Methods("GET")
	r.HandleFunc("/api/v1/workflows/{id}", h.DeleteWorkflowHandler).Methods("DELETE")

	// Health endpoint
	r.HandleFunc("/health", h.HealthCheckHandler).Methods("GET")
}

// SubmitJobHandler handles job submission requests
// @Summary Submit a new job
// @Description Submits a job to the queue for processing
// @Tags jobs
// @Accept json
// @Produce json
// @Param job body SubmitJobRequest true "Job details"
// @Success 200 {object} Response
// @Failure 400 {object} Response "Invalid request"
// @Failure 500 {object} Response "Server error"
// @Router /api/v1/jobs [post]
func (h *Handler) SubmitJobHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		h.metrics.RecordAPIRequestDuration("submit_job", time.Since(startTime).Seconds())
	}()

	var req SubmitJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate request
	if req.Type == "" {
		h.respondWithError(w, http.StatusBadRequest, "Job type is required")
		return
	}

	// Create a task
	task := &queue.Task{
		ID:        uuid.New().String(),
		Type:      req.Type,
		Data:      req.Data,
		Priority:  req.Priority,
		CreatedAt: time.Now(),
		Status:    "pending",
	}

	var err error

	// Either publish immediately or with delay
	if req.DelaySeconds > 0 {
		err = h.queue.PublishDelayed(task, req.DelaySeconds)
	} else {
		err = h.queue.Publish(task)
	}

	if err != nil {
		h.logger.Error("Failed to publish job: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to publish job")
		return
	}

	h.metrics.IncrementJobCounter("submitted")
	h.logger.Info(fmt.Sprintf("Job %s of type %s submitted successfully", task.ID, task.Type))

	h.respondWithJSON(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]string{
			"job_id": task.ID,
		},
	})
}

// GetJobStatusHandler handles job status requests
// @Summary Get job status
// @Description Gets the current status of a job
// @Tags jobs
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} Response
// @Failure 404 {object} Response "Job not found"
// @Failure 500 {object} Response "Server error"
// @Router /api/v1/jobs/{id} [get]
func (h *Handler) GetJobStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	task, err := h.queue.GetTaskStatus(jobID)

	if err != nil {
		if err.Error() == "task not found" {
			h.respondWithError(w, http.StatusNotFound, "Job not found")
			return
		}

		h.logger.Error("Failed to get job status: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get job status")
		return
	}

	h.respondWithJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    task,
	})
}

// CancelJobHandler handles job cancellation requests
// @Summary Cancel a job
// @Description Cancels a pending job
// @Tags jobs
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} Response
// @Failure 404 {object} Response "Job not found"
// @Failure 400 {object} Response "Job cannot be cancelled"
// @Failure 500 {object} Response "Server error"
// @Router /api/v1/jobs/{id}/cancel [post]
func (h *Handler) CancelJobHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	// Get current status
	task, err := h.queue.GetTaskStatus(jobID)

	if err != nil {
		if err.Error() == "task not found" {
			h.respondWithError(w, http.StatusNotFound, "Job not found")
			return
		}

		h.logger.Error("Failed to get job status: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get job status")
		return
	}

	// Only pending jobs can be cancelled
	if task.Status != "pending" && task.Status != "scheduled" {
		h.respondWithError(w, http.StatusBadRequest, "Only pending or scheduled jobs can be cancelled")
		return
	}

	// Update status to cancelled
	task.Status = "cancelled"
	if err := h.queue.UpdateStatus(task); err != nil {
		h.logger.Error("Failed to update job status: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to cancel job")
		return
	}

	h.metrics.IncrementJobCounter("cancelled")
	h.logger.Info(fmt.Sprintf("Job %s cancelled successfully", jobID))

	h.respondWithJSON(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]string{
			"status": "cancelled",
		},
	})
}

// GetQueueStatsHandler handles queue statistics requests
// @Summary Get queue statistics
// @Description Gets statistics about all queues
// @Tags queues
// @Produce json
// @Success 200 {object} Response
// @Failure 500 {object} Response "Server error"
// @Router /api/v1/queues/stats [get]
func (h *Handler) GetQueueStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := h.queue.GetQueueStats()

	if err != nil {
		h.logger.Error("Failed to get queue stats: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get queue statistics")
		return
	}

	h.respondWithJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    stats,
	})
}

// CreateWorkflowHandler handles workflow creation requests
// @Summary Create a new workflow
// @Description Creates a new job workflow
// @Tags workflows
// @Accept json
// @Produce json
// @Param workflow body object true "Workflow details"
// @Success 200 {object} Response
// @Failure 400 {object} Response "Invalid request"
// @Failure 500 {object} Response "Server error"
// @Router /api/v1/workflows [post]
func (h *Handler) CreateWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string                  `json:"name"`
		Steps    []job.WorkflowStepInput `json:"steps"`
		Metadata map[string]interface{}  `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate request
	if req.Name == "" {
		h.respondWithError(w, http.StatusBadRequest, "Workflow name is required")
		return
	}

	if len(req.Steps) == 0 {
		h.respondWithError(w, http.StatusBadRequest, "Workflow must have at least one step")
		return
	}

	// Create workflow
	workflow := job.NewWorkflow(req.Name)

	if req.Metadata != nil {
		workflow.Metadata = req.Metadata
	}

	// Add steps
	for _, stepInput := range req.Steps {
		workflow.AddStep(stepInput.JobType, stepInput.Params, stepInput.DependsOn)
	}

	// Save workflow
	if err := h.workflowManager.SaveWorkflow(workflow); err != nil {
		h.logger.Error("Failed to save workflow: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create workflow")
		return
	}

	h.logger.Info(fmt.Sprintf("Workflow %s created successfully with %d steps", workflow.ID, len(workflow.Steps)))

	h.respondWithJSON(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]string{
			"workflow_id": workflow.ID,
		},
	})
}

// GetWorkflowHandler handles workflow retrieval requests
// @Summary Get workflow details
// @Description Gets the details of a workflow
// @Tags workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {object} Response
// @Failure 404 {object} Response "Workflow not found"
// @Failure 500 {object} Response "Server error"
// @Router /api/v1/workflows/{id} [get]
func (h *Handler) GetWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	workflow, err := h.workflowManager.GetWorkflow(workflowID)

	if err != nil {
		if err.Error() == fmt.Sprintf("workflow %s not found", workflowID) {
			h.respondWithError(w, http.StatusNotFound, "Workflow not found")
			return
		}

		h.logger.Error("Failed to get workflow: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get workflow")
		return
	}

	h.respondWithJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    workflow,
	})
}

// ListWorkflowsHandler handles workflow listing requests
// @Summary List workflows
// @Description Lists workflows with pagination
// @Tags workflows
// @Produce json
// @Param limit query int false "Number of workflows to return (default 20)"
// @Param offset query int false "Offset for pagination (default 0)"
// @Success 200 {object} Response
// @Failure 500 {object} Response "Server error"
// @Router /api/v1/workflows [get]
func (h *Handler) ListWorkflowsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20
	offset := 0

	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	workflows, err := h.workflowManager.ListWorkflows(limit, offset)

	if err != nil {
		h.logger.Error("Failed to list workflows: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to list workflows")
		return
	}

	h.respondWithJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    workflows,
	})
}

// DeleteWorkflowHandler handles workflow deletion requests
// @Summary Delete a workflow
// @Description Deletes a workflow and its data
// @Tags workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {object} Response
// @Failure 404 {object} Response "Workflow not found"
// @Failure 500 {object} Response "Server error"
// @Router /api/v1/workflows/{id} [delete]
func (h *Handler) DeleteWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	// Check if workflow exists
	_, err := h.workflowManager.GetWorkflow(workflowID)

	if err != nil {
		if err.Error() == fmt.Sprintf("workflow %s not found", workflowID) {
			h.respondWithError(w, http.StatusNotFound, "Workflow not found")
			return
		}

		h.logger.Error("Failed to get workflow: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to delete workflow")
		return
	}

	// Delete workflow
	if err := h.workflowManager.DeleteWorkflow(workflowID); err != nil {
		h.logger.Error("Failed to delete workflow: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to delete workflow")
		return
	}

	h.logger.Info(fmt.Sprintf("Workflow %s deleted successfully", workflowID))

	h.respondWithJSON(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]string{
			"message": "Workflow deleted successfully",
		},
	})
}

// HealthCheckHandler handles health check requests
// @Summary API health check
// @Description Checks if the API is healthy
// @Tags health
// @Produce json
// @Success 200 {object} Response
// @Failure 500 {object} Response "Server error"
// @Router /health [get]
func (h *Handler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Check Redis connection
	_, err := h.queue.GetQueueStats()

	if err != nil {
		h.logger.Error("Health check failed: " + err.Error())
		h.respondWithJSON(w, http.StatusServiceUnavailable, Response{
			Success: false,
			Error:   "Service unhealthy: " + err.Error(),
		})
		return
	}

	h.respondWithJSON(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]interface{}{
			"status":  "healthy",
			"version": "1.0.0", // Replace with actual version from config
		},
	})
}

// Helper to respond with JSON
func (h *Handler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// Helper to respond with an error
func (h *Handler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.metrics.IncrementErrorCounter(fmt.Sprintf("api_%d", code))
	h.respondWithJSON(w, code, Response{
		Success: false,
		Error:   message,
	})
}
