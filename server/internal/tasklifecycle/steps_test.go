package tasklifecycle

import (
	"reflect"
	"strings"
	"testing"
)

func TestTaskPatchChangeSummary(t *testing.T) {
	ptr := func(s string) *string { return &s }
	deps := []string{"B-1"}
	before := Task{Status: "todo", Assignee: "ann", Priority: 3, ProjectID: "p1", RelativePath: "Work/p1/tasks/a.md", Milestone: "m1", DependsOn: []string{"A-1"}}
	after := Task{Status: "doing", Assignee: "bob", Priority: 2, ProjectID: "p2", RelativePath: "Work/p2/tasks/a.md", Milestone: "m2", DependsOn: []string{"B-1"}}
	patch := TaskPatch{Milestone: ptr("m2"), DependsOn: &deps, Body: ptr("x"), Comment: " note "}
	got := TaskPatchChangeSummary(before, after, patch)
	want := []string{
		"status=todo->doing", "assignee=ann->bob", "priority=3->2", "project=p1->p2",
		"path=Work/p1/tasks/a.md->Work/p2/tasks/a.md", "milestone=m1->m2", "depends_on=A-1->B-1",
		"body_changed=true", "comment_added=true",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %v\nwant %v", got, want)
	}
	if got := TaskPatchChangeSummary(before, before, TaskPatch{}); len(got) != 0 {
		t.Fatalf("an unchanged task has no changes, got %v", got)
	}
	// Milestone and dependencies only count when the patch names them.
	if got := TaskPatchChangeSummary(before, after, TaskPatch{}); contains(got, "milestone=m1->m2") || contains(got, "depends_on=A-1->B-1") {
		t.Fatalf("got %v", got)
	}
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func TestNormalizeTaskCreateRequest(t *testing.T) {
	five := 5
	ok := TaskCreateRequest{Title: " T ", Request: " do it ", Project: " p ", Assignee: " Ann ", Type: "bug"}
	got, err := normalizeTaskCreateRequest(ok)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "T" || got.Status != "backlog" || got.Assignee != "ann" || got.Type != "bug" || *got.Priority != five {
		t.Fatalf("got %+v", got)
	}
	if !strings.Contains(got.AssignmentReason, "Assigned to ann") {
		t.Fatalf("reason = %q", got.AssignmentReason)
	}

	unassigned, err := normalizeTaskCreateRequest(TaskCreateRequest{Title: "T", Request: "r", Project: "p"})
	if err != nil || unassigned.Assignee != Unassigned || !strings.Contains(unassigned.AssignmentReason, "left unassigned") {
		t.Fatalf("got %+v err %v", unassigned, err)
	}
	custom, _ := normalizeTaskCreateRequest(TaskCreateRequest{Title: "T", Request: "r", Project: "p", AssignmentReason: "because"})
	if custom.AssignmentReason != "because" {
		t.Fatalf("reason = %q", custom.AssignmentReason)
	}

	zero, seven := 0, 7
	cases := []struct {
		name    string
		mutate  func(*TaskCreateRequest)
		message string
	}{
		{"no title", func(r *TaskCreateRequest) { r.Title = " " }, "title is required"},
		{"no request", func(r *TaskCreateRequest) { r.Request = "" }, "request/description is required"},
		{"no project", func(r *TaskCreateRequest) { r.Project = "" }, "project is required"},
		{"dotdot project", func(r *TaskCreateRequest) { r.Project = "../x" }, "project id"},
		{"colon project", func(r *TaskCreateRequest) { r.Project = "a:b" }, "project id"},
		{"bad status", func(r *TaskCreateRequest) { r.Status = "nope" }, "unknown task status"},
		{"bad type", func(r *TaskCreateRequest) { r.Type = "nope" }, "unknown task type"},
		{"bad assignee", func(r *TaskCreateRequest) { r.Assignee = "A B!" }, "not a valid name"},
		{"low priority", func(r *TaskCreateRequest) { r.Priority = &zero }, "unknown task priority 0"},
		{"high priority", func(r *TaskCreateRequest) { r.Priority = &seven }, "unknown task priority 7"},
	}
	for _, c := range cases {
		req := TaskCreateRequest{Title: "T", Request: "r", Project: "p"}
		c.mutate(&req)
		if _, err := normalizeTaskCreateRequest(req); err == nil || !strings.Contains(err.Error(), c.message) {
			t.Errorf("%s: err = %v, want %q", c.name, err, c.message)
		}
	}
}

func TestJSONObjectCandidates(t *testing.T) {
	text := `intro {"a": "}"} middle {"b": {"c": 1}} and "{not}" {"d": "q\"}"} tail {unclosed`
	got := jsonObjectCandidates(text)
	want := []string{`{"a": "}"}`, `{"b": {"c": 1}}`, `{"d": "q\"}"}`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	if got := jsonObjectCandidates("} stray { "); len(got) != 0 {
		t.Fatalf("got %q", got)
	}
}
