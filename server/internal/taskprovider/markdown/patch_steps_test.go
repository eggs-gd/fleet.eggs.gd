package markdown

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

func patchErr(t *testing.T, patch tasklifecycle.TaskPatch) error {
	t.Helper()
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-05-t.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-1", "T", "todo"))
	patch.Path = taskPath
	_, err := PatchTaskFile(root, patch)
	return err
}

func TestPatchTaskFileRejectsBadFields(t *testing.T) {
	zero, nine := 0, 9
	cases := []struct {
		name    string
		patch   tasklifecycle.TaskPatch
		message string
	}{
		{"unknown status", tasklifecycle.TaskPatch{Status: "sideways"}, "unknown task status"},
		{"rejected transition", tasklifecycle.TaskPatch{Status: "done"}, "todo"},
		{"low priority", tasklifecycle.TaskPatch{Priority: &zero}, "unknown task priority 0"},
		{"high priority", tasklifecycle.TaskPatch{Priority: &nine}, "unknown task priority 9"},
		{"project path escape", tasklifecycle.TaskPatch{Project: "../outside"}, "invalid"},
		{"project with colon", tasklifecycle.TaskPatch{Project: "a:b"}, "invalid"},
		{"project that does not exist", tasklifecycle.TaskPatch{Project: "missing-project"}, "does not exist"},
	}
	for _, c := range cases {
		if err := patchErr(t, c.patch); err == nil || !strings.Contains(err.Error(), c.message) {
			t.Errorf("%s: err = %v, want %q", c.name, err, c.message)
		}
	}
}

func TestPatchTaskFileMilestoneNeedsAProjectCardEntry(t *testing.T) {
	missing := "no-such-milestone"
	if err := patchErr(t, tasklifecycle.TaskPatch{Milestone: &missing}); err == nil || !strings.Contains(err.Error(), "milestone") {
		t.Fatalf("err = %v", err)
	}
}

func TestPatchTaskFileSetsAssigneePriorityAndDependsOn(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-05-t.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-1", "T", "todo"))
	prio := 2
	deps := []string{"CORE-7", "CORE-7"}
	task, err := PatchTaskFile(root, tasklifecycle.TaskPatch{Path: taskPath, Assignee: "claude", Priority: &prio, DependsOn: &deps})
	if err != nil {
		t.Fatal(err)
	}
	if task.Assignee != "claude" || task.Priority != 2 || len(task.DependsOn) != 1 || task.DependsOn[0] != "CORE-7" {
		t.Fatalf("task = %+v", task)
	}
}

func TestNoUnlockIsSafeToCall(t *testing.T) {
	noUnlock()
}
