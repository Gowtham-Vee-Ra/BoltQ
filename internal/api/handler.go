package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"BoltQ/pkg/logger"
)

// JobQueue defines the interface for a queue implementation
type JobQueue interface {
	Publish(task string) error
	Consume() (string, error)
}

// JobHandler handles job-related API requests
type JobHandler struct {
	queue JobQueue
}

// NewJobHandler creates a new JobHandler with the given queue
func NewJobHandler(queue JobQueue) *JobHandler {
	return &JobHandler{queue: queue}
}

// Job represents a task to be processed
type Job struct {
	ID       string                 `json:"id,omitempty"`
	Type     string                 `json:"type"`
	Payload  map[string]interface{} `json:"payload"`
	Priority int                    `json:"priority,omitempty"`
}

// JobResponse is the API response for job operations
type JobResponse struct {
	Success bool   `json:"success"`
	JobID   string `json:"job_id,omitempty"`
	Message string `json:"message,omitempty"`
}

// SubmitJobHandler processes job submission requests
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

	// Parse the job data
	var job Job
	if err := json.Unmarshal(body, &job); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate the job
	if job.Type == "" {
		http.Error(w, "Job type is required", http.StatusBadRequest)
		return
	}

	// Generate a job ID if not provided
	if job.ID == "" {
		job.ID = fmt.Sprintf("job_%d", generateJobID())
	}

	// Convert the job to JSON
	jobJSON, err := json.Marshal(job)
	if err != nil {
		http.Error(w, "Error serializing job", http.StatusInternalServerError)
		return
	}

	// Publish the job to the queue
	err = h.queue.Publish(string(jobJSON))
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to queue job: %v", err)
		logger.Error(errorMsg)
		http.Error(w, "Error queuing job", http.StatusInternalServerError)
		return
	}

	// Log the job submission
	logMsg := fmt.Sprintf("Job submitted: %s", job.ID)
	logger.Info(logMsg)

	// Return a success response
	response := JobResponse{
		Success: true,
		JobID:   job.ID,
		Message: "Job submitted successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

// JobStatusHandler retrieves the status of a job
func (h *JobHandler) JobStatusHandler(w http.ResponseWriter, r *http.Request) {
	// For Phase 1, we'll return a simple message
	// This will be expanded in Phase 2 with actual status tracking

	w.Header().Set("Content-Type", "application/json")
	response := JobResponse{
		Success: true,
		Message: "Status feature will be implemented in Phase 2",
	}
	json.NewEncoder(w).Encode(response)
}

// Simple ID generator - would be replaced with UUID in production
var jobCounter int64 = 0

func generateJobID() int64 {
	jobCounter++
	return jobCounter
}
