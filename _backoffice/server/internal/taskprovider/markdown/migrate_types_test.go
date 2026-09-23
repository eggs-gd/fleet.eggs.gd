package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRebuildWorkIndexMigratesLegacyTodoType(t *testing.T) {
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
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-05-legacy.md")
	content := strings.Replace(
		testTaskMarkdown("CORE-139", "Legacy type", "todo"),
		"type: feature\n",
		"type: todo\n",
		1,
	)
	writeTestFile(t, taskPath, content)

	keepPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-05-bug.md")
	writeTestFile(t, keepPath, strings.Replace(
		testTaskMarkdown("CORE-140", "Keep bug type", "backlog"),
		"type: feature\n",
		"type: bug\n",
		1,
	))

	if err := RebuildWorkIndex(root); err != nil {
		t.Fatal(err)
	}

	migrated, err := os.ReadFile(taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(migrated), "type: feature\n") {
		t.Fatalf("expected type migrated to feature\n%s", migrated)
	}
	if strings.Contains(string(migrated), "type: todo\n") {
		t.Fatalf("legacy type todo still present\n%s", migrated)
	}
	// Status todo must stay a status.
	if !strings.Contains(string(migrated), "status: todo\n") {
		t.Fatalf("status todo was incorrectly changed\n%s", migrated)
	}

	kept, err := os.ReadFile(keepPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(kept), "type: bug\n") {
		t.Fatalf("unrelated type was changed\n%s", kept)
	}

	index, err := os.ReadFile(filepath.Join(root, "Work", "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), "| `CORE-139` | [Legacy type]") {
		t.Fatalf("index missing migrated task\n%s", index)
	}
	if !strings.Contains(string(index), "| `feature` | `codex` |") {
		t.Fatalf("index missing feature type column for migrated task\n%s", index)
	}
}
