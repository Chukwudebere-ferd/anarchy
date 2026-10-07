package integrations

import (
	"anarchy/internal/character"
	"anarchy/internal/training"
)

// Permissions are least-privilege by default. Personality never grants access.
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
	Permissions        Permissions          `json:"permissions"`
	DisclosureRequired bool                 `json:"disclosure_required"`
	TaskAssignedBy     string               `json:"task_assigned_by"`
}

// DefaultPermissions returns a safe starting point: no tools, approvals required.
func DefaultPermissions() Permissions {
	return Permissions{
		AllowedTools:     []string{},
		NeedApproval:     []string{"destructive", "financial", "privacy-sensitive", "irreversible", "external-post"},
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
		Permissions:        perms,
		DisclosureRequired: true,
		TaskAssignedBy:     "client",
	}
}
