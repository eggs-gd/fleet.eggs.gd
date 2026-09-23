package taskflow

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNormalizeExecutionOutcome(t *testing.T) {
	cases := []struct {
		raw string
		out ExecutionOutcome
		ok  bool
	}{
		{"completed", ExecutionCompleted, true},
		{"failed", ExecutionFailed, true},
		{"needs_input", ExecutionNeedsInput, true},
		{"waiting_input", ExecutionNeedsInput, true},
		{"needs_rework", ExecutionNeedsRework, true},
		{"blocked", ExecutionBlocked, true},
		{"cancelled", ExecutionCancelled, true},
		{"timed_out", ExecutionTimedOut, true},
		{"orphaned", ExecutionOrphaned, true},
		{"<completed|failed>", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := NormalizeExecutionOutcome(tc.raw)
		if ok != tc.ok || got != tc.out {
			t.Fatalf("%q: got (%q,%v) want (%q,%v)", tc.raw, got, ok, tc.out, tc.ok)
		}
	}
}

func TestParseExecutionResultCanonicalAndLegacyShapes(t *testing.T) {
	canonical := ParseExecutionResult(`Done.
{"outcome":"needs_input","summary":"Need a token","question":"What is CORE_STAGING_TOKEN?","error":"missing secret"}
`)
	if canonical == nil {
		t.Fatal("expected canonical parse")
	}
	if canonical.Outcome != ExecutionNeedsInput {
		t.Fatalf("outcome = %q", canonical.Outcome)
	}
	if canonical.Question != "What is CORE_STAGING_TOKEN?" {
		t.Fatalf("question = %q", canonical.Question)
	}

	legacy := ParseExecutionResult(`{"outcome":"waiting_input","summary":"Need approval","blockers":["missing sign-off"],"review_notes":"ask Alex","suggested_next_status":"todo"}`)
	if legacy == nil {
		t.Fatal("expected legacy parse")
	}
	if legacy.Outcome != ExecutionNeedsInput {
		t.Fatalf("legacy outcome = %q", legacy.Outcome)
	}
	if legacy.Question != "missing sign-off" {
		t.Fatalf("legacy question = %q", legacy.Question)
	}
	if !strings.Contains(legacy.Summary, "ask Alex") {
		t.Fatalf("summary missing review notes: %q", legacy.Summary)
	}
	if !strings.Contains(legacy.Summary, "Worker-suggested next status") {
		t.Fatalf("summary missing suggested status fold: %q", legacy.Summary)
	}
}

func TestParseExecutionResultIgnoresPromptPlaceholder(t *testing.T) {
	text := `Instructions:
{"outcome": "<completed|failed|needs_input|needs_rework|blocked>", "summary": "..."}
Real result later:
{"outcome":"completed","summary":"shipped","artifacts":["a.go"]}
`
	got := ParseExecutionResult(text)
	if got == nil || got.Outcome != ExecutionCompleted || got.Summary != "shipped" {
		t.Fatalf("got = %#v", got)
	}
}

func TestServiceClaimMovesPickupToDoing(t *testing.T) {
	provider := newMemoryProvider()
	svc := NewService(provider)
	defer svc.Close()

	task, err := svc.Create(context.Background(), CreateTask{Title: "claim me", Status: StatusTodo})
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := svc.Claim(context.Background(), task.Locator)
	if err != nil {
		t.Fatal(err)
	}
	if claimed.Status != StatusDoing {
		t.Fatalf("status = %s, want doing", claimed.Status)
	}

	_, err = svc.Claim(context.Background(), task.Locator)
	if !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("err = %v, want ErrAlreadyClaimed", err)
	}
}
