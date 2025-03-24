// internal/job/workflow_manager.go
package job

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"BoltQ/pkg/logger"

	"github.com/go-redis/redis/v8"
)

const (
	// Redis keys for workflow storage
	workflowKeyPrefix  = "workflow:"
	workflowQueueKey   = "workflow_queue"
	workflowStatusKey  = "workflow_status"
	workflowStepKey    = "workflow_step:"
	workflowResultsKey = "workflow_results:"
	workflowTTL        = 72 * time.Hour
)

// WorkflowManager handles workflow operations and persistence
type WorkflowManager struct {
	redisClient *redis.Client
	logger      *logger.Logger
	ctx         context.Context
	mu          sync.Mutex
}

// NewWorkflowManager creates a new workflow manager
func NewWorkflowManager(client *redis.Client, logger *logger.Logger) *WorkflowManager {
	return &WorkflowManager{
		redisClient: client,
		logger:      logger,
		ctx:         context.Background(),
	}
}

// SaveWorkflow stores a workflow in Redis
func (wm *WorkflowManager) SaveWorkflow(workflow *Workflow) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Convert workflow to JSON
	workflowJSON, err := workflow.ToJSON()
	if err != nil {
		return fmt.Errorf("error serializing workflow: %v", err)
	}

	// Store workflow data
	key := fmt.Sprintf("%s%s", workflowKeyPrefix, workflow.ID)
	err = wm.redisClient.Set(wm.ctx, key, workflowJSON, workflowTTL).Err()
	if err != nil {
		return fmt.Errorf("error storing workflow: %v", err)
	}

	// Store workflow status for quick access
	statusKey := fmt.Sprintf("%s:%s", workflowStatusKey, workflow.ID)
	err = wm.redisClient.Set(wm.ctx, statusKey, string(workflow.Status), workflowTTL).Err()
	if err != nil {
		return fmt.Errorf("error storing workflow status: %v", err)
	}

	// If workflow is pending, add to queue
	if workflow.Status == WorkflowStatusPending {
		err = wm.redisClient.LPush(wm.ctx, workflowQueueKey, workflow.ID).Err()
		if err != nil {
			return fmt.Errorf("error adding workflow to queue: %v", err)
		}
	}

	wm.logger.Info(fmt.Sprintf("Saved workflow %s with status %s", workflow.ID, workflow.Status))
	return nil
}

// GetWorkflow retrieves a workflow from Redis
func (wm *WorkflowManager) GetWorkflow(workflowID string) (*Workflow, error) {
	key := fmt.Sprintf("%s%s", workflowKeyPrefix, workflowID)
	workflowJSON, err := wm.redisClient.Get(wm.ctx, key).Result()

	if err == redis.Nil {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}

	if err != nil {
		return nil, fmt.Errorf("error retrieving workflow: %v", err)
	}

	workflow, err := WorkflowFromJSON(workflowJSON)
	if err != nil {
		return nil, fmt.Errorf("error deserializing workflow: %v", err)
	}

	return workflow, nil
}

// GetNextWorkflow gets the next pending workflow from the queue
func (wm *WorkflowManager) GetNextWorkflow() (*Workflow, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Pop next workflow ID from queue
	workflowID, err := wm.redisClient.RPop(wm.ctx, workflowQueueKey).Result()

	if err == redis.Nil {
		return nil, nil // No workflows in queue
	}

	if err != nil {
		return nil, fmt.Errorf("error retrieving next workflow: %v", err)
	}

	// Get the workflow
	return wm.GetWorkflow(workflowID)
}

// SaveStepResult stores a step's result in Redis
func (wm *WorkflowManager) SaveStepResult(workflowID, stepID string, result map[string]interface{}) error {
	resultKey := fmt.Sprintf("%s%s:%s", workflowResultsKey, workflowID, stepID)

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("error serializing step result: %v", err)
	}

	err = wm.redisClient.Set(wm.ctx, resultKey, string(resultJSON), workflowTTL).Err()
	if err != nil {
		return fmt.Errorf("error storing step result: %v", err)
	}

	return nil
}

// GetStepResult retrieves a step's result from Redis
func (wm *WorkflowManager) GetStepResult(workflowID, stepID string) (map[string]interface{}, error) {
	resultKey := fmt.Sprintf("%s%s:%s", workflowResultsKey, workflowID, stepID)

	resultJSON, err := wm.redisClient.Get(wm.ctx, resultKey).Result()

	if err == redis.Nil {
		return nil, fmt.Errorf("result for step %s in workflow %s not found", stepID, workflowID)
	}

	if err != nil {
		return nil, fmt.Errorf("error retrieving step result: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal([]byte(resultJSON), &result)
	if err != nil {
		return nil, fmt.Errorf("error deserializing step result: %v", err)
	}

	return result, nil
}

// ListWorkflows retrieves a list of workflow IDs with their status
func (wm *WorkflowManager) ListWorkflows(limit, offset int) ([]map[string]interface{}, error) {
	// Get workflow keys with pagination
	pattern := fmt.Sprintf("%s*", workflowKeyPrefix)
	keys, _, err := wm.redisClient.Scan(wm.ctx, uint64(offset), pattern, int64(limit)).Result()

	if err != nil {
		return nil, fmt.Errorf("error scanning workflows: %v", err)
	}

	workflows := make([]map[string]interface{}, 0, len(keys))

	for _, key := range keys {
		// Extract workflow ID from key
		workflowID := key[len(workflowKeyPrefix):]

		// Get workflow data
		workflow, err := wm.GetWorkflow(workflowID)
		if err != nil {
			wm.logger.Error(fmt.Sprintf("Error retrieving workflow %s: %v", workflowID, err))
			continue
		}

		// Create summarized info
		summary := map[string]interface{}{
			"id":         workflow.ID,
			"name":       workflow.Name,
			"status":     workflow.Status,
			"created_at": workflow.CreatedAt,
			"step_count": len(workflow.Steps),
		}

		if workflow.StartedAt != nil {
			summary["started_at"] = workflow.StartedAt
		}

		if workflow.FinishedAt != nil {
			summary["finished_at"] = workflow.FinishedAt
		}

		workflows = append(workflows, summary)
	}

	return workflows, nil
}

// DeleteWorkflow removes a workflow and its data from Redis
func (wm *WorkflowManager) DeleteWorkflow(workflowID string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Get workflow first to get step IDs
	workflow, err := wm.GetWorkflow(workflowID)
	if err != nil {
		return err
	}

	// Delete workflow data
	key := fmt.Sprintf("%s%s", workflowKeyPrefix, workflowID)
	err = wm.redisClient.Del(wm.ctx, key).Err()
	if err != nil {
		return fmt.Errorf("error deleting workflow: %v", err)
	}

	// Delete workflow status
	statusKey := fmt.Sprintf("%s:%s", workflowStatusKey, workflowID)
	err = wm.redisClient.Del(wm.ctx, statusKey).Err()
	if err != nil {
		return fmt.Errorf("error deleting workflow status: %v", err)
	}

	// Delete step results
	for stepID := range workflow.Steps {
		resultKey := fmt.Sprintf("%s%s:%s", workflowResultsKey, workflowID, stepID)
		err = wm.redisClient.Del(wm.ctx, resultKey).Err()
		if err != nil {
			wm.logger.Error(fmt.Sprintf("Error deleting step result: %v", err))
		}
	}

	wm.logger.Info(fmt.Sprintf("Deleted workflow %s", workflowID))
	return nil
}
