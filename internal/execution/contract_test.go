package execution

import "testing"

func TestSensitiveTypesAlwaysHold(t *testing.T) {
	for _, typ := range []string{"financial", "destructive", "privacy-sensitive",
		"irreversible", "external-publish", "real-user-contact", "financial:refund"} {
		task, err := New("t", "dep", typ, `{}`, "@every 1h")
		if err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
		if !task.NeedsApproval || task.Runnable() {
			t.Fatalf("%s must hold unapproved", typ)
		}
		if err := CheckTask(task); err == nil {
			t.Fatalf("%s must fail CheckTask unapproved", typ)
		}
		if err := task.Approve("owner"); err != nil {
			t.Fatal(err)
		}
		if !task.Runnable() {
			t.Fatalf("%s must run after approval", typ)
		}
	}
}

func TestRoutineTypesRunFreely(t *testing.T) {
	cases := map[string]string{
		"signup": `{"username":"ada","email":"a@b.co"}`,
		"post":   `{"author":"a","body":"hello"}`,
		"apply":  `{"user":"a","job_id":"j1","resume":"r"}`,
	}
	for typ, payload := range cases {
		task, err := New("t", "dep", typ, payload, "@every 1h")
		if err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
		if task.NeedsApproval || !task.Runnable() {
			t.Fatalf("%s must run freely", typ)
		}
		if err := CheckTask(task); err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
	}
}

func TestValidatePayloadMissingFields(t *testing.T) {
	if err := ValidatePayload("signup", `{"username":"a"}`); err == nil {
		t.Fatal("missing email must fail")
	}
	if err := ValidatePayload("signup", `{"username":"","email":"x@y.z"}`); err == nil {
		t.Fatal("empty username must fail")
	}
	if err := ValidatePayload("signup", `not-json`); err == nil {
		t.Fatal("non-object payload must fail")
	}
}

func TestUnknownTypePassesThrough(t *testing.T) {
	if err := ValidatePayload("gig_bid", `anything at all`); err != nil {
		t.Fatalf("custom type must pass through: %v", err)
	}
}

func TestRegisterTaskType(t *testing.T) {
	t.Cleanup(func() { delete(contract, "test_custom") })
	if err := ValidatePayload("test_custom", `anything`); err != nil {
		t.Fatal("unregistered type must pass through")
	}
	if err := RegisterTaskType("test_custom", []string{"user", "gig_id"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePayload("test_custom", `{"user":"a"}`); err == nil {
		t.Fatal("missing gig_id must fail after registration")
	}
	if err := ValidatePayload("test_custom", `{"user":"a","gig_id":"g"}`); err != nil {
		t.Fatal(err)
	}
	if err := RegisterTaskType("", []string{"x"}); err == nil {
		t.Fatal("empty type must be rejected")
	}
	if err := RegisterTaskType("also_empty", nil); err == nil {
		t.Fatal("empty fields must be rejected")
	}
}

func TestParseSchedule(t *testing.T) {
	if d, err := ParseSchedule("@every 2h"); err != nil || d.String() != "2h0m0s" {
		t.Fatalf("@every 2h: %v %v", d, err)
	}
	if d, err := ParseSchedule("90"); err != nil || d.String() != "1m30s" {
		t.Fatalf("90: %v %v", d, err)
	}
	for _, bad := range []string{"", "@every -5m", "@every 0", "0", "-3", "nonsense"} {
		if _, err := ParseSchedule(bad); err == nil {
			t.Fatalf("%q must be rejected", bad)
		}
	}
}
