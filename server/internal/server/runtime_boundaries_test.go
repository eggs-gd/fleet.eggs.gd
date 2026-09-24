package server

import (
	"path/filepath"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/execution"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider/markdown"
)

// Provider-neutral execution from TaskEvent uses execution.Entry directly.
func TestProviderNeutralExecutionFromTaskEvent(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-neutral.md")
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
id: work-2026-08-04-neutral
ref: CORE-107
title: "Provider-neutral execution"
type: maintenance
status: todo
priority: 1
project: core-eggs-gd
repositories:
  - core.eggs.gd
assignee: cursor
created_at: 2026-08-04T00:00:00+03:00
updated_at: 2026-08-04T00:00:00+03:00
---

# Provider-neutral execution
`)

	provider := markdown.New(root, markdown.Hooks{})
	service := taskflow.NewService(provider.Flow())
	defer service.Close()

	store := taskprovider.NewBoard(root)
	task, err := provider.Load(taskPath)
	if err != nil {
		t.Fatal(err)
	}
	store.UpsertTask(task)

	entry := &execution.Entry{Service: service, Lookup: store.TaskByPath}
	loaded, err := entry.Decorate(taskflow.NewTaskEvent(
		task.RelativePath,
		nil,
		taskflow.Task{
			Locator:  task.RelativePath,
			Ref:      "CORE-107",
			Status:   taskflow.StatusTodo,
			Assignee: "cursor",
		},
		taskflow.TaskEventSourceMarkdownFS,
	))
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.Ref != "CORE-107" {
		t.Fatalf("loaded = %#v", loaded)
	}
	if loaded.Status != "todo" || loaded.Assignee != "cursor" {
		t.Fatalf("status=%q assignee=%q", loaded.Status, loaded.Assignee)
	}
}

func TestActiveExecutionGuardWritesThroughTaskService(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-guard.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-107G", "Guard via service", "doing"))

	app := Compose(ComposeConfig{CoreRoot: root})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	app.Store.UpsertTask(task)
	app.Exec.UpsertSession(execution.RuntimeSession{
		ClaimID:         "core-107g-active",
		TaskRef:         task.Ref,
		TaskID:          task.ID,
		TaskPath:        task.RelativePath,
		ProjectID:       task.ProjectID,
		Repository:      "core.eggs.gd",
		Status:          "running",
		ExecutionStatus: "running",
	})

	// Simulate an out-of-band card edit to needs_review while session is live.
	if _, err := app.TaskStore.Mutate(tasklifecycle.TaskPatch{
		Path:                task.RelativePath,
		Status:              "needs_review",
		CommentAuthor:       "codex",
		Comment:             `{"outcome":"completed","summary":"early finalize"}`,
		AllowStatusOverride: true,
	}); err != nil {
		t.Fatal(err)
	}

	observed := observeTaskFileChange(t, app, taskPath)
	if !observed.GuardApplied || observed.After.Status != "doing" {
		t.Fatalf("guarded observe = %#v, want status doing", observed)
	}
	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "doing" {
		t.Fatalf("persisted status = %q, want doing", loaded.Status)
	}
}
