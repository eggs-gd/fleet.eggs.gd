package open

import (
	"strings"
	"testing"

	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/core.eggs.gd/internal/taskprovider"
)

func TestTaskProjectionChangeSummaryDescribesAgentEdits(t *testing.T) {
	before := taskprovider.Task{
		Status:   "doing",
		Priority: 1,
		Assignee: "claude",
		Body: strings.Join([]string{
			"# Smoke",
			"",
			"## Acceptance Criteria",
			"",
			"- [ ] Create artifact",
			"- [ ] Update task",
		}, "\n"),
	}
	after := before
	after.Status = "needs_review"
	after.UpdatedAt = "2026-08-01T14:16:30+03:00"
	after.Body = strings.Join([]string{
		"# Smoke",
		"",
		"## Acceptance Criteria",
		"",
		"- [x] Create artifact",
		"- [x] Update task",
		"",
		"## Deliverable",
		"",
		"Created `_docs/LIVE_AGENT_SMOKE_RESULT.md`.",
		"",
		"## Review Comments",
		"",
		"- Claude: ready",
	}, "\n")
	after.Comments = []tasklifecycle.Comment{{Author: "Claude", Text: "ready"}}

	changes := taskProjectionChangeSummary(before, after, true)
	for _, expected := range []string{
		"status=doing->needs_review",
		"updated_at=->2026-08-01T14:16:30+03:00",
		"comments=0->1",
		"acceptance=0/2->2/2",
		"section_added=Deliverable",
		"section_added=Review Comments",
	} {
		if !containsString(changes, expected) {
			t.Fatalf("changes missing %q: %#v", expected, changes)
		}
	}
}

func containsString(items []string, expected string) bool {
	for _, item := range items {
		if item == expected {
			return true
		}
	}
	return false
}
