package job

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusFailed    WorkflowStatus = "failed"
)

type WorkflowStepStatus string

const (
	StepStatusPending   WorkflowStepStatus = "pending"
	StepStatusRunning   WorkflowStepStatus = "running"
	StepStatusCompleted WorkflowStepStatus = "completed"
	StepStatusFailed    WorkflowStepStatus = "failed"
	StepStatusSkipped   WorkflowStepStatus = "skipped"
)

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

type WorkflowStepInput struct {
	ID        string                 `json:"id,omitempty"`
	JobType   string                 `json:"job_type"`
	Params    map[string]interface{} `json:"params"`
	DependsOn []string               `json:"depends_on,omitempty"`
}

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

func (w *Workflow) AddStep(jobType string, params map[string]interface{}, dependsOn []string) string {
	return w.AddStepWithID("", jobType, params, dependsOn)
}

func (w *Workflow) AddStepWithID(id, jobType string, params map[string]interface{}, dependsOn []string) string {
	stepID := id
	if stepID == "" {
		stepID = uuid.New().String()
	}
	if params == nil {
		params = make(map[string]interface{})
	}

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

func (w *Workflow) GetReadySteps() []*WorkflowStep {
	readySteps := make([]*WorkflowStep, 0)

	for _, stepID := range w.StepOrder {
		step := w.Steps[stepID]
		if step.Status != StepStatusPending {
			continue
		}

		allDepsComplete := true
		for _, depID := range step.DependsOn {
			depStep, exists := w.Steps[depID]
			if !exists || depStep.Status != StepStatusCompleted {
				allDepsComplete = false
				break
			}
		}

		if allDepsComplete {
			readySteps = append(readySteps, step)
		}
	}

	return readySteps
}

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
		if w.Status == WorkflowStatusPending {
			w.Status = WorkflowStatusRunning
			w.StartedAt = &now
		}

	case StepStatusCompleted:
		step.CompletedAt = &now
		step.Result = result

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
		w.Status = WorkflowStatusFailed
		w.FinishedAt = &now
		w.skipDependentSteps(stepID)
	}

	return nil
}

func (w *Workflow) skipDependentSteps(failedStepID string) {
	for _, stepID := range w.StepOrder {
		step := w.Steps[stepID]
		if step.Status != StepStatusPending {
			continue
		}
		for _, depID := range step.DependsOn {
			if depID == failedStepID {
				step.Status = StepStatusSkipped
				step.ErrorMessage = fmt.Sprintf("Skipped because dependency %s failed", failedStepID)
				w.skipDependentSteps(stepID)
				break
			}
		}
	}
}

func (w *Workflow) ToJSON() (string, error) {
	bytes, err := json.Marshal(w)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func WorkflowFromJSON(data string) (*Workflow, error) {
	workflow := &Workflow{}
	if err := json.Unmarshal([]byte(data), workflow); err != nil {
		return nil, err
	}
	return workflow, nil
}
