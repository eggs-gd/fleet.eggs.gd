package server

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/eggs-gd/core.eggs.gd/internal/execution"
	"github.com/eggs-gd/core.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/core.eggs.gd/internal/taskprovider"
	"github.com/eggs-gd/core.eggs.gd/internal/taskprovider/markdown"
	"github.com/eggs-gd/core.eggs.gd/lib/chain"
)

func TestMarkdownFileChangeToTaskEventExecutionEntry(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-03-events.md")
	writeTestFile(t, filepath.Join(root, "Work", "core-eggs-gd", "PROJECT.md"), `---
id: core-eggs-gd
title: Core
kind: standalone_repository
status: draft
review_status: draft
repositories:
  - core.eggs.gd
---

# Core
`)
	writeTestFile(t, taskPath, `---
schema_version: 1
id: work-2026-08-03-events
ref: CORE-103
title: "Markdown file change to task event"
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

# Markdown file change to task event
`)

	store := taskprovider.NewBoard(root)
	provider := markdown.New(root, markdown.Hooks{})
	service := taskflow.NewService(provider.Flow())
	defer service.Close()

	observed, err := provider.ObserveTaskFileChange(taskPath, markdown.ObserveTaskHooks{
		Before: func(locator string) (tasklifecycle.Task, bool) {
			return store.TaskByPath(locator)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	store.UpsertTask(observed.After)
	event := observed.Event
	if event.TaskID == "" {
		t.Fatal("observed change missing TaskEvent")
	}
	if event.Source != taskflow.TaskEventSourceMarkdownFS {
		t.Fatalf("Source = %q, want %q", event.Source, taskflow.TaskEventSourceMarkdownFS)
	}
	if event.Before != nil {
		t.Fatalf("Before = %#v, want nil on first observation", event.Before)
	}
	if event.After.Ref != "CORE-103" || event.After.Status != taskflow.StatusTodo {
		t.Fatalf("After = %#v", event.After)
	}
	if event.Kind != taskflow.TaskEventUpserted {
		t.Fatalf("TaskEvent.Kind = %q, want upserted", event.Kind)
	}

	writeTestFile(t, taskPath, `---
schema_version: 1
id: work-2026-08-03-events
ref: CORE-103
title: "Markdown file change to task event"
type: maintenance
status: doing
priority: 1
project: core-eggs-gd
repositories:
  - core.eggs.gd
assignee: cursor
created_at: 2026-08-03T23:10:00+03:00
updated_at: 2026-08-03T23:15:00+03:00
---

# Markdown file change to task event
`)
	observed2, err := provider.ObserveTaskFileChange(taskPath, markdown.ObserveTaskHooks{
		Before: func(locator string) (tasklifecycle.Task, bool) {
			return store.TaskByPath(locator)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	event2 := observed2.Event
	if event2.Source != taskflow.TaskEventSourceMarkdownFS {
		t.Fatalf("second Source = %q", event2.Source)
	}
	if event2.Before == nil || event2.Before.Status != taskflow.StatusTodo {
		t.Fatalf("second Before = %#v, want todo", event2.Before)
	}
	if event2.After.Status != taskflow.StatusDoing {
		t.Fatalf("second After.Status = %q", event2.After.Status)
	}
	if !event2.StatusChanged() {
		t.Fatal("expected StatusChanged on second observation")
	}

	entry := &execution.Entry{Service: service, Lookup: store.TaskByPath}
	loaded, err := entry.Decorate(event)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.Ref != "CORE-103" {
		t.Fatalf("execution entry loaded = %#v", loaded)
	}
	if loaded.Status != "todo" || loaded.Assignee != "cursor" {
		t.Fatalf("loaded task fields = status=%q assignee=%q", loaded.Status, loaded.Assignee)
	}

	_, err = entry.Decorate(event2)
	if !errors.Is(err, chain.ErrSkippedItem) {
		t.Fatalf("doing After err = %v, want ErrSkippedItem", err)
	}
}

func TestExecutionContourPublishesExecutionResult(t *testing.T) {
	root := t.TempDir()
	app := Compose(ComposeConfig{CoreRoot: root, DryRun: true})
	if app.Exec == nil || app.Exec.ExecutionResults() == nil {
		t.Fatal("execution must expose ExecutionResults channel")
	}

	task := tasklifecycle.Task{RelativePath: "Work/x/tasks/y.md", Path: "Work/x/tasks/y.md", Ref: "CORE-112", Status: "doing"}
	result := taskflow.ExecutionResult{
		TaskID:  task.RelativePath,
		Outcome: taskflow.ExecutionCompleted,
		Summary: "publish probe",
	}
	app.Exec.PublishExecutionResult(result)

	select {
	case got := <-app.Exec.ExecutionResults():
		if got.TaskID != result.TaskID || got.Outcome != result.Outcome || got.Summary != result.Summary {
			t.Fatalf("published = %#v, want %#v", got, result)
		}
	default:
		t.Fatal("expected ExecutionResult on execution publish channel")
	}
}
