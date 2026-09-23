package tasklifecycle

import "testing"

func TestLaunchableCandidatesForAssigneeSortsByPriorityThenRef(t *testing.T) {
	tasks := []Task{
		{Ref: "CORE-10", Status: "todo", Priority: 3, Assignee: "codex"},
		{Ref: "CORE-2", Status: "todo", Priority: 3, Assignee: "codex"},
		{Ref: "CORE-9", Status: "todo", Priority: 1, Assignee: "codex"},
		{Ref: "CORE-8", Status: "needs_rework", Priority: 5, Assignee: "codex"},
		{Ref: "CORE-1", Status: "backlog", Priority: 1, Assignee: "codex"},
		{Ref: "CORE-3", Status: "todo", Priority: 1, Assignee: "claude"},
		{Ref: "CORE-4", Status: "todo", Priority: 1, Assignee: "codex"},
	}

	candidates := LaunchableCandidatesForAssignee(tasks, "codex")
	if len(candidates) != 5 {
		t.Fatalf("len = %d, want 5: %#v", len(candidates), candidates)
	}
	for index, ref := range []string{"CORE-8", "CORE-4", "CORE-9", "CORE-2", "CORE-10"} {
		if candidates[index].Ref != ref {
			t.Fatalf("candidate %d = %s, want %s", index, candidates[index].Ref, ref)
		}
	}
}
