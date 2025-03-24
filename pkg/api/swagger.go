// pkg/api/swagger.go
package api

import (
	"BoltQ/internal/job"
	"BoltQ/internal/queue"
)

// @title BoltQ API
// @version 1.0.0
// @description Distributed Task Queue API for asynchronous job processing
// @termsOfService http://swagger.io/terms/

// @contact.name BoltQ Support
// @contact.url https://boltq.example.com/support
// @contact.email support@boltq.example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

// Job submission request
type SubmitJobRequest struct {
	Type         string                 `json:"type" example:"echo" description:"Type of job to run"`
	Data         map[string]interface{} `json:"data" example:"{\"message\":\"Hello World\"}" description:"Job parameters"`
	Priority     int                    `json:"priority,omitempty" example:"1" description:"Job priority (0=high, 1=normal, 2=low)"`
	DelaySeconds int                    `json:"delay_seconds,omitempty" example:"60" description:"Delay execution by this many seconds"`
}

// Job submission response
type SubmitJobResponse struct {
	Success bool `json:"success" example:"true"`
	Data    struct {
		JobID string `json:"job_id" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
	} `json:"data"`
}

// Job status response
type JobStatusResponse struct {
	Success bool       `json:"success" example:"true"`
	Data    queue.Task `json:"data"`
}

// Queue stats response
type QueueStatsResponse struct {
	Success bool `json:"success" example:"true"`
	Data    struct {
		TaskQueueHigh   int64 `json:"task_queue:0" example:"5"`
		TaskQueueNormal int64 `json:"task_queue:1" example:"10"`
		TaskQueueLow    int64 `json:"task_queue:2" example:"3"`
		DelayedTasks    int64 `json:"delayed_tasks" example:"7"`
		DeadLetterQueue int64 `json:"dead_letter_queue" example:"2"`
	} `json:"data"`
}

// Workflow creation request
type CreateWorkflowRequest struct {
	Name     string                  `json:"name" example:"Data Processing Pipeline"`
	Steps    []job.WorkflowStepInput `json:"steps"`
	Metadata map[string]interface{}  `json:"metadata,omitempty"`
}

// Workflow creation response
type CreateWorkflowResponse struct {
	Success bool `json:"success" example:"true"`
	Data    struct {
		WorkflowID string `json:"workflow_id" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
	} `json:"data"`
}

// Workflow status response
type WorkflowStatusResponse struct {
	Success bool         `json:"success" example:"true"`
	Data    job.Workflow `json:"data"`
}

// Workflow list response
type WorkflowListResponse struct {
	Success bool `json:"success" example:"true"`
	Data    []struct {
		ID         string `json:"id" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
		Name       string `json:"name" example:"Data Processing Pipeline"`
		Status     string `json:"status" example:"running"`
		CreatedAt  string `json:"created_at" example:"2023-01-01T12:00:00Z"`
		StartedAt  string `json:"started_at,omitempty" example:"2023-01-01T12:01:00Z"`
		FinishedAt string `json:"finished_at,omitempty"`
		StepCount  int    `json:"step_count" example:"3"`
	} `json:"data"`
}

// Error response
type ErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Error   string `json:"error" example:"Job not found"`
}

// Health check response
type HealthResponse struct {
	Success bool `json:"success" example:"true"`
	Data    struct {
		Status  string `json:"status" example:"healthy"`
		Version string `json:"version" example:"1.0.0"`
	} `json:"data"`
}

// SwaggerInfo holds exported Swagger Info so clients can modify it
type SwaggerInfo struct {
	Version     string
	Host        string
	BasePath    string
	Schemes     []string
	Title       string
	Description string
}

// WorkflowStepInput represents input for a workflow step
type WorkflowStepInput struct {
	JobType   string                 `json:"job_type" example:"process_data"`
	Params    map[string]interface{} `json:"params" example:"{\"input_file\":\"data.csv\"}"`
	DependsOn []string               `json:"depends_on,omitempty" example:"[\"step-1\",\"step-2\"]"`
}
