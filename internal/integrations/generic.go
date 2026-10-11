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

// Generic bundle: no OpenCode-specific files. Any agent framework
// (CrewAI, LangChain, AutoGen, custom runners) can consume it:
//
//	bundle/
//	  agents/<slug>.md    # plain system prompt per character version, no frontmatter
//	  roster.json          # every bot: identity, agent, schedule, jitter, approvals
//	  rules.md             # approval gate + disclosure rules
//	  README.md            # wire-up steps

// RenderGenericAgent builds a frontmatter-free system prompt.
func RenderGenericAgent(company string, c character.Character, perms Permissions) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Worker agent — %s (%s)\n\n", c.Name, company)
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

func renderGenericReadme(company string, nBots, nAgents int) string {
	return fmt.Sprintf(`# Anarchy generic bundle — %s

%d bots, %d worker agents. Works with any agent framework.

## Wire-up

1. Use `+"`agents/<name>.md`"+` as the system prompt for your worker.
   One file per character version — point every roster row's `+"`agent`"+`
   at its file.
2. Drive scheduling from `+"`roster.json`"+`: one cron row per entry
   (`+"`schedule`"+` ± `+"`jitter_seconds`"+`), passing the row's
   identity, task type and payload.
3. Enforce `+"`rules.md`"+`: task types under "hold for approval" must
   be staged and reported, never executed, until the named approver
   clears them.
4. Map your platform tools (API clients, MCP servers, SDK calls) to the
   worker: allow scheduled mock-data actions, deny everything sensitive
   by default, keep a human in the loop for holds.

## Files

- `+"`agents/*.md`"+` — worker system prompts (one per character version)
- `+"`roster.json`"+` — full roster with schedules and approval lists
- `+"`rules.md`"+` — approval gate + disclosure rules
`, company, nBots, nAgents)
}

// WriteGenericBundle generates a framework-neutral company bundle.
func WriteGenericBundle(dir, company string, deps []training.Deployment) error {
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

	for _, s := range slugs {
		content := RenderGenericAgent(company, seen[s], perms)
		path := filepath.Join(dir, "agents", s+".md")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
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
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "roster.json"), raw, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "rules.md"), []byte(renderRules(company, perms)), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "README.md"), []byte(renderGenericReadme(company, len(deps), len(slugs))), 0644)
}
