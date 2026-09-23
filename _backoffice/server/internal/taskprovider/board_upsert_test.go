package taskprovider

import (
	"path/filepath"
	"testing"

	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
)

func TestUpsertTaskCollapsesAbsoluteAndRelativePathAliases(t *testing.T) {
	root := t.TempDir()
	store := NewBoard(root)
	rel := filepath.ToSlash(filepath.Join("Work", "core-eggs-gd", "tasks", "core-87.md"))
	abs := filepath.Join(root, filepath.FromSlash(rel))

	store.UpsertTask(tasklifecycle.Task{
		Ref:          "CORE-87",
		ID:           "work-core-87",
		Status:       "backlog",
		Path:         abs,
		RelativePath: "",
		Title:        "Stale abs key",
	})
	store.UpsertTask(tasklifecycle.Task{
		Ref:          "CORE-87",
		ID:           "work-core-87",
		Status:       "todo",
		Path:         abs,
		RelativePath: rel,
		Title:        "Canonical relative key",
	})

	state := store.Snapshot()
	if len(state.Tasks) != 1 {
		t.Fatalf("tasks = %d, want 1 unique identity after upsert alias collapse", len(state.Tasks))
	}
	got := state.Tasks[0]
	if got.Status != "todo" || got.Title != "Canonical relative key" {
		t.Fatalf("task = %#v, want todo canonical row", got)
	}
	if _, ok := store.TaskByPath(abs); !ok {
		t.Fatal("TaskByPath(abs) miss, want lookup by absolute path")
	}
	if _, ok := store.TaskByPath(rel); !ok {
		t.Fatal("TaskByPath(rel) miss, want lookup by relative path")
	}
}

func TestUpsertTaskCollapsesIDKeyedAliasWhenRelativePathArrives(t *testing.T) {
	root := t.TempDir()
	store := NewBoard(root)
	rel := filepath.ToSlash(filepath.Join("Work", "core-eggs-gd", "tasks", "core-87.md"))

	store.UpsertTask(tasklifecycle.Task{
		Ref:    "CORE-87",
		ID:     "work-core-87",
		Status: "backlog",
		Title:  "ID keyed backlog",
	})
	store.UpsertTask(tasklifecycle.Task{
		Ref:          "CORE-87",
		ID:           "work-core-87",
		Status:       "todo",
		RelativePath: rel,
		Title:        "Path keyed todo",
	})

	state := store.Snapshot()
	if len(state.Tasks) != 1 {
		t.Fatalf("tasks = %d, want 1 after ID/path alias collapse", len(state.Tasks))
	}
	if state.Tasks[0].Status != "todo" {
		t.Fatalf("status = %q, want todo", state.Tasks[0].Status)
	}
}
