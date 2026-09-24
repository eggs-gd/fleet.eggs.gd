package markdown

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

func TestObserveFileChangeEmitsNormalizedTaskEvent(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-03-observe.md")
	writeTestFile(t, taskPath, `---
schema_version: 1
id: work-2026-08-03-observe
ref: CORE-103
title: "Observe markdown change"
type: maintenance
status: todo
priority: 1
project: core-eggs-gd
repositories:
  - core.eggs.gd
assignee: cursor
created_at: 2026-08-03T23:10:00+03:00
updated_at: 2026-08-03T23:10:00+03:00
---

# Observe markdown change
`)

	provider := New(root, Hooks{})
	event, task, err := provider.ObserveFileChange(taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if task.Ref != "CORE-103" {
		t.Fatalf("reloaded ref = %q, want CORE-103", task.Ref)
	}
	wantID := "Work/core-eggs-gd/tasks/2026-08-03-observe.md"
	if event.TaskID != wantID {
		t.Fatalf("TaskID = %q, want %q", event.TaskID, wantID)
	}
	if event.Source != taskflow.TaskEventSourceMarkdownFS {
		t.Fatalf("Source = %q, want %q", event.Source, taskflow.TaskEventSourceMarkdownFS)
	}
	if event.Before != nil {
		t.Fatalf("Before = %#v, want nil on first observe", event.Before)
	}
	if event.After.Ref != "CORE-103" || event.After.Status != taskflow.StatusTodo {
		t.Fatalf("After = %#v", event.After)
	}
	if event.Kind != taskflow.TaskEventUpserted {
		t.Fatalf("Kind = %q, want %q", event.Kind, taskflow.TaskEventUpserted)
	}

	// TaskService.Get must accept the event TaskID (opaque locator).
	got, err := provider.Flow().Get(context.Background(), event.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Ref != "CORE-103" || got.Status != taskflow.StatusTodo {
		t.Fatalf("TaskService load via event = %+v", got)
	}
}

func TestTaskEventForPopulatesBeforeAfterSource(t *testing.T) {
	before := tasklifecycle.Task{
		RelativePath: "Work/core-eggs-gd/tasks/x.md",
		Ref:          "CORE-111",
		Status:       "todo",
		Assignee:     "cursor",
	}
	after := before
	after.Status = "doing"
	event := TaskEventFor(&before, after, taskflow.TaskEventSourceMarkdownFS)
	if event.Source != taskflow.TaskEventSourceMarkdownFS {
		t.Fatalf("Source = %q", event.Source)
	}
	if event.Before == nil || event.Before.Status != taskflow.StatusTodo {
		t.Fatalf("Before = %#v", event.Before)
	}
	if event.After.Status != taskflow.StatusDoing {
		t.Fatalf("After.Status = %q", event.After.Status)
	}
	if !event.StatusChanged() || event.Kind != taskflow.TaskEventStatusChanged {
		t.Fatalf("expected status_changed, got Kind=%q StatusChanged=%v", event.Kind, event.StatusChanged())
	}
}

func TestClassifyTaskEventKindStatusChanged(t *testing.T) {
	current := tasklifecycle.Task{Status: "todo"}
	if got := ClassifyTaskEventKind("todo", true, current); got != taskflow.TaskEventUpserted {
		t.Fatalf("same status: kind = %q, want upserted", got)
	}
	current.Status = "doing"
	if got := ClassifyTaskEventKind("todo", true, current); got != taskflow.TaskEventStatusChanged {
		t.Fatalf("status change: kind = %q, want status_changed", got)
	}
	if got := ClassifyTaskEventKind("", false, current); got != taskflow.TaskEventUpserted {
		t.Fatalf("new task: kind = %q, want upserted", got)
	}
}

func TestObserveTaskFileChangeAppliesActiveExecutionGuard(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-03-guard.md")
	writeTestFile(t, taskPath, `---
schema_version: 1
id: work-2026-08-03-guard
ref: CORE-130
title: "Observe markdown guard"
type: maintenance
status: needs_review
priority: 1
project: core-eggs-gd
repositories:
  - core.eggs.gd
assignee: cursor
created_at: 2026-08-03T23:10:00+03:00
updated_at: 2026-08-03T23:10:00+03:00
---

# Observe markdown guard
`)

	provider := New(root, Hooks{})
	before := tasklifecycle.Task{
		RelativePath: "Work/core-eggs-gd/tasks/2026-08-03-guard.md",
		Ref:          "CORE-130",
		Status:       "doing",
		Assignee:     "cursor",
	}
	reverted := before
	reverted.UpdatedAt = "2026-08-03T23:12:00+03:00"

	change, err := provider.ObserveTaskFileChange(taskPath, ObserveTaskHooks{
		Before: func(locator string) (tasklifecycle.Task, bool) {
			if locator != before.RelativePath {
				t.Fatalf("locator = %q, want %q", locator, before.RelativePath)
			}
			return before, true
		},
		ActiveSessionForTask: func(task tasklifecycle.Task) (ActiveSession, bool) {
			return ActiveSession{ClaimID: "claim-1", ExecutionStatus: "running", Status: "running"}, true
		},
		RevertActiveExecution: func(locator string, session ActiveSession) (tasklifecycle.Task, error) {
			if locator != before.RelativePath {
				t.Fatalf("revert locator = %q, want %q", locator, before.RelativePath)
			}
			if session.ClaimID != "claim-1" {
				t.Fatalf("claim_id = %q", session.ClaimID)
			}
			return reverted, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !change.GuardApplied {
		t.Fatal("expected active execution guard to apply")
	}
	if change.ObservedStatus != "needs_review" {
		t.Fatalf("ObservedStatus = %q, want needs_review", change.ObservedStatus)
	}
	if change.Before == nil || change.Before.Status != "doing" {
		t.Fatalf("Before = %#v, want doing", change.Before)
	}
	if change.After.Status != "doing" {
		t.Fatalf("After.Status = %q, want doing", change.After.Status)
	}
	if change.Event.After.Status != taskflow.StatusDoing {
		t.Fatalf("Event.After.Status = %q, want doing", change.Event.After.Status)
	}
}
