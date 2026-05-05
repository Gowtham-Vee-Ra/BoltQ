package api

import (
	"encoding/json"
	"errors"
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

type Handler struct {
	queue           *queue.RedisQueue
	logger          *logger.Logger
	metrics         *metrics.MetricsCollector
	workflowManager *job.WorkflowManager
}

func NewHandler(queue *queue.RedisQueue, logger *logger.Logger, metrics *metrics.MetricsCollector,
	workflowManager *job.WorkflowManager) *Handler {
	return &Handler{
		queue:           queue,
		logger:          logger,
		metrics:         metrics,
		workflowManager: workflowManager,
	}
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type SubmitJobRequest struct {
	Type         string                 `json:"type"`
	Data         map[string]interface{} `json:"data"`
	Priority     int                    `json:"priority,omitempty"`
	DelaySeconds int                    `json:"delay_seconds,omitempty"`
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/jobs", h.ListJobsHandler).Methods("GET")
	r.HandleFunc("/api/v1/jobs", h.SubmitJobHandler).Methods("POST")
	r.HandleFunc("/api/v1/jobs/{id}", h.GetJobStatusHandler).Methods("GET")
	r.HandleFunc("/api/v1/jobs/{id}/cancel", h.CancelJobHandler).Methods("POST")

	r.HandleFunc("/api/v1/queues/stats", h.GetQueueStatsHandler).Methods("GET")

	r.HandleFunc("/api/v1/workflows", h.CreateWorkflowHandler).Methods("POST")
	r.HandleFunc("/api/v1/workflows", h.ListWorkflowsHandler).Methods("GET")
	r.HandleFunc("/api/v1/workflows/{id}", h.GetWorkflowHandler).Methods("GET")
	r.HandleFunc("/api/v1/workflows/{id}", h.DeleteWorkflowHandler).Methods("DELETE")

	r.HandleFunc("/health", h.HealthCheckHandler).Methods("GET")
}

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

	if req.Type == "" {
		h.respondWithError(w, http.StatusBadRequest, "Job type is required")
		return
	}

	task := &queue.Task{
		ID:        uuid.New().String(),
		Type:      req.Type,
		Data:      req.Data,
		Priority:  req.Priority,
		CreatedAt: time.Now(),
		Status:    "pending",
	}

	var err error
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

	h.respondWithJSON(w, http.StatusCreated, Response{
		Success: true,
		Data:    map[string]string{"job_id": task.ID},
	})
}

func (h *Handler) ListJobsHandler(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20
	offset := 0

	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	tasks, err := h.queue.ListJobs(limit, offset)
	if err != nil {
		h.logger.Error("Failed to list jobs: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to list jobs")
		return
	}

	h.respondWithJSON(w, http.StatusOK, Response{Success: true, Data: tasks})
}

func (h *Handler) GetJobStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	task, err := h.queue.GetTaskStatus(jobID)
	if err != nil {
		if errors.Is(err, queue.ErrTaskNotFound) {
			h.respondWithError(w, http.StatusNotFound, "Job not found")
			return
		}
		h.logger.Error("Failed to get job status: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get job status")
		return
	}

	h.respondWithJSON(w, http.StatusOK, Response{Success: true, Data: task})
}

func (h *Handler) CancelJobHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	task, err := h.queue.GetTaskStatus(jobID)
	if err != nil {
		if errors.Is(err, queue.ErrTaskNotFound) {
			h.respondWithError(w, http.StatusNotFound, "Job not found")
			return
		}
		h.logger.Error("Failed to get job status: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get job status")
		return
	}

	if task.Status != "pending" && task.Status != "scheduled" {
		h.respondWithError(w, http.StatusBadRequest, "Only pending or scheduled jobs can be cancelled")
		return
	}

	if _, err := h.queue.RemoveFromQueue(task.ID); err != nil {
		h.logger.Error("Failed to remove job from queue: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to cancel job")
		return
	}

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
		Data:    map[string]string{"status": "cancelled"},
	})
}

func (h *Handler) GetQueueStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := h.queue.GetQueueStats()
	if err != nil {
		h.logger.Error("Failed to get queue stats: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get queue statistics")
		return
	}

	h.respondWithJSON(w, http.StatusOK, Response{Success: true, Data: stats})
}

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

	if req.Name == "" {
		h.respondWithError(w, http.StatusBadRequest, "Workflow name is required")
		return
	}
	if len(req.Steps) == 0 {
		h.respondWithError(w, http.StatusBadRequest, "Workflow must have at least one step")
		return
	}

	workflow := job.NewWorkflow(req.Name)
	if req.Metadata != nil {
		workflow.Metadata = req.Metadata
	}

	// caller-provided IDs allow dependencies to be expressed client-side
	for _, stepInput := range req.Steps {
		workflow.AddStepWithID(stepInput.ID, stepInput.JobType, stepInput.Params, stepInput.DependsOn)
	}

	if err := h.workflowManager.SaveWorkflow(workflow); err != nil {
		h.logger.Error("Failed to save workflow: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to create workflow")
		return
	}

	h.logger.Info(fmt.Sprintf("Workflow %s created successfully with %d steps", workflow.ID, len(workflow.Steps)))

	h.respondWithJSON(w, http.StatusCreated, Response{
		Success: true,
		Data:    map[string]string{"workflow_id": workflow.ID},
	})
}

func (h *Handler) GetWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	workflow, err := h.workflowManager.GetWorkflow(workflowID)
	if err != nil {
		if errors.Is(err, job.ErrWorkflowNotFound) {
			h.respondWithError(w, http.StatusNotFound, "Workflow not found")
			return
		}
		h.logger.Error("Failed to get workflow: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to get workflow")
		return
	}

	h.respondWithJSON(w, http.StatusOK, Response{Success: true, Data: workflow})
}

func (h *Handler) ListWorkflowsHandler(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20
	offset := 0

	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	workflows, err := h.workflowManager.ListWorkflows(limit, offset)
	if err != nil {
		h.logger.Error("Failed to list workflows: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to list workflows")
		return
	}

	h.respondWithJSON(w, http.StatusOK, Response{Success: true, Data: workflows})
}

func (h *Handler) DeleteWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["id"]

	_, err := h.workflowManager.GetWorkflow(workflowID)
	if err != nil {
		if errors.Is(err, job.ErrWorkflowNotFound) {
			h.respondWithError(w, http.StatusNotFound, "Workflow not found")
			return
		}
		h.logger.Error("Failed to get workflow: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to delete workflow")
		return
	}

	if err := h.workflowManager.DeleteWorkflow(workflowID); err != nil {
		h.logger.Error("Failed to delete workflow: " + err.Error())
		h.respondWithError(w, http.StatusInternalServerError, "Failed to delete workflow")
		return
	}

	h.logger.Info(fmt.Sprintf("Workflow %s deleted successfully", workflowID))

	h.respondWithJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    map[string]string{"message": "Workflow deleted successfully"},
	})
}

func (h *Handler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
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
		Data:    map[string]interface{}{"status": "healthy", "version": "1.0.0"},
	})
}

func (h *Handler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func (h *Handler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.metrics.IncrementErrorCounter(fmt.Sprintf("api_%d", code))
	h.respondWithJSON(w, code, Response{Success: false, Error: message})
}
