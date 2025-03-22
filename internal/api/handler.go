// internal/api/handler.go
package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"BoltQ/internal/job"
	"BoltQ/internal/queue"
	"BoltQ/pkg/logger"

	"github.com/gorilla/mux"
)

// API represents the API service
type API struct {
	queue queue.Queue
}

// NewAPI creates a new API instance
func NewAPI(q queue.Queue) *API {
	return &API{queue: q}
}

// JobSubmissionRequest represents a request to submit a new job
type JobSubmissionRequest struct {
	Type        string   `json:"type"`
	Payload     string   `json:"payload"`
	Priority    int      `json:"priority,omitempty"`
	MaxAttempts int      `json:"max_attempts,omitempty"`
	Timeout     int      `json:"timeout,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// SubmitJobHandler handles job submission requests
func (api *API) SubmitJobHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req JobSubmissionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Type == "" || req.Payload == "" {
		http.Error(w, "Job type and payload are required", http.StatusBadRequest)
		return
	}

	// Create a new job
	newJob := job.NewJob(req.Type, req.Payload)

	// Set optional fields if provided
	if req.Priority > 0 {
		newJob.SetPriority(req.Priority)
	}
	if req.MaxAttempts > 0 {
		newJob.SetMaxAttempts(req.MaxAttempts)
	}
	if req.Timeout > 0 {
		newJob.SetTimeout(req.Timeout)
	}
	for _, tag := range req.Tags {
		newJob.AddTag(tag)
	}

	// Convert job to JSON
	jobJSON, err := newJob.ToJSON()
	if err != nil {
		http.Error(w, "Failed to serialize job", http.StatusInternalServerError)
		return
	}

	// Save job status
	err = api.queue.SaveJobStatus(newJob.ID, jobJSON)
	if err != nil {
		http.Error(w, "Failed to save job status", http.StatusInternalServerError)
		return
	}

	// Publish job to queue
	err = api.queue.Publish(jobJSON)
	if err != nil {
		http.Error(w, "Failed to queue job", http.StatusInternalServerError)
		return
	}

	// Log job submission
	msg := fmt.Sprintf("Job submitted: %s, Type: %s, Priority: %d", newJob.ID, newJob.Type, newJob.Priority)
	logger.Info(msg)

	// Return job ID and status
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"job_id": newJob.ID,
		"status": newJob.Status,
	})
}

// GetJobStatusHandler handles requests to check job status
func (api *API) GetJobStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get job ID from URL params
	vars := mux.Vars(r)
	jobID := vars["id"]
	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	// Get job status from Redis
	jobJSON, err := api.queue.GetJobStatus(jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Parse job JSON
	var job job.Job
	if err := json.Unmarshal([]byte(jobJSON), &job); err != nil {
		http.Error(w, "Failed to parse job data", http.StatusInternalServerError)
		return
	}

	// Return job status
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// GetQueueStatsHandler handles requests for queue statistics
func (api *API) GetQueueStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get queue stats
	stats, err := api.queue.GetQueueStats()
	if err != nil {
		http.Error(w, "Failed to retrieve queue statistics", http.StatusInternalServerError)
		return
	}

	// Return stats
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// RetryJobHandler handles requests to retry a dead letter job
func (api *API) RetryJobHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get job ID from URL params
	vars := mux.Vars(r)
	jobID := vars["id"]
	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	// Get job status from Redis
	jobJSON, err := api.queue.GetJobStatus(jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Parse job JSON
	var j job.Job
	if err := json.Unmarshal([]byte(jobJSON), &j); err != nil {
		http.Error(w, "Failed to parse job data", http.StatusInternalServerError)
		return
	}

	// Check if job is in dead letter queue
	if j.Status != job.StatusDeadLetter {
		http.Error(w, "Only jobs in dead letter queue can be retried", http.StatusBadRequest)
		return
	}

	// Reset job for retry
	j.Status = job.StatusRetrying
	j.Attempts = 0
	j.Error = ""
	j.NextRetryAt = time.Now()

	// Convert job back to JSON
	updatedJobJSON, err := json.Marshal(j)
	if err != nil {
		http.Error(w, "Failed to serialize job", http.StatusInternalServerError)
		return
	}

	// Update job status
	err = api.queue.SaveJobStatus(j.ID, string(updatedJobJSON))
	if err != nil {
		http.Error(w, "Failed to update job status", http.StatusInternalServerError)
		return
	}

	// Publish job to retry queue
	err = api.queue.PublishToRetry(string(updatedJobJSON))
	if err != nil {
		http.Error(w, "Failed to queue job for retry", http.StatusInternalServerError)
		return
	}

	// Log job retry
	msg := fmt.Sprintf("Job queued for retry: %s", j.ID)
	logger.Info(msg)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"job_id": j.ID,
		"status": j.Status,
	})
}

// CancelJobHandler handles requests to cancel a pending job
func (api *API) CancelJobHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get job ID from URL params
	vars := mux.Vars(r)
	jobID := vars["id"]
	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	// Get job status from Redis
	jobJSON, err := api.queue.GetJobStatus(jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Parse job JSON
	var j job.Job
	if err := json.Unmarshal([]byte(jobJSON), &j); err != nil {
		http.Error(w, "Failed to parse job data", http.StatusInternalServerError)
		return
	}

	// Check if job can be cancelled (pending or retrying)
	if j.Status != job.StatusPending && j.Status != job.StatusRetrying {
		http.Error(w, "Only pending or retrying jobs can be cancelled", http.StatusBadRequest)
		return
	}

	// Update job status to cancelled
	j.Status = job.StatusCancelled
	j.CompletedAt = time.Now()

	// Convert job back to JSON
	updatedJobJSON, err := json.Marshal(j)
	if err != nil {
		http.Error(w, "Failed to serialize job", http.StatusInternalServerError)
		return
	}

	// Update job status
	err = api.queue.SaveJobStatus(j.ID, string(updatedJobJSON))
	if err != nil {
		http.Error(w, "Failed to update job status", http.StatusInternalServerError)
		return
	}

	// Log job cancellation
	msg := fmt.Sprintf("Job cancelled: %s", j.ID)
	logger.Info(msg)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"job_id": j.ID,
		"status": j.Status,
	})
}

// HealthCheckHandler handles health check requests
func (api *API) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// SetupRoutes configures the API routes
func (api *API) SetupRoutes(router *mux.Router) {
	router.HandleFunc("/api/jobs", api.SubmitJobHandler).Methods("POST")
	router.HandleFunc("/api/jobs/{id}", api.GetJobStatusHandler).Methods("GET")
	router.HandleFunc("/api/jobs/{id}/retry", api.RetryJobHandler).Methods("POST")
	router.HandleFunc("/api/jobs/{id}/cancel", api.CancelJobHandler).Methods("POST")
	router.HandleFunc("/api/stats", api.GetQueueStatsHandler).Methods("GET")
	router.HandleFunc("/health", api.HealthCheckHandler).Methods("GET")
}
