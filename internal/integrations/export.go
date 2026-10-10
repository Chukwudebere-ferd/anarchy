package integrations

import (
	"anarchy/internal/character"
	"anarchy/internal/training"
)

// Permissions are least-privilege by default. Personality never grants access.
// Bots run scheduled mock-data tasks autonomously; task types listed in
// NeedApproval always require client permission before running.
type Permissions struct {
	AllowedTools []string `json:"allowed_tools"`
	NeedApproval []string `json:"need_approval"`
	RateLimitPerMin int   `json:"rate_limit_per_min"`
}

// ExportableBot is what companies wire into their agent (e.g. OpenCode adapter later).
// Contact details are excluded by default; include only with explicit basis.
type ExportableBot struct {
	FormatVersion      string               `json:"format_version"`
	DeploymentID       string               `json:"deployment_id"`
	IdentityName       string               `json:"identity_name"`
	IdentityID         string               `json:"identity_id"`
	Character          character.Character `json:"character"`
	Schedule           string              `json:"schedule"`
	JitterSeconds      int                 `json:"jitter_seconds"`
	Permissions        Permissions          `json:"permissions"`
	DisclosureRequired bool                 `json:"disclosure_required"`
	TaskAssignedBy     string               `json:"task_assigned_by"`
}

// DefaultPermissions returns a safe starting point: scheduled mock-data
// tasks run freely, sensitive types need client approval. See execution.RequiresApproval.
func DefaultPermissions() Permissions {
	return Permissions{
		AllowedTools:     []string{"mock-seed", "scheduled-task"},
		NeedApproval:     []string{"financial", "destructive", "privacy-sensitive", "irreversible", "external-publish", "real-user-contact"},
		RateLimitPerMin: 10,
	}
}

// ToExport converts a deployment to an agent-ready config.
func ToExport(d training.Deployment, perms Permissions) ExportableBot {
	if perms.AllowedTools == nil {
		perms.AllowedTools = []string{}
	}
	return ExportableBot{
		FormatVersion:      "anarchy-export-v1",
		DeploymentID:       d.DeploymentID,
		IdentityName:       d.Identity.Name,
		IdentityID:         d.Identity.ID,
		Character:          d.Character,
		Schedule:           d.Schedule,
		JitterSeconds:      d.JitterSeconds,
		Permissions:        perms,
		DisclosureRequired: true,
		TaskAssignedBy:     "client",
	}
}
