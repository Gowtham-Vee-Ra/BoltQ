package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"BoltQ/internal/job"
	"BoltQ/internal/queue"
	"BoltQ/pkg/logger"
)

// JobHandler handles job-related API endpoints
type JobHandler struct {
	queue *queue.RedisQueue
}

// NewJobHandler creates a new job handler
func NewJobHandler(q *queue.RedisQueue) *JobHandler {
	return &JobHandler{queue: q}
}

// SubmitJobHandler handles job submission requests
func (h *JobHandler) SubmitJobHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	// Parse the request
	var request struct {
		Type        string                 `json:"type"`
		Payload     map[string]interface{} `json:"payload"`
		MaxAttempts int                    `json:"max_attempts,omitempty"`
	}

	err = json.Unmarshal(body, &request)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate request
	if request.Type == "" {
		http.Error(w, "Job type is required", http.StatusBadRequest)
		return
	}

	// Create a new job
	j := job.NewJob(request.Type, request.Payload)

	// Set max attempts if provided
	if request.MaxAttempts > 0 {
		j.MaxAttempts = request.MaxAttempts
	}

	// Add the job to the queue
	err = h.queue.PublishJob(j)
	if err != nil {
		errMsg := fmt.Sprintf("Error publishing job: %v", err)
		logger.Error(&errMsg)
		http.Error(w, "Error processing job", http.StatusInternalServerError)
		return
	}

	// Return the job ID
	response := map[string]string{
		"job_id": j.ID,
		"status": string(j.Status),
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Error generating response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	w.Write(responseJSON)
}

// GetJobStatusHandler handles job status requests
func (h *JobHandler) GetJobStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract job ID from query parameters
	jobID := r.URL.Query().Get("id")
	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	// Get job status
	j, err := h.queue.GetJobStatus(jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Return the job status
	responseJSON, err := json.Marshal(j)
	if err != nil {
		http.Error(w, "Error generating response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}

// GetQueueStatsHandler returns statistics about the job queues
func (h *JobHandler) GetQueueStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get queue statistics
	counts, err := h.queue.CountJobs()
	if err != nil {
		http.Error(w, "Error fetching queue stats", http.StatusInternalServerError)
		return
	}

	// Return the queue statistics
	responseJSON, err := json.Marshal(counts)
	if err != nil {
		http.Error(w, "Error generating response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}
