package execution

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// contract maps a platform task type to its required payload fields.
// These 5 are starter shapes only — NOT the full list. Unknown types pass
// through ValidatePayload freely (approval-checked only). Extend per platform
// with RegisterTaskType; e.g. job boards register apply/save_job/post_job.
var contract = map[string][]string{
	"signup":  {"username", "email"},
	"post":    {"author", "body"},
	"comment": {"author", "post_id", "body"},
	"view":    {"target"},
	"follow":  {"follower", "target"},
	// Job-board pack.
	"job_search": {"query"},
	"save_job":   {"user", "job_id"},
	"apply":      {"user", "job_id", "resume"},
	"post_job":   {"employer", "title", "description"},
}

// RegisterTaskType adds (or overrides) a task type's required payload fields,
// so each platform adapts the contract without touching core code.
func RegisterTaskType(taskType string, fields []string) error {
	t := strings.ToLower(strings.TrimSpace(taskType))
	if t == "" {
		return errors.New("execution: type is required")
	}
	if len(fields) == 0 {
		return errors.New("execution: at least one field is required")
	}
	contract[t] = fields
	return nil
}

// KnownTypes returns the supported platform task types, sorted.
func KnownTypes() []string {
	out := append([]string{}, "financial", "destructive", "privacy-sensitive",
		"irreversible", "external-publish", "real-user-contact")
	for t := range contract {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// ValidatePayload checks that payload is a JSON object containing every
// required field for the task type. Unknown platform types pass through
// (validated only for approval); sensitive types always require approval.
func ValidatePayload(taskType, payload string) error {
	required, ok := contract[strings.ToLower(strings.TrimSpace(taskType))]
	if !ok {
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(payload), &obj); err != nil {
		return fmt.Errorf("execution: payload must be a JSON object: %w", err)
	}
	for _, f := range required {
		v, present := obj[f]
		if !present {
			return fmt.Errorf("execution: payload missing required field %q", f)
		}
		if s, isStr := v.(string); isStr && strings.TrimSpace(s) == "" {
			return fmt.Errorf("execution: payload field %q must not be empty", f)
		}
	}
	return nil
}

// CheckTask validates type + payload + approval state together.
func CheckTask(t Task) error {
	if strings.TrimSpace(t.Type) == "" {
		return errors.New("execution: type is required")
	}
	if err := ValidatePayload(t.Type, t.Payload); err != nil {
		return err
	}
	if RequiresApproval(t.Type) && t.ApprovedBy == "" {
		return fmt.Errorf("execution: task type %q needs client approval", t.Type)
	}
	return nil
}
