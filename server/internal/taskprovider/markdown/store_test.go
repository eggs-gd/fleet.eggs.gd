package markdown

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

func TestPatchTaskFileAppendsReviewComment(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-test-task.md")
	writeTestFile(t, taskPath, `---
id: work-2026-07-31-test-task
ref: CORE-999
title: "Test task"
type: feature
status: needs_review
project: core-eggs-gd
repositories:
  - core.eggs.gd
assignee: codex
created_at: 2026-07-31T10:00:00+03:00
updated_at: 2026-07-31T10:00:00+03:00
launch:
---

# Test task

## Request

Do the thing.
`)

	task, err := PatchTaskFile(root, tasklifecycle.TaskPatch{
		Path:          taskPath,
		Status:        "todo",
		CommentAuthor: "owner",
		Comment:       "Fix the returned task before trying again.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != "todo" {
		t.Fatalf("status = %q, want todo", task.Status)
	}
	if len(task.Comments) != 1 || task.Comments[0].Author != "owner" {
		t.Fatalf("comments = %#v", task.Comments)
	}
}

func TestLoadTaskFileDefaultsMissingPriorityToFive(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-test-task.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-996", "Default priority", "todo"))

	task, err := LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if task.Priority != 5 {
		t.Fatalf("priority = %d, want 5", task.Priority)
	}
}

func TestLoadTaskFileParsesDependsOn(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-09-08-depends.md")
	writeTestFile(t, taskPath, `---
id: work-depends
ref: CORE-147
title: "Depends"
type: bug
status: todo
project: core-eggs-gd
repositories:
  - core.eggs.gd
depends_on:
  - CORE-144
  - work-docs-id
assignee: claude
created_at: 2026-09-08T10:00:00+03:00
updated_at: 2026-09-08T10:00:00+03:00
---

# Depends
`)

	task, err := LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(task.DependsOn) != 2 || task.DependsOn[0] != "CORE-144" || task.DependsOn[1] != "work-docs-id" {
		t.Fatalf("depends_on = %#v, want [CORE-144 work-docs-id]", task.DependsOn)
	}
}

func TestLoadTaskFileDerivesBlockedReasonFromLatestWorkerReviewComment(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-01-blocked.md")
	writeTestFile(t, taskPath, `---
id: work-blocked
ref: CORE-999
title: "Blocked"
type: test
status: blocked
project: core-eggs-gd
repositories:
  - core.eggs.gd
assignee: claude
created_at: 2026-08-01T10:00:00+03:00
updated_at: 2026-08-01T10:00:00+03:00
---

# Blocked

## Review Comments

- 2026-08-01T10:00:00+03:00 — claude: First blocker.
- 2026-08-01T10:01:00+03:00 — claude: Alex, please choose the launch mode before I continue.
- 2026-08-01T10:02:00+03:00 — alex: Ask me what to do next.
`)

	task, err := LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if task.BlockedReason != "Alex, please choose the launch mode before I continue." {
		t.Fatalf("blocked reason = %q", task.BlockedReason)
	}
}

func TestProviderRejectsStaleExpectedUpdatedAt(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-test-task.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-990", "Conflict task", "todo"))

	_, err := New(root, Hooks{}).Mutate(tasklifecycle.TaskPatch{
		Path:              taskPath,
		Status:            "doing",
		ExpectedUpdatedAt: "2026-07-31T09:00:00+03:00",
	})
	if !errors.Is(err, tasklifecycle.ErrTaskConflict) {
		t.Fatalf("err = %v, want ErrTaskConflict", err)
	}
}

func TestProviderTransitionRejectsUnconfiguredTransition(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-test-task.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-987", "Rejected transition task", "todo"))

	_, err := New(root, Hooks{}).Transition(taskPath, "done")
	if !errors.Is(err, tasklifecycle.ErrTaskTransitionRejected) {
		t.Fatalf("err = %v, want ErrTaskTransitionRejected", err)
	}
}

func TestProviderAllowsBlockedToNeedsReview(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-blocked-recovery.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-126", "Blocked recovery", "blocked"))

	provider := New(root, Hooks{})
	updated, err := provider.Mutate(tasklifecycle.TaskPatch{
		Path:          taskPath,
		Status:        "needs_review",
		Comment:       "Alex verified artifacts; infrastructure blocker cleared.",
		CommentAuthor: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "needs_review" {
		t.Fatalf("status = %q, want needs_review", updated.Status)
	}
	if len(updated.Comments) == 0 {
		t.Fatal("expected recovery comment on task")
	}
}

func TestClaimTaskForLaunchMovesTodoTaskToDoing(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-test-task.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-998", "Claim me", "todo"))

	task, err := ClaimTaskForLaunch(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != "doing" {
		t.Fatalf("status = %q, want doing", task.Status)
	}
}

func TestClaimTaskForLaunchRejectsNonPickupTask(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-test-task.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-997", "Already claimed", "doing"))

	_, err := ClaimTaskForLaunch(root, taskPath)
	if !errors.Is(err, tasklifecycle.ErrTaskAlreadyClaimed) {
		t.Fatalf("err = %v, want ErrTaskAlreadyClaimed", err)
	}
}

func TestProviderListExcludesReadmeAndSorts(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-low.md"), `---
id: work-low
ref: CORE-1
title: "Low priority"
type: feature
status: todo
priority: 5
project: core-eggs-gd
repositories:
  - core.eggs.gd
assignee: claude
created_at: 2026-07-31T10:00:00+03:00
updated_at: 2026-07-31T10:00:00+03:00
launch:
---

# Low priority
`)
	writeTestFile(t, filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-high.md"), `---
id: work-high
ref: CORE-2
title: "High priority"
type: feature
status: todo
priority: 1
project: core-eggs-gd
repositories:
  - core.eggs.gd
assignee: claude
created_at: 2026-07-31T10:00:00+03:00
updated_at: 2026-07-31T10:00:00+03:00
launch:
---

# High priority
`)
	writeTestFile(t, filepath.Join(root, "Work", "core-eggs-gd", "tasks", "README.md"), "# not a task\n")

	provider := New(root, Hooks{})
	if provider.Type() != "markdown" {
		t.Fatalf("Type() = %q", provider.Type())
	}
	tasks, err := provider.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 {
		t.Fatalf("List() len = %d, want 2", len(tasks))
	}
	if tasks[0].Ref != "CORE-2" {
		t.Fatalf("first = %q, want CORE-2", tasks[0].Ref)
	}
}

func TestPatchTaskFileMovesTaskToAnotherProject(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Work", "core-eggs-gd", "PROJECT.md"), "---\nid: core-eggs-gd\ntitle: Core\n---\n")
	writeTestFile(t, filepath.Join(root, "Work", "_life", "PROJECT.md"), "---\nid: _life\ntitle: Life\n---\n")
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-05-move-me.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-137", "Move me", "todo"))

	task, err := PatchTaskFile(root, tasklifecycle.TaskPatch{
		Path:       taskPath,
		Project:    "_life",
		Repository: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.Project != "_life" {
		t.Fatalf("project = %q, want _life", task.Project)
	}
	if task.WorkspaceID != "_life" {
		t.Fatalf("workspace = %q, want _life", task.WorkspaceID)
	}
	wantRel := "Work/_life/tasks/2026-08-05-move-me.md"
	if task.RelativePath != wantRel {
		t.Fatalf("relative_path = %q, want %q", task.RelativePath, wantRel)
	}
	if _, err := os.Stat(taskPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old path still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, wantRel)); err != nil {
		t.Fatalf("new path missing: %v", err)
	}
}

func TestPatchTaskFileUpdatesNestedProjectWithoutMovingFolder(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Work", "eggs-gd-prod", "PROJECT.md"), "---\nid: eggs-gd-prod\ntitle: eggs\n---\n")
	taskPath := filepath.Join(root, "Work", "eggs-gd-prod", "tasks", "2026-08-05-nested.md")
	writeTestFile(t, taskPath, `---
schema_version: 1
id: work-nested
ref: CORE-138
title: "Nested"
type: feature
status: todo
project: eggs-gd-prod
repositories:
  - eGGs.gd.prod/other
assignee: codex
created_at: 2026-07-31T10:00:00+03:00
updated_at: 2026-07-31T10:00:00+03:00
launch:
---

# Nested
`)

	task, err := PatchTaskFile(root, tasklifecycle.TaskPatch{
		Path:       taskPath,
		Project:    "eggs-gd-prod/career-wizard",
		Repository: "eGGs.gd.prod/career-wizard",
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.Project != "eggs-gd-prod" {
		t.Fatalf("project = %q, want eggs-gd-prod", task.Project)
	}
	if task.ProjectID != "eggs-gd-prod/career-wizard" {
		t.Fatalf("project_id = %q, want eggs-gd-prod/career-wizard", task.ProjectID)
	}
	if task.RelativePath != "Work/eggs-gd-prod/tasks/2026-08-05-nested.md" {
		t.Fatalf("relative_path = %q", task.RelativePath)
	}
	if len(task.Repositories) != 1 || task.Repositories[0] != "eGGs.gd.prod/career-wizard" {
		t.Fatalf("repositories = %#v", task.Repositories)
	}
}

func TestPatchTaskFileUpdatesDependsOn(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-09-08-depends-edit.md")
	writeTestFile(t, taskPath, `---
id: work-depends-edit
ref: CORE-148
title: "Depends edit"
type: bug
status: todo
project: core-eggs-gd
repositories:
  - core.eggs.gd
depends_on: []
assignee: cursor
created_at: 2026-09-08T10:00:00+03:00
updated_at: 2026-09-08T10:00:00+03:00
---

# Depends edit
`)

	deps := []string{"CORE-144", "CORE-145"}
	task, err := PatchTaskFile(root, tasklifecycle.TaskPatch{
		Path:      taskPath,
		DependsOn: &deps,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(task.DependsOn) != 2 || task.DependsOn[0] != "CORE-144" || task.DependsOn[1] != "CORE-145" {
		t.Fatalf("depends_on = %#v, want [CORE-144 CORE-145]", task.DependsOn)
	}

	cleared := []string{}
	task, err = PatchTaskFile(root, tasklifecycle.TaskPatch{
		Path:      taskPath,
		DependsOn: &cleared,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(task.DependsOn) != 0 {
		t.Fatalf("depends_on after clear = %#v, want empty", task.DependsOn)
	}
}

func TestSetFrontmatterRejectsNestedKeys(t *testing.T) {
	_, err := setFrontmatterValue("launch:\n  agent: claude\n", "launch.agent", "codex")
	if err == nil {
		t.Fatal("expected nested frontmatter edit error")
	}
}

func TestSetFrontmatterRepositoriesReplacesList(t *testing.T) {
	frontmatter := "project: eggs-gd-prod\nrepositories:\n  - old/repo\nassignee: codex\n"
	updated, err := setFrontmatterRepositories(frontmatter, []string{"eGGs.gd.prod/career-wizard"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(updated, "repositories:\n  - eGGs.gd.prod/career-wizard\n") {
		t.Fatalf("updated = %q", updated)
	}
	if strings.Contains(updated, "old/repo") {
		t.Fatalf("old repository remained: %q", updated)
	}
	cleared, err := setFrontmatterRepositories(updated, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cleared, "repositories: []\n") {
		t.Fatalf("cleared = %q", cleared)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testTaskMarkdown(ref, title, status string) string {
	return `---
schema_version: 1
id: work-test
ref: ` + ref + `
title: "` + title + `"
type: feature
status: ` + status + `
project: core-eggs-gd
repositories:
  - core.eggs.gd
assignee: codex
created_at: 2026-07-31T10:00:00+03:00
updated_at: 2026-07-31T10:00:00+03:00
launch:
---

# ` + title + `
`
}
