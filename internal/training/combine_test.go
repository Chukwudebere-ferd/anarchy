package training

import (
	"math/rand"
	"testing"

	"anarchy/internal/character"
	"anarchy/internal/identity"
)

func nil2() *rand.Rand { return rand.New(rand.NewSource(1)) }

func testIdentity(id string, disabled bool) identity.BotIdentity {
	return identity.BotIdentity{ID: id, Name: "Ferd " + id, Email: "f@x.co", Disabled: disabled}
}

func testCharacter(id string) character.Character {
	return character.Character{
		ID: id, Name: "line-" + id, Version: 1,
		Traits:             []string{"calm"},
		CommunicationStyle: "steady",
		Goals:              []string{"complete tasks"},
		Knowledge:          []string{"onboarding"},
	}
}

func TestRandomPoolSkipsDisabled(t *testing.T) {
	ids := []identity.BotIdentity{testIdentity("on1", false), testIdentity("off1", true)}
	chars := []character.Character{testCharacter("c1")}
	deps, err := RandomPool(nil2(), ids, chars, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 20 {
		t.Fatalf("want 20, got %d", len(deps))
	}
	for _, d := range deps {
		if d.Identity.Disabled {
			t.Fatal("disabled identity must never be deployed")
		}
	}
}

func TestRandomPoolRejectsEmptyPools(t *testing.T) {
	if _, err := RandomPool(nil2(), nil, []character.Character{testCharacter("c")}, 1); err == nil {
		t.Fatal("empty identities must fail")
	}
	if _, err := RandomPool(nil2(), []identity.BotIdentity{testIdentity("i", false)}, nil, 1); err == nil {
		t.Fatal("empty characters must fail")
	}
	if _, err := RandomPool(nil2(), []identity.BotIdentity{testIdentity("i", false)}, []character.Character{testCharacter("c")}, 0); err == nil {
		t.Fatal("n<1 must fail")
	}
}

func TestCombineValidation(t *testing.T) {
	if _, err := Combine("", testIdentity("i", false), testCharacter("c")); err == nil {
		t.Fatal("empty deployment id must fail")
	}
	if _, err := Combine("d", testIdentity("i", true), testCharacter("c")); err == nil {
		t.Fatal("disabled identity must fail combine")
	}
}

func TestAssignScheduleStampsAll(t *testing.T) {
	deps := []Deployment{{DeploymentID: "a"}, {DeploymentID: "b"}}
	out := AssignSchedule(deps, "@every 5m", 60)
	for _, d := range out {
		if d.Schedule != "@every 5m" || d.JitterSeconds != 60 {
			t.Fatalf("stamping failed on %s", d.DeploymentID)
		}
	}
}
