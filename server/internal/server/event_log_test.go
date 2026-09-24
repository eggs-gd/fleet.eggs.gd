package server

import (
	"path/filepath"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/audit"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

func TestRuntimePatchTaskWritesStatusUpdateDetails(t *testing.T) {
	root := t.TempDir()
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
	writeTestFile(t, filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-a.md"), testTaskMarkdown("CORE-2", "Second", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root})
	_, err := app.PatchTask(tasklifecycle.TaskPatch{
		Path:   "Work/core-eggs-gd/tasks/2026-07-31-a.md",
		Status: "doing",
	})
	if err != nil {
		t.Fatal(err)
	}

	events := readTestEvents(t, root)
	var patchEvent audit.Event
	for _, event := range events {
		if event.Type == "task_patched" {
			patchEvent = event
			break
		}
	}
	if patchEvent.Type == "" {
		t.Fatalf("task_patched event missing in %#v", events)
	}
	if patchEvent.Details["stage"] != "status-update" {
		t.Fatalf("stage = %#v, want status-update", patchEvent.Details["stage"])
	}
	if patchEvent.Details["index_rebuilt"] != true {
		t.Fatalf("index_rebuilt = %#v, want true", patchEvent.Details["index_rebuilt"])
	}
	previous := patchEvent.Details["previous"].(map[string]any)
	current := patchEvent.Details["current"].(map[string]any)
	if previous["status"] != "todo" || current["status"] != "doing" {
		t.Fatalf("status details = previous %#v current %#v", previous, current)
	}
}

func readTestEvents(t *testing.T, root string) []audit.Event {
	t.Helper()
	events, err := audit.ReadEvents(root)
	if err != nil {
		t.Fatal(err)
	}
	return events
}
