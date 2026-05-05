package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"BoltQ/pkg/logger"

	"github.com/go-redis/redis/v8"
)

var ErrWorkflowNotFound = errors.New("workflow not found")

const (
	workflowKeyPrefix  = "workflow:"
	workflowQueueKey   = "workflow_queue"
	workflowStatusKey  = "workflow_status"
	workflowStepKey    = "workflow_step:"
	workflowResultsKey = "workflow_results:"
	workflowIndexKey   = "workflow_index"
	workflowTTL        = 72 * time.Hour
)

type WorkflowManager struct {
	redisClient *redis.Client
	logger      *logger.Logger
	ctx         context.Context
	mu          sync.Mutex
}

func NewWorkflowManager(client *redis.Client, logger *logger.Logger) *WorkflowManager {
	return &WorkflowManager{
		redisClient: client,
		logger:      logger,
		ctx:         context.Background(),
	}
}

func (wm *WorkflowManager) SaveWorkflow(workflow *Workflow) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	workflowJSON, err := workflow.ToJSON()
	if err != nil {
		return fmt.Errorf("error serializing workflow: %v", err)
	}

	key := fmt.Sprintf("%s%s", workflowKeyPrefix, workflow.ID)
	if err = wm.redisClient.Set(wm.ctx, key, workflowJSON, workflowTTL).Err(); err != nil {
		return fmt.Errorf("error storing workflow: %v", err)
	}

	statusKey := fmt.Sprintf("%s:%s", workflowStatusKey, workflow.ID)
	if err = wm.redisClient.Set(wm.ctx, statusKey, string(workflow.Status), workflowTTL).Err(); err != nil {
		return fmt.Errorf("error storing workflow status: %v", err)
	}

	if workflow.Status == WorkflowStatusPending {
		// SetNX on a "queued" marker prevents double-enqueuing on repeated saves
		queuedKey := fmt.Sprintf("workflow_queued:%s", workflow.ID)
		added, err := wm.redisClient.SetNX(wm.ctx, queuedKey, "1", workflowTTL).Result()
		if err != nil {
			return fmt.Errorf("error checking workflow queue state: %v", err)
		}
		if added {
			if err = wm.redisClient.LPush(wm.ctx, workflowQueueKey, workflow.ID).Err(); err != nil {
				return fmt.Errorf("error adding workflow to queue: %v", err)
			}
			if err = wm.redisClient.LPush(wm.ctx, workflowIndexKey, workflow.ID).Err(); err != nil {
				return fmt.Errorf("error adding workflow to index: %v", err)
			}
		}
	}

	wm.logger.Info(fmt.Sprintf("Saved workflow %s with status %s", workflow.ID, workflow.Status))
	return nil
}

func (wm *WorkflowManager) GetWorkflow(workflowID string) (*Workflow, error) {
	key := fmt.Sprintf("%s%s", workflowKeyPrefix, workflowID)
	workflowJSON, err := wm.redisClient.Get(wm.ctx, key).Result()

	if err == redis.Nil {
		return nil, ErrWorkflowNotFound
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

func (wm *WorkflowManager) GetNextWorkflow() (*Workflow, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	workflowID, err := wm.redisClient.RPop(wm.ctx, workflowQueueKey).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error retrieving next workflow: %v", err)
	}

	return wm.GetWorkflow(workflowID)
}

// CompleteWorkflowStep marks a step done and re-enqueues the workflow so the
// processor advances to the next ready steps on its next tick.
func (wm *WorkflowManager) CompleteWorkflowStep(workflowID, stepID string, result map[string]interface{}) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	workflow, err := wm.GetWorkflow(workflowID)
	if err != nil {
		return err
	}

	now := time.Now()
	if step, ok := workflow.Steps[stepID]; ok {
		step.Status = StepStatusCompleted
		step.CompletedAt = &now
		if result != nil {
			step.Result = result
		}
	}

	if err := wm.saveWorkflowLocked(workflow); err != nil {
		return err
	}

	return wm.redisClient.LPush(wm.ctx, workflowQueueKey, workflowID).Err()
}

// saveWorkflowLocked persists the workflow; caller must hold wm.mu.
func (wm *WorkflowManager) saveWorkflowLocked(workflow *Workflow) error {
	data, err := json.Marshal(workflow)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s%s", workflowKeyPrefix, workflow.ID)
	if err := wm.redisClient.Set(wm.ctx, key, string(data), workflowTTL).Err(); err != nil {
		return err
	}
	statusKey := fmt.Sprintf("%s:%s", workflowStatusKey, workflow.ID)
	return wm.redisClient.Set(wm.ctx, statusKey, string(workflow.Status), workflowTTL).Err()
}

func (wm *WorkflowManager) SaveStepResult(workflowID, stepID string, result map[string]interface{}) error {
	resultKey := fmt.Sprintf("%s%s:%s", workflowResultsKey, workflowID, stepID)
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("error serializing step result: %v", err)
	}
	return wm.redisClient.Set(wm.ctx, resultKey, string(resultJSON), workflowTTL).Err()
}

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
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return nil, fmt.Errorf("error deserializing step result: %v", err)
	}

	return result, nil
}

func (wm *WorkflowManager) ListWorkflows(limit, offset int) ([]map[string]interface{}, error) {
	start := int64(offset)
	stop := int64(offset + limit - 1)

	ids, err := wm.redisClient.LRange(wm.ctx, workflowIndexKey, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("error listing workflows: %v", err)
	}

	workflows := make([]map[string]interface{}, 0, len(ids))
	for _, workflowID := range ids {
		workflow, err := wm.GetWorkflow(workflowID)
		if err != nil {
			wm.logger.Error(fmt.Sprintf("Error retrieving workflow %s: %v", workflowID, err))
			continue
		}

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

func (wm *WorkflowManager) DeleteWorkflow(workflowID string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	workflow, err := wm.GetWorkflow(workflowID)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s%s", workflowKeyPrefix, workflowID)
	if err = wm.redisClient.Del(wm.ctx, key).Err(); err != nil {
		return fmt.Errorf("error deleting workflow: %v", err)
	}

	statusKey := fmt.Sprintf("%s:%s", workflowStatusKey, workflowID)
	if err = wm.redisClient.Del(wm.ctx, statusKey).Err(); err != nil {
		return fmt.Errorf("error deleting workflow status: %v", err)
	}

	wm.redisClient.Del(wm.ctx, fmt.Sprintf("workflow_queued:%s", workflowID))
	wm.redisClient.LRem(wm.ctx, workflowIndexKey, 0, workflowID)

	for stepID := range workflow.Steps {
		resultKey := fmt.Sprintf("%s%s:%s", workflowResultsKey, workflowID, stepID)
		if err = wm.redisClient.Del(wm.ctx, resultKey).Err(); err != nil {
			wm.logger.Error(fmt.Sprintf("Error deleting step result: %v", err))
		}
	}

	wm.logger.Info(fmt.Sprintf("Deleted workflow %s", workflowID))
	return nil
}
