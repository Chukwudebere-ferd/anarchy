package integrations

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"anarchy/internal/character"
	"anarchy/internal/training"
)

// Bundle layout (drop into the company's OpenCode project root):
//
//	bundle/
//	  opencode.json            # project config: permission defaults + instructions
//	  anarchy-rules.md         # approval gate + disclosure rules for the orchestrator
//	  .opencode/agents/*.md    # one worker subagent per character version
//	  anarchy-bots.json        # roster: every bot, identity, schedule, jitter
//	  README.md                # wire-up steps
//
// One agent per character version (not per bot): 3 files cover 542 bots.
// The roster maps each scheduled deployment to its agent.

// agentSlug turns "ferd v2" into "ferd-v2".
func agentSlug(c character.Character) string {
	s := strings.ToLower(strings.TrimSpace(c.Name)) + fmt.Sprintf("-v%d", c.Version)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '_' || r == '-':
			b.WriteRune('-')
		}
	}
	return b.String()
}

// RenderAgent builds the markdown subagent file for one character version.
func RenderAgent(company string, c character.Character, perms Permissions) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "description: Anarchy worker (%s) for %s — scheduled mock-data platform tasks\n", c.Name, company)
	b.WriteString("mode: subagent\n")
	b.WriteString("permission:\n")
	b.WriteString("  edit: deny\n")
	b.WriteString("  bash: ask\n")
	b.WriteString("  webfetch: ask\n")
	b.WriteString("---\n")
	fmt.Fprintf(&b, "You are an Anarchy-trained worker running scheduled platform tasks for %s.\n\n", company)
	fmt.Fprintf(&b, "Behavior: %s\n\n", c.CommunicationStyle)
	b.WriteString("Traits: " + strings.Join(c.Traits, ", ") + "\n\n")
	if c.Backstory != "" {
		fmt.Fprintf(&b, "Backstory: %s\n\n", c.Backstory)
	}
	if len(c.Quirks) > 0 {
		b.WriteString("Quirks (write like this human, not a template):\n")
		for _, q := range c.Quirks {
			b.WriteString("- " + q + "\n")
		}
		b.WriteString("\n")
	}
	if c.Routine != "" {
		fmt.Fprintf(&b, "Daily rhythm: %s\n\n", c.Routine)
	}
	b.WriteString("Goals:\n")
	for _, g := range c.Goals {
		b.WriteString("- " + g + "\n")
	}
	b.WriteString("\nKnowledge:\n")
	for _, k := range c.Knowledge {
		b.WriteString("- " + k + "\n")
	}
	b.WriteString("\nApproval gate — hold and report, do not execute:\n")
	for _, t := range perms.NeedApproval {
		b.WriteString("- " + t + "\n")
	}
	if len(c.Examples) > 0 {
		b.WriteString("\nExamples:\n")
		for _, e := range c.Examples {
			b.WriteString("input: " + e.User + "\n")
			b.WriteString("output: " + e.Bot + "\n")
		}
	}
	b.WriteString(fmt.Sprintf("\nRate limit: %d actions/min. Log every outcome.\n", perms.RateLimitPerMin))
	return b.String()
}

// rosterEntry is one row of anarchy-bots.json.
type rosterEntry struct {
	DeploymentID  string `json:"deployment_id"`
	Agent         string `json:"agent"`
	IdentityID    string `json:"identity_id"`
	IdentityName  string `json:"identity_name"`
	Character     string `json:"character"`
	Version       int    `json:"version"`
	Schedule      string `json:"schedule"`
	JitterSeconds int    `json:"jitter_seconds"`
}

func renderConfig(company string) string {
	return `{
  "$schema": "https://opencode.ai/config.json",
  "permission": {
    "edit": "deny",
    "bash": "ask"
  },
  "instructions": [
    "anarchy-rules.md"
  ]
}
`
}

func renderRules(company string, perms Permissions) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Anarchy rules — %s\n\n", company)
	b.WriteString("Bots below are scheduled workers running mock-data platform tasks.\n\n")
	b.WriteString("## Approval gate\n\n")
	b.WriteString("The following task types HOLD for client approval before running. Stage the action and report it; do not execute:\n\n")
	for _, t := range perms.NeedApproval {
		b.WriteString("- " + t + "\n")
	}
	b.WriteString("\n## Disclosure\n\n")
	b.WriteString("Bot activity must stay within what the platform owner authorized. Disclose bot use wherever end users are affected.\n\n")
	b.WriteString("## Roster\n\n")
	b.WriteString("`anarchy-bots.json` maps every deployment to its worker agent, identity, schedule and jitter. Spread runs across each jitter window.\n")
	return b.String()
}

func renderBundleReadme(company string, nBots, nAgents int) string {
	return fmt.Sprintf(`# Anarchy bundle — %s

%d bots, %d worker agents.

## Wire-up

1. Copy `+"`opencode.json`"+`, `+"`anarchy-rules.md`"+`, `+"`.opencode/`"+` and `+"`anarchy-bots.json`"+` into the project root.
2. Merge `+"`opencode.json`"+` with any existing project config (configs merge; project overrides global).
3. Map platform tools (your MCP server) to each worker: allow scheduled mock-data tools, keep `+"`edit: deny`"+`, sensitive types on `+"`ask`"+`.
4. Schedule: one cron row per roster entry (`+"`schedule`"+` ± `+"`jitter_seconds`"+`). Invoke the row's `+"`agent`"+` with the task payload.
5. Sensitive tasks (`+"`financial`, `destructive`, `privacy-sensitive`, `irreversible`, `external-publish`, `real-user-contact`"+`) hold for approval — stage, report, wait.

## Files

- `+"`.opencode/agents/*.md`"+` — worker subagents (one per character version)
- `+"`anarchy-bots.json`"+` — full roster with schedules
- Regenerate anytime: `+"`anarchy-server.exe --use-db -n <N> --bundle <dir> --company %s`"+`
`, company, nBots, nAgents, company)
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// WriteBundle generates a company-ready OpenCode bundle from deployments.
func WriteBundle(dir, company string, deps []training.Deployment) error {
	perms := DefaultPermissions()
	seen := map[string]character.Character{}
	for _, d := range deps {
		seen[agentSlug(d.Character)] = d.Character
	}
	slugs := make([]string, 0, len(seen))
	for s := range seen {
		slugs = append(slugs, s)
	}
	sort.Strings(slugs)

	if err := writeFile(filepath.Join(dir, "opencode.json"), renderConfig(company)); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(dir, "anarchy-rules.md"), renderRules(company, perms)); err != nil {
		return err
	}
	for _, s := range slugs {
		if err := writeFile(filepath.Join(dir, ".opencode", "agents", s+".md"), RenderAgent(company, seen[s], perms)); err != nil {
			return err
		}
	}
	roster := make([]rosterEntry, 0, len(deps))
	for _, d := range deps {
		roster = append(roster, rosterEntry{
			DeploymentID:  d.DeploymentID,
			Agent:         agentSlug(d.Character),
			IdentityID:    d.Identity.ID,
			IdentityName:  d.Identity.Name,
			Character:     d.Character.Name,
			Version:       d.Character.Version,
			Schedule:      d.Schedule,
			JitterSeconds: d.JitterSeconds,
		})
	}
	raw, _ := json.MarshalIndent(roster, "", "  ")
	if err := writeFile(filepath.Join(dir, "anarchy-bots.json"), string(raw)); err != nil {
		return err
	}
	return writeFile(filepath.Join(dir, "README.md"), renderBundleReadme(company, len(deps), len(slugs)))
}
