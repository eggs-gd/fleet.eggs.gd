package taskflow

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestApplyStatusPatch(t *testing.T) {
	cases := []struct {
		name     string
		task     Task
		patch    PatchInput
		wantErr  bool
		changed  bool
		wantTask Status
		blocked  string
	}{
		{"no status", Task{Status: StatusTodo}, PatchInput{}, false, false, StatusTodo, ""},
		{"same status", Task{Status: StatusTodo}, PatchInput{Status: StatusTodo}, false, false, StatusTodo, ""},
		{"allowed move", Task{Status: StatusTodo}, PatchInput{Status: StatusDoing}, false, true, StatusDoing, ""},
		{"rejected move", Task{Status: StatusDone}, PatchInput{Status: StatusDoing}, true, false, StatusDone, ""},
		{"override", Task{Status: StatusDone}, PatchInput{Status: StatusDoing, AllowStatusOverride: true}, false, true, StatusDoing, ""},
		{"blocked with reason", Task{Status: StatusDoing}, PatchInput{Status: StatusBlocked, Reason: " stuck "}, false, true, StatusBlocked, "stuck"},
		{"reason only counts for blocked", Task{Status: StatusTodo}, PatchInput{Status: StatusDoing, Reason: "x"}, false, true, StatusDoing, ""},
	}
	for _, c := range cases {
		task := c.task
		changed, err := applyStatusPatch(&task, c.patch)
		if (err != nil) != c.wantErr || (err != nil && !errors.Is(err, ErrInvalidTransition)) {
			t.Errorf("%s: err = %v", c.name, err)
			continue
		}
		if changed != c.changed || task.Status != c.wantTask || task.BlockedReason != c.blocked {
			t.Errorf("%s: changed %v status %v reason %q", c.name, changed, task.Status, task.BlockedReason)
		}
	}
}

func TestApplyPatchFields(t *testing.T) {
	prio := 2
	milestone := " m1 "
	deps := []string{"B-2", " A-1 ", "B-2"}
	body := "new body"
	task := Task{Status: StatusTodo, Body: "old", Repositories: []string{"old"}}
	changed, err := applyPatchFields(&task, PatchInput{
		Priority: &prio, Assignee: " ann ", Project: " p1 ", Repository: " r1 ",
		Milestone: &milestone, DependsOn: &deps, Body: &body,
	})
	if err != nil || !changed {
		t.Fatalf("changed %v err %v", changed, err)
	}
	if task.Priority != 2 || task.Assignee != "ann" || task.Project != "p1" || task.Milestone != "m1" || task.Body != "new body" {
		t.Fatalf("task = %+v", task)
	}
	if len(task.Repositories) != 1 || task.Repositories[0] != "r1" {
		t.Fatalf("repositories = %v", task.Repositories)
	}
	if len(task.DependsOn) != 2 {
		t.Fatalf("depends_on = %v", task.DependsOn)
	}

	moved := Task{Project: "a", Repositories: []string{"x"}, Body: "keep out of the write"}
	changed, _ = applyPatchFields(&moved, PatchInput{Project: "b"})
	if !changed || moved.Repositories != nil {
		t.Fatalf("a project move without a repository clears repositories, got %+v", moved)
	}
	if moved.Body != "" {
		t.Fatal("a patch without a body omits the body rewrite")
	}
	same := Task{}
	if changed, _ := applyPatchFields(&same, PatchInput{}); changed {
		t.Fatal("an empty patch changes nothing")
	}
	if _, err := applyPatchFields(&Task{Status: StatusDone}, PatchInput{Status: StatusDoing}); err == nil {
		t.Fatal("a rejected transition must stop the patch")
	}
}

func TestCurrentLocator(t *testing.T) {
	ctx := context.Background()
	p := newMemoryProvider()
	p.tasks["t1"] = Task{ID: "t1", Ref: "R-1", Locator: "loc/t1"}
	p.tasks["t2"] = Task{ID: "t2", Ref: "R-2"}
	p.tasks["moved-new"] = Task{ID: "moved-new", Ref: "R-9", Locator: "moved-new"}

	if got := currentLocator(ctx, p, "t1", "R-1"); got != "loc/t1" {
		t.Fatalf("a task with a locator uses it, got %q", got)
	}
	if got := currentLocator(ctx, p, "t2", "R-2"); got != "t2" {
		t.Fatalf("no locator falls back to the id, got %q", got)
	}
	if got := currentLocator(ctx, p, "moved-old", "R-9"); got != "moved-new" {
		t.Fatalf("the ref finds a moved task, got %q", got)
	}
	if got := currentLocator(ctx, p, "gone", ""); got != "gone" {
		t.Fatalf("no ref, got %q", got)
	}
	if got := currentLocator(ctx, p, "gone", "R-404"); got != "gone" {
		t.Fatalf("an unknown ref, got %q", got)
	}
}

func TestFormatExecutionComment(t *testing.T) {
	full := formatExecutionComment(ExecutionResult{
		Outcome: "completed", Agent: " codex ", ExecutionID: "c-1", Summary: " did it ", Question: "why?", Error: "oops",
		Artifacts: []string{"a.txt", "  ", " b.txt "}, Tests: []string{"go test"},
	})
	for _, want := range []string{
		"Execution outcome: `completed`", "\nAgent: `codex`", "\nExecution ID: `c-1`", "\n\ndid it",
		"\n\nQuestion: why?", "\n\nError: oops", "\n\nArtifacts:\n- `a.txt`\n- `b.txt`", "\n\nTests:\n- `go test`",
	} {
		if !strings.Contains(full, want) {
			t.Errorf("comment lacks %q:\n%s", want, full)
		}
	}
	if strings.Contains(full, "- ``") {
		t.Error("blank items are skipped")
	}
	if got := formatExecutionComment(ExecutionResult{Outcome: "failed"}); got != "Execution outcome: `failed`" {
		t.Fatalf("minimal = %q", got)
	}
}
