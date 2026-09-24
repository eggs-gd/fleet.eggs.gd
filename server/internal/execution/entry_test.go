package execution_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/execution"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider/markdown"
	"github.com/eggs-gd/fleet.eggs.gd/lib/chain"
)

func TestEntrySkipsDeletedEvents(t *testing.T) {
	service := taskflow.NewService(markdown.New(t.TempDir(), markdown.Hooks{}).Flow())
	defer service.Close()
	entry := &execution.Entry{Service: service}

	before := &taskflow.Task{Locator: "Work/x/tasks/y.md", Status: taskflow.StatusTodo}
	_, err := entry.Decorate(taskflow.NewTaskEvent("Work/x/tasks/y.md", before, taskflow.Task{}, taskflow.TaskEventSourceMarkdownFS))
	if !errors.Is(err, chain.ErrSkippedItem) {
		t.Fatalf("err = %v, want ErrSkippedItem", err)
	}
}

func TestEntryLaunchEligibilityFromBeforeAfter(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-elig.md")
	writeProjectAndTask(t, root, taskPath, "CORE-112", "todo")

	provider := markdown.New(root, markdown.Hooks{})
	service := taskflow.NewService(provider.Flow())
	defer service.Close()
	task, err := provider.Load(taskPath)
	if err != nil {
		t.Fatal(err)
	}
	lookup := mapLookup(task)
	entry := &execution.Entry{Service: service, Lookup: lookup}

	todoAfter := taskflow.Task{
		Locator:  task.RelativePath,
		Ref:      "CORE-112",
		Status:   taskflow.StatusTodo,
		Assignee: "cursor",
	}
	reworkAfter := todoAfter
	reworkAfter.Status = taskflow.StatusNeedsRework
	doingAfter := todoAfter
	doingAfter.Status = taskflow.StatusDoing
	backlogAfter := todoAfter
	backlogAfter.Status = taskflow.StatusBacklog
	beforeTodo := &taskflow.Task{Locator: task.RelativePath, Status: taskflow.StatusTodo}
	beforeDoing := &taskflow.Task{Locator: task.RelativePath, Status: taskflow.StatusDoing}
	beforeBacklog := &taskflow.Task{Locator: task.RelativePath, Status: taskflow.StatusBacklog}

	cases := []struct {
		name    string
		event   taskflow.TaskEvent
		wantErr error
	}{
		{name: "markdown new todo", event: taskflow.NewTaskEvent(task.RelativePath, nil, todoAfter, taskflow.TaskEventSourceMarkdownFS)},
		{name: "plane new todo (same path)", event: taskflow.NewTaskEvent(task.RelativePath, nil, todoAfter, taskflow.TaskEventSourcePlanePoll)},
		{name: "webhook needs_rework from doing", event: taskflow.NewTaskEvent(task.RelativePath, beforeDoing, reworkAfter, taskflow.TaskEventSourcePlaneWebhook)},
		{name: "task_service backlog to todo", event: taskflow.NewTaskEvent(task.RelativePath, beforeBacklog, todoAfter, taskflow.TaskEventSourceTaskService)},
		{name: "todo to doing skipped", event: taskflow.NewTaskEvent(task.RelativePath, beforeTodo, doingAfter, taskflow.TaskEventSourceMarkdownFS), wantErr: chain.ErrSkippedItem},
		{name: "plane still doing skipped", event: taskflow.NewTaskEvent(task.RelativePath, beforeDoing, doingAfter, taskflow.TaskEventSourcePlanePoll), wantErr: chain.ErrSkippedItem},
		{name: "backlog not pickup", event: taskflow.NewTaskEvent(task.RelativePath, nil, backlogAfter, taskflow.TaskEventSourceMarkdownFS), wantErr: chain.ErrSkippedItem},
		{name: "deleted skipped", event: taskflow.NewTaskEvent(task.RelativePath, beforeTodo, taskflow.Task{}, taskflow.TaskEventSourcePlaneWebhook), wantErr: chain.ErrSkippedItem},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loaded, err := entry.Decorate(tc.event)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if loaded == nil || loaded.Ref != "CORE-112" {
				t.Fatalf("loaded = %#v", loaded)
			}
		})
	}
}

func TestEntryMarkdownAndPlaneIndistinguishable(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-neutral-src.md")
	writeProjectAndTask(t, root, taskPath, "CORE-112N", "todo")

	provider := markdown.New(root, markdown.Hooks{})
	service := taskflow.NewService(provider.Flow())
	defer service.Close()
	task, err := provider.Load(taskPath)
	if err != nil {
		t.Fatal(err)
	}
	entry := &execution.Entry{Service: service, Lookup: mapLookup(task)}

	after := taskflow.Task{
		Locator:  task.RelativePath,
		Ref:      "CORE-112N",
		Status:   taskflow.StatusTodo,
		Assignee: "cursor",
	}
	md, err := entry.Decorate(taskflow.NewTaskEvent(task.RelativePath, nil, after, taskflow.TaskEventSourceMarkdownFS))
	if err != nil {
		t.Fatal(err)
	}
	pl, err := entry.Decorate(taskflow.NewTaskEvent(task.RelativePath, nil, after, taskflow.TaskEventSourcePlanePoll))
	if err != nil {
		t.Fatal(err)
	}
	if md.Ref != pl.Ref || md.Status != pl.Status || md.Assignee != pl.Assignee {
		t.Fatalf("markdown loaded=%#v plane loaded=%#v; sources must be treated the same", md, pl)
	}
}

func mapLookup(task tasklifecycle.Task) execution.Lookup {
	return func(key string) (tasklifecycle.Task, bool) {
		if key == task.RelativePath || key == task.Path || key == task.ID {
			return task, true
		}
		return tasklifecycle.Task{}, false
	}
}

func writeProjectAndTask(t *testing.T, root, taskPath, ref, status string) {
	t.Helper()
	project := filepath.Join(root, "Work", "core-eggs-gd", "PROJECT.md")
	if err := os.MkdirAll(filepath.Dir(taskPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(project, []byte(`---
id: core-eggs-gd
title: Core
kind: standalone_repository
status: draft
review_status: draft
repositories:
  - core.eggs.gd
---

# Core
`), 0o644); err != nil {
		t.Fatal(err)
	}
	body := `---
schema_version: 1
id: work-` + stringsToID(ref) + `
ref: ` + ref + `
title: "Launch eligibility"
type: refactor
status: ` + status + `
priority: 1
project: core-eggs-gd
repositories:
  - core.eggs.gd
assignee: cursor
created_at: 2026-08-04T01:00:00+03:00
updated_at: 2026-08-04T01:00:00+03:00
---

# Launch eligibility
`
	if err := os.WriteFile(taskPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func stringsToID(ref string) string {
	out := make([]byte, 0, len(ref))
	for i := 0; i < len(ref); i++ {
		c := ref[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			out = append(out, c)
		} else {
			out = append(out, '-')
		}
	}
	return string(out)
}
