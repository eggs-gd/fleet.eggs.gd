package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRebuildWorkIndexFromSourceFiles(t *testing.T) {
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
	writeTestFile(t, filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-b.md"), testTaskMarkdown("CORE-10", "Tenth", "backlog"))
	writeTestFile(t, filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-c.md"), testTaskMarkdown("CORE-1", "First", "needs_review"))

	if err := RebuildWorkIndex(root); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(root, "Work", "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	index := string(data)
	for _, expected := range []string{
		"Generated read-only projection from canonical Core task and workspace files. Do not edit by hand.",
		"| `backlog` | 1 |",
		"| `todo` | 1 |",
		"| `needs_review` | 1 |",
		"| 5 | `CORE-1` | [First](core-eggs-gd/tasks/2026-07-31-c.md)",
		"| 5 | `CORE-2` | [Second](core-eggs-gd/tasks/2026-07-31-a.md)",
		"| `core-eggs-gd` | `core-eggs-gd` | `workspace` | 1 | 3 |",
		"| [Core](core-eggs-gd/PROJECT.md)",
	} {
		if !strings.Contains(index, expected) {
			t.Fatalf("index missing %q\n%s", expected, index)
		}
	}
	if strings.Contains(index, "Human-editable Work index") {
		t.Fatalf("index still claims to be human-editable\n%s", index)
	}
}

func TestApplyObservedChangeRebuildsWorkIndex(t *testing.T) {
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
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-a.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-2", "Second", "todo"))

	provider := New(root, Hooks{})
	observed, err := provider.ObserveTaskFileChange(taskPath, ObserveTaskHooks{})
	if err != nil {
		t.Fatal(err)
	}
	if err := provider.applyObservedChange(observed); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "Work", "INDEX.md")); err != nil {
		t.Fatalf("observed change did not rebuild Work/INDEX.md: %v", err)
	}
}
