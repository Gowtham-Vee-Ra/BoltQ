// internal/api/dashboard.go
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"BoltQ/internal/queue"
	"BoltQ/pkg/logger"

	"github.com/gorilla/mux"
)

// JobStatus constants
const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusRetrying  = "retrying"
)

// DashboardService provides API endpoints for the dashboard UI
type DashboardService struct {
	queue  *queue.RedisQueue
	logger *logger.Logger
}

// NewDashboardService creates a new dashboard service
func NewDashboardService(queue *queue.RedisQueue, logger *logger.Logger) *DashboardService {
	return &DashboardService{
		queue:  queue,
		logger: logger,
	}
}

// JobListItem represents a job item in the list response
type JobListItem struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Priority    int       `json:"priority"`
	ScheduledAt time.Time `json:"scheduled_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListJobsHandler handles requests to list jobs
func (s *DashboardService) ListJobsHandler(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, this would query the database or Redis
	// For simplicity, we'll return a mock response
	jobs := []JobListItem{
		{
			ID:        "job-1",
			Type:      "email",
			Status:    StatusCompleted,
			Priority:  queue.PriorityHigh,
			CreatedAt: time.Now().Add(-1 * time.Hour),
			UpdatedAt: time.Now().Add(-30 * time.Minute),
		},
		{
			ID:          "job-2",
			Type:        "report",
			Status:      StatusPending,
			Priority:    queue.PriorityNormal,
			ScheduledAt: time.Now().Add(30 * time.Minute),
			CreatedAt:   time.Now().Add(-2 * time.Hour),
			UpdatedAt:   time.Now().Add(-2 * time.Hour),
		},
	}

	writeJSON(w, jobs, http.StatusOK)
}

// DashboardStatsResponse contains dashboard stats
type DashboardStatsResponse struct {
	QueueStats      map[string]interface{} `json:"queue_stats"`
	JobCountByType  map[string]int         `json:"job_count_by_type"`
	JobStatusCounts map[string]int         `json:"job_status_counts"`
	RecentJobs      []JobListItem          `json:"recent_jobs"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// DashboardStatsHandler returns stats for the dashboard
func (s *DashboardService) DashboardStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Get queue stats
	queueStats, err := s.queue.GetQueueStats()
	if err != nil {
		s.logger.Error("Failed to get queue stats: " + err.Error())
		writeJSONError(w, "Failed to get queue statistics", http.StatusInternalServerError)
		return
	}

	// Mock data for job counts by type and status
	jobCountByType := map[string]int{
		"email":  25,
		"report": 15,
		"export": 10,
	}

	jobStatusCounts := map[string]int{
		StatusPending:   20,
		StatusRunning:   5,
		StatusCompleted: 40,
		StatusFailed:    3,
		StatusRetrying:  2,
	}

	// Mock data for recent jobs
	recentJobs := []JobListItem{
		{
			ID:        "job-1",
			Type:      "email",
			Status:    StatusCompleted,
			Priority:  queue.PriorityHigh,
			CreatedAt: time.Now().Add(-1 * time.Hour),
			UpdatedAt: time.Now().Add(-30 * time.Minute),
		},
		{
			ID:        "job-2",
			Type:      "report",
			Status:    StatusPending,
			Priority:  queue.PriorityNormal,
			CreatedAt: time.Now().Add(-2 * time.Hour),
			UpdatedAt: time.Now().Add(-2 * time.Hour),
		},
	}

	// Create response
	response := DashboardStatsResponse{
		QueueStats:      queueStats,
		JobCountByType:  jobCountByType,
		JobStatusCounts: jobStatusCounts,
		RecentJobs:      recentJobs,
		UpdatedAt:       time.Now(),
	}

	writeJSON(w, response, http.StatusOK)
}

// RegisterHandlers registers all dashboard API handlers
func (s *DashboardService) RegisterHandlers(router *mux.Router) {
	// API endpoints for dashboard
	router.HandleFunc("/jobs", s.ListJobsHandler).Methods("GET")
	router.HandleFunc("/stats", s.DashboardStatsHandler).Methods("GET")

	// Serve static files for the UI
	fs := http.FileServer(http.Dir("./ui/dist"))
	router.PathPrefix("/").Handler(http.StripPrefix("/", fs))
}

// Helper to write JSON responses
func writeJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// If we can't encode the JSON, fallback to a plain text error
		http.Error(w, "Error encoding JSON response", http.StatusInternalServerError)
	}
}

// Helper to write JSON error responses
func writeJSONError(w http.ResponseWriter, message string, status int) {
	response := map[string]interface{}{
		"success": false,
		"error":   message,
	}

	writeJSON(w, response, status)
}
