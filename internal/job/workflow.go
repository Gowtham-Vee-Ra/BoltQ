// internal/job/workflow.go
package job

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// WorkflowStatus represents the current state of a workflow
type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusFailed    WorkflowStatus = "failed"
)

// WorkflowStepStatus represents the current state of a workflow step
type WorkflowStepStatus string

const (
	StepStatusPending   WorkflowStepStatus = "pending"
	StepStatusRunning   WorkflowStepStatus = "running"
	StepStatusCompleted WorkflowStepStatus = "completed"
	StepStatusFailed    WorkflowStepStatus = "failed"
	StepStatusSkipped   WorkflowStepStatus = "skipped"
)

// WorkflowStep represents a single job in a workflow
type WorkflowStep struct {
	ID           string                 `json:"id"`
	JobType      string                 `json:"job_type"`
	Params       map[string]interface{} `json:"params"`
	DependsOn    []string               `json:"depends_on,omitempty"`
	Status       WorkflowStepStatus     `json:"status"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Result       map[string]interface{} `json:"result,omitempty"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
}

// WorkflowStepInput represents input for a workflow step
type WorkflowStepInput struct {
	JobType   string                 `json:"job_type" example:"process_data"`
	Params    map[string]interface{} `json:"params" example:"{\"input_file\":\"data.csv\"}"`
	DependsOn []string               `json:"depends_on,omitempty" example:"[\"step-1\",\"step-2\"]"`
}

// Workflow represents a collection of jobs that have dependencies between them
type Workflow struct {
	ID         string                   `json:"id"`
	Name       string                   `json:"name"`
	Status     WorkflowStatus           `json:"status"`
	Steps      map[string]*WorkflowStep `json:"steps"`
	StepOrder  []string                 `json:"step_order"`
	CreatedAt  time.Time                `json:"created_at"`
	StartedAt  *time.Time               `json:"started_at,omitempty"`
	FinishedAt *time.Time               `json:"finished_at,omitempty"`
	Metadata   map[string]interface{}   `json:"metadata,omitempty"`
}

// NewWorkflow creates a new workflow with the given name
func NewWorkflow(name string) *Workflow {
	return &Workflow{
		ID:        uuid.New().String(),
		Name:      name,
		Status:    WorkflowStatusPending,
		Steps:     make(map[string]*WorkflowStep),
		StepOrder: make([]string, 0),
		CreatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// AddStep adds a new step to the workflow
func (w *Workflow) AddStep(jobType string, params map[string]interface{}, dependsOn []string) string {
	stepID := uuid.New().String()

	step := &WorkflowStep{
		ID:        stepID,
		JobType:   jobType,
		Params:    params,
		DependsOn: dependsOn,
		Status:    StepStatusPending,
	}

	w.Steps[stepID] = step
	w.StepOrder = append(w.StepOrder, stepID)

	return stepID
}

// GetReadySteps returns all steps that are ready to be executed
func (w *Workflow) GetReadySteps() []*WorkflowStep {
	readySteps := make([]*WorkflowStep, 0)

	for _, stepID := range w.StepOrder {
		step := w.Steps[stepID]

		// Skip steps that are already running, completed, failed, or skipped
		if step.Status != StepStatusPending {
			continue
		}

		// Check if all dependencies are satisfied
		allDependenciesSatisfied := true
		for _, depID := range step.DependsOn {
			depStep, exists := w.Steps[depID]
			if !exists || depStep.Status != StepStatusCompleted {
				allDependenciesSatisfied = false
				break
			}
		}

		if allDependenciesSatisfied {
			readySteps = append(readySteps, step)
		}
	}

	return readySteps
}

// UpdateStepStatus updates the status of a step and potentially the workflow itself
func (w *Workflow) UpdateStepStatus(stepID string, status WorkflowStepStatus, errorMsg string, result map[string]interface{}) error {
	step, exists := w.Steps[stepID]
	if !exists {
		return fmt.Errorf("step %s not found in workflow", stepID)
	}

	step.Status = status

	now := time.Now()

	switch status {
	case StepStatusRunning:
		step.StartedAt = &now
		// If this is the first step to run, update workflow status
		if w.Status == WorkflowStatusPending {
			w.Status = WorkflowStatusRunning
			w.StartedAt = &now
		}

	case StepStatusCompleted:
		step.CompletedAt = &now
		step.Result = result

		// Check if all steps are complete
		allComplete := true
		for _, s := range w.Steps {
			if s.Status != StepStatusCompleted && s.Status != StepStatusSkipped {
				allComplete = false
				break
			}
		}

		if allComplete {
			w.Status = WorkflowStatusCompleted
			w.FinishedAt = &now
		}

	case StepStatusFailed:
		step.CompletedAt = &now
		step.ErrorMessage = errorMsg

		// Mark as failed, but check if we should skip dependent steps
		w.Status = WorkflowStatusFailed
		w.FinishedAt = &now

		// Mark all dependent steps as skipped
		w.skipDependentSteps(stepID)
	}

	return nil
}

// skipDependentSteps marks all steps that depend on the given step as skipped
func (w *Workflow) skipDependentSteps(failedStepID string) {
	for _, stepID := range w.StepOrder {
		step := w.Steps[stepID]

		// Skip steps that are already processed
		if step.Status != StepStatusPending {
			continue
		}

		// Check if this step depends on the failed step
		for _, depID := range step.DependsOn {
			if depID == failedStepID {
				step.Status = StepStatusSkipped
				step.ErrorMessage = fmt.Sprintf("Skipped because dependency %s failed", failedStepID)

				// Recursively skip steps that depend on this one
				w.skipDependentSteps(stepID)
				break
			}
		}
	}
}

// ToJSON serializes the workflow to JSON
func (w *Workflow) ToJSON() (string, error) {
	bytes, err := json.Marshal(w)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// FromJSON deserializes a workflow from JSON
func WorkflowFromJSON(data string) (*Workflow, error) {
	workflow := &Workflow{}
	err := json.Unmarshal([]byte(data), workflow)
	if err != nil {
		return nil, err
	}
	return workflow, nil
}
