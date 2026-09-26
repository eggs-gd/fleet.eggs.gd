package server

import (
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeCreateAndPatchTaskUseTaskService(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Work", "core-eggs-gd", "PROJECT.md"), `---
id: core-eggs-gd
tag: "CORE"
title: Core
kind: standalone_repository
status: draft
review_status: draft
repositories:
  - core.eggs.gd
---

# Core
`)
	writeTestFile(t, filepath.Join(root, "_registry", "counters.json"), `{
  "work_ref_prefix": "CORE",
  "next_work_ref": 104,
  "inbox_ref_prefix": "INBOX",
  "next_inbox_ref": 1,
  "life_ref_prefix": "LIFE",
  "next_life_ref": 1,
  "notes": []
}
`)

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: true})
	if app.TaskService() == nil {
		t.Fatal("TaskService must be wired on Runtime")
	}

	priority := 2
	created, err := app.CreateTask(tasklifecycle.TaskCreateRequest{
		Title:      "Manager route through TaskService",
		Request:    "Dashboard and manager writes must share the service boundary.",
		Project:    "core-eggs-gd",
		Repository: "core.eggs.gd",
		Status:     "backlog",
		Type:       "maintenance",
		Assignee:   "cursor",
		Priority:   &priority,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Ref != "CORE-104" {
		t.Fatalf("ref = %q, want CORE-104", created.Ref)
	}
	if created.Status != "backlog" {
		t.Fatalf("status = %q, want backlog", created.Status)
	}
	if _, err := os.Stat(created.Path); err != nil {
		t.Fatalf("created task file missing: %v", err)
	}

	moved, err := app.PatchTask(tasklifecycle.TaskPatch{
		Path:   created.RelativePath,
		Status: "todo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if moved.Status != "todo" {
		t.Fatalf("status after move = %q, want todo", moved.Status)
	}

	commented, err := app.PatchTask(tasklifecycle.TaskPatch{
		Path:          created.RelativePath,
		Comment:       "dashboard write via TaskService",
		CommentAuthor: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(commented.Comments) == 0 {
		t.Fatal("expected review comment from PatchTask")
	}
	foundAlex := false
	for _, c := range commented.Comments {
		if c.Author == "owner" && strings.Contains(c.Text, "dashboard write via TaskService") {
			foundAlex = true
		}
	}
	if !foundAlex {
		t.Fatalf("comments = %#v, want owner dashboard note", commented.Comments)
	}

	assigned, err := app.PatchTask(tasklifecycle.TaskPatch{
		Path:     created.RelativePath,
		Assignee: "claude",
	})
	if err != nil {
		t.Fatal(err)
	}
	if assigned.Assignee != "claude" {
		t.Fatalf("assignee = %q, want claude", assigned.Assignee)
	}
}
