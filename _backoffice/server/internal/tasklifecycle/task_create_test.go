package tasklifecycle

import (
	"strings"
	"testing"
)

func TestNormalizeTaskCreateRequestDefaultsAndValidates(t *testing.T) {
	got, err := NormalizeTaskCreateRequest(TaskCreateRequest{
		Title:   "Default backlog task",
		Request: "Keep it in backlog until prioritized.",
		Project: "core-eggs-gd",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "backlog" {
		t.Fatalf("status = %q, want backlog", got.Status)
	}
	if got.Type != "feature" {
		t.Fatalf("type = %q, want feature", got.Type)
	}

	legacy, err := NormalizeTaskCreateRequest(TaskCreateRequest{
		Title:   "Legacy type todo",
		Request: "todo was a status, not a type",
		Project: "core-eggs-gd",
		Type:    "todo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Type != "feature" {
		t.Fatalf("legacy type todo = %q, want feature", legacy.Type)
	}
	if got.Assignee != "unassigned" {
		t.Fatalf("assignee = %q, want unassigned", got.Assignee)
	}
	if got.Priority == nil || *got.Priority != 5 {
		t.Fatalf("priority = %v, want 5", got.Priority)
	}

	_, err = NormalizeTaskCreateRequest(TaskCreateRequest{
		Title:   "",
		Request: "no title",
		Project: "core-eggs-gd",
	})
	if err == nil || !strings.Contains(err.Error(), "title is required") {
		t.Fatalf("err = %v, want title required", err)
	}

	_, err = NormalizeTaskCreateRequest(TaskCreateRequest{
		Title:   "Bad status",
		Request: "nope",
		Project: "core-eggs-gd",
		Status:  "launching",
	})
	if err == nil || !strings.Contains(err.Error(), "unknown task status") {
		t.Fatalf("err = %v, want unknown status", err)
	}
}
