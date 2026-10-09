package identity

import (
	"errors"
	"strings"
	"time"
)

// BotIdentity is the user-facing + operational data for one bot.
// Email/Phone are sensitive: never log in plaintext, support deletion/disable.
type BotIdentity struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Age         int       `json:"age,omitempty"`
	Email       string    `json:"email,omitempty"`
	Phone       string    `json:"phone,omitempty"`
	Profile     string    `json:"profile,omitempty"`
	Description string    `json:"description,omitempty"`
	Disabled    bool      `json:"disabled"`
	CreatedAt   time.Time `json:"created_at"`
}

func (b BotIdentity) Validate() error {
	if strings.TrimSpace(b.ID) == "" {
		return errors.New("identity: id is required")
	}
	if strings.TrimSpace(b.Name) == "" {
		return errors.New("identity: name is required")
	}
	if b.Email != "" && !strings.Contains(b.Email, "@") {
		return errors.New("identity: invalid email")
	}
	if b.Age < 0 || b.Age > 120 {
		return errors.New("identity: invalid age")
	}
	return nil
}

// Redacted returns a log-safe copy without contact details.
func (b BotIdentity) Redacted() BotIdentity {
	c := b
	if c.Email != "" {
		c.Email = "[redacted]"
	}
	if c.Phone != "" {
		c.Phone = "[redacted]"
	}
	return c
}
