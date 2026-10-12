package execution

import (
	"errors"
	"strings"
	"time"
)

// Task is one unit of scheduled platform work for a deployment.
// Bots are cron-like workers, not chatbots: they run platform tasks
// on a schedule using mock data that feels real. Anything with real
// side-effects (money, external publish, destructive, privacy-sensitive
// or irreversible) must carry client approval first.
type Task struct {
	ID            string    `json:"id"`
	DeploymentID  string    `json:"deployment_id"`
	Type          string    `json:"type"`
	Payload       string    `json:"payload,omitempty"`
	Schedule      string    `json:"schedule"`
	NeedsApproval bool      `json:"needs_approval"`
	ApprovedBy    string    `json:"approved_by,omitempty"`
	ApprovedAt    time.Time `json:"approved_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// approvalTypes are task types that always require client permission.
var approvalTypes = []string{
	"financial",
	"destructive",
	"privacy-sensitive",
	"irreversible",
	"external-publish",
	"real-user-contact",
}

// RequiresApproval reports whether a task type needs client permission.
func RequiresApproval(taskType string) bool {
	t := strings.ToLower(strings.TrimSpace(taskType))
	for _, a := range approvalTypes {
		if t == a || strings.HasPrefix(t, a+":") {
			return true
		}
	}
	return false
}

// New builds a task, auto-setting the approval flag from its type.
func New(id, deploymentID, taskType, payload, schedule string) (Task, error) {
	if strings.TrimSpace(id) == "" {
		return Task{}, errors.New("execution: id is required")
	}
	if strings.TrimSpace(deploymentID) == "" {
		return Task{}, errors.New("execution: deployment_id is required")
	}
	if strings.TrimSpace(taskType) == "" {
		return Task{}, errors.New("execution: type is required")
	}
	return Task{
		ID:            id,
		DeploymentID:  deploymentID,
		Type:          taskType,
		Payload:       payload,
		Schedule:      schedule,
		NeedsApproval: RequiresApproval(taskType),
		CreatedAt:     time.Now().UTC(),
	}, nil
}

// Approve records client permission for a sensitive task.
func (t *Task) Approve(by string) error {
	if strings.TrimSpace(by) == "" {
		return errors.New("execution: approver is required")
	}
	t.ApprovedBy = by
	t.ApprovedAt = time.Now().UTC()
	t.NeedsApproval = false
	return nil
}

// Runnable reports whether the task may run now.
func (t Task) Runnable() bool {
	return !RequiresApproval(t.Type) || t.ApprovedBy != ""
}
