package tasklifecycle

import "testing"

func TestAllowedTaskStatusTransitionMatrix(t *testing.T) {
	allowed := [][2]string{
		{"backlog", "todo"},
		{"backlog", "archived"},
		{"todo", "doing"},
		{"todo", "blocked"},
		{"todo", "backlog"},
		{"todo", "archived"},
		{"needs_rework", "doing"},
		{"needs_rework", "blocked"},
		{"needs_rework", "todo"},
		{"needs_rework", "archived"},
		{"doing", "needs_review"},
		{"doing", "needs_rework"},
		{"doing", "blocked"},
		{"doing", "todo"},
		{"doing", "done"},
		{"blocked", "needs_review"},
		{"blocked", "needs_rework"},
		{"blocked", "todo"},
		{"blocked", "archived"},
		{"needs_review", "needs_rework"},
		{"needs_review", "todo"},
		{"needs_review", "done"},
		{"needs_review", "archived"},
		{"done", "archived"},
		{"archived", "backlog"},
		{"todo", "todo"},
	}

	for _, transition := range allowed {
		if !AllowedTaskStatusTransition(transition[0], transition[1]) {
			t.Fatalf("%s -> %s should be allowed", transition[0], transition[1])
		}
	}
}

func TestAllowedTaskStatusTransitionRejectsUnknownTransitions(t *testing.T) {
	rejected := [][2]string{
		{"backlog", "doing"},
		{"needs_rework", "done"},
		{"done", "todo"},
		{"archived", "done"},
		{"blocked", "doing"},
		{"blocked", "done"},
		{"unknown", "todo"},
		{"todo", "unknown"},
	}

	for _, transition := range rejected {
		if AllowedTaskStatusTransition(transition[0], transition[1]) {
			t.Fatalf("%s -> %s should be rejected", transition[0], transition[1])
		}
	}
}

func TestAllowedNextStatusesFollowsCanonicalOrder(t *testing.T) {
	got := AllowedNextStatuses("needs_review")
	want := []string{"needs_rework", "todo", "done", "archived"}
	if len(got) != len(want) {
		t.Fatalf("needs_review next = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("needs_review next = %#v, want %#v", got, want)
		}
	}
	if next := AllowedNextStatuses("done"); len(next) != 1 || next[0] != "archived" {
		t.Fatalf("done next = %#v, want [archived]", next)
	}
}
