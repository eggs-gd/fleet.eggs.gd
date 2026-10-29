package taskflow

import (
	"context"
	"errors"
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
		{"needs_rework", ExecutionNeedsRework, true},
		{"blocked", ExecutionBlocked, true},
		{"cancelled", ExecutionCancelled, true},
		{"timed_out", ExecutionTimedOut, true},
		{"orphaned", ExecutionOrphaned, true},
		{"waiting_input", "", false},
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
