package character

import (
	"errors"
	"strings"
	"time"
)

// Example is a single demonstration response.
type Example struct {
	User string `json:"user"`
	Bot  string `json:"bot"`
}

// Character is structured behavior data, versioned per bot line.
// Identity (who) and Character (how it behaves) are separate so they can be
// randomised: e.g. identity "ferd" + character v2, identity "sophia" + "ferd" character.
// NPC-style: backstory, quirks and routine make each line read like a distinct human.
type Character struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Version            int       `json:"version"`
	Traits             []string  `json:"traits"`
	CommunicationStyle string    `json:"communication_style"`
	Backstory          string    `json:"backstory,omitempty"`
	Quirks             []string  `json:"quirks,omitempty"`
	Routine            string    `json:"routine,omitempty"`
	Goals              []string  `json:"goals"`
	Knowledge          []string  `json:"knowledge"`
	Boundaries         []string  `json:"boundaries,omitempty"`
	Examples           []Example `json:"examples"`
	CreatedAt          time.Time `json:"created_at"`
}

var blockedSubstrings = []string{
	"ignore previous instructions",
	"ignore all instructions",
	"reveal system prompt",
	"disregard all prior instructions",
}

func containsBlocked(s string) bool {
	l := strings.ToLower(s)
	for _, b := range blockedSubstrings {
		if strings.Contains(l, b) {
			return true
		}
	}
	return false
}

func (c Character) Validate() error {
	if strings.TrimSpace(c.ID) == "" {
		return errors.New("character: id is required")
	}
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("character: name is required")
	}
	if c.Version < 1 {
		return errors.New("character: version must be >= 1")
	}
	if len(c.Traits) == 0 {
		return errors.New("character: at least one trait is required")
	}
	// Prompt-injection / persona-manipulation screening.
	fields := append([]string{c.CommunicationStyle}, c.Traits...)
	fields = append(fields, c.Goals...)
	fields = append(fields, c.Knowledge...)
	fields = append(fields, c.Boundaries...)
	for _, e := range c.Examples {
		fields = append(fields, e.User, e.Bot)
	}
	for _, f := range fields {
		if containsBlocked(f) {
			return errors.New("character: field contains blocked instruction-like content")
		}
	}
	return nil
}
