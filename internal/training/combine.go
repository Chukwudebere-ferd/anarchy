package training

import (
	"errors"
	"math/rand"

	"anarchy/internal/character"
	"anarchy/internal/identity"
)

// Deployment is one randomised bot: any identity x any character version.
// Personality and tool permissions stay separate; task is assigned later by the company.
type Deployment struct {
	DeploymentID  string               `json:"deployment_id"`
	Identity      identity.BotIdentity `json:"identity"`
	Character     character.Character  `json:"character"`
	Schedule      string               `json:"schedule"`
	JitterSeconds int                  `json:"jitter_seconds"`
}

// AssignSchedule stamps every deployment with the same schedule window.
// Jitter spreads runs so activity does not land in one burst.
func AssignSchedule(deps []Deployment, schedule string, jitter int) []Deployment {
	for i := range deps {
		deps[i].Schedule = schedule
		deps[i].JitterSeconds = jitter
	}
	return deps
}

// Combine validates then pairs one identity with one character.
func Combine(deploymentID string, id identity.BotIdentity, ch character.Character) (Deployment, error) {
	if deploymentID == "" {
		return Deployment{}, errors.New("training: deployment_id is required")
	}
	if err := id.Validate(); err != nil {
		return Deployment{}, err
	}
	if err := ch.Validate(); err != nil {
		return Deployment{}, err
	}
	if id.Disabled {
		return Deployment{}, errors.New("training: identity is disabled")
	}
	return Deployment{DeploymentID: deploymentID, Identity: id, Character: ch}, nil
}

// RandomPool builds n randomised deployments from pools.
// Same character can land on different identities and vice versa.
func RandomPool(rng *rand.Rand, ids []identity.BotIdentity, chars []character.Character, n int) ([]Deployment, error) {
	if len(ids) == 0 || len(chars) == 0 {
		return nil, errors.New("training: need at least one identity and one character")
	}
	if n < 1 {
		return nil, errors.New("training: n must be >= 1")
	}
	if rng == nil {
		return nil, errors.New("training: rng is required")
	}
	out := make([]Deployment, 0, n)
	idx := 0
	for len(out) < n {
		id := ids[rng.Intn(len(ids))]
		ch := chars[rng.Intn(len(chars))]
		d, err := Combine(randID(rng, idx), id, ch)
		if err != nil {
			// Skip disabled/invalid entries, keep pool moving.
			idx++
			if idx > n*10 {
				return out, errors.New("training: too many invalid identities")
			}
			continue
		}
		out = append(out, d)
		idx++
	}
	return out, nil
}

func randID(rng *rand.Rand, i int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for j := range b {
		b[j] = letters[rng.Intn(len(letters))]
	}
	return "dep-" + string(b)
}
