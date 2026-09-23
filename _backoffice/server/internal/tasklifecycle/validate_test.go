package tasklifecycle

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestEvaluateLaunchRequiresPickupStatusAssigneeProfileAndRepository(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Fleet", "claude.md"), "# Claude\n")

	task := &Task{
		Ref:          "CORE-5",
		Status:       "todo",
		Assignee:     "claude",
		Repositories: []string{"core.eggs.gd"},
	}

	EvaluateLaunch(task, root, nil)

	if !task.LaunchEvaluation.Launchable {
		t.Fatalf("Launchable = false, failed gates = %#v", task.LaunchEvaluation.FailedGates)
	}
	for _, step := range []string{"status:pickup", "assignee", "depends_on", "fleet_profile", "repository"} {
		if !slices.Contains(task.LaunchEvaluation.PassedGates, step) {
			t.Fatalf("missing passed step %q in %#v", step, task.LaunchEvaluation.PassedGates)
		}
	}
}

func TestEvaluateLaunchBlocksBacklogTasks(t *testing.T) {
	task := &Task{
		Ref:          "CORE-5",
		Status:       "backlog",
		Assignee:     "codex",
		Repositories: []string{"core.eggs.gd"},
	}

	EvaluateLaunch(task, t.TempDir(), nil)

	if task.LaunchEvaluation.Launchable {
		t.Fatal("backlog task became launchable")
	}
	if len(task.LaunchEvaluation.FailedGates) != 1 || task.LaunchEvaluation.FailedGates[0] != `status:pickup: status is "backlog", expected "needs_rework" or "todo"` {
		t.Fatalf("failed gates = %#v", task.LaunchEvaluation.FailedGates)
	}
}

func TestEvaluateLaunchBlocksMultiRepositoryTasks(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Fleet", "claude.md"), "# Claude\n")

	task := &Task{
		Ref:          "CORE-46",
		Status:       "todo",
		Assignee:     "claude",
		Repositories: []string{"core.eggs.gd", "another.repo"},
	}

	EvaluateLaunch(task, root, nil)

	if task.LaunchEvaluation.Launchable {
		t.Fatal("multi-repository task became launchable")
	}
	if len(task.LaunchEvaluation.FailedGates) != 1 || task.LaunchEvaluation.FailedGates[0] != "repository: automatic launch requires exactly one repository" {
		t.Fatalf("failed gates = %#v", task.LaunchEvaluation.FailedGates)
	}
}

func TestEvaluateLaunchBlocksMissingFleetProfile(t *testing.T) {
	task := &Task{
		Ref:          "CORE-47",
		Status:       "todo",
		Assignee:     "nobody",
		Repositories: []string{"core.eggs.gd"},
	}

	EvaluateLaunch(task, t.TempDir(), nil)

	if task.LaunchEvaluation.Launchable {
		t.Fatal("task with no Fleet profile became launchable")
	}
	if len(task.LaunchEvaluation.FailedGates) != 1 || task.LaunchEvaluation.FailedGates[0] != `fleet_profile: Fleet profile not found for agent "nobody"` {
		t.Fatalf("failed gates = %#v", task.LaunchEvaluation.FailedGates)
	}
}

// CORE-148 regression: CORE-147-style implementation must not launch while
// documentation prerequisite CORE-144 is still unfinished.
func TestEvaluateLaunchBlocksUnresolvedDependsOn(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Fleet", "claude.md"), "# Claude\n")

	prerequisite := Task{
		Ref:    "CORE-144",
		ID:     "work-2026-09-08-i-ching-pwa-deeplink-forwarding-docs",
		Status: "needs_review",
	}
	dependent := &Task{
		Ref:          "CORE-147",
		Status:       "todo",
		Assignee:     "claude",
		Repositories: []string{"core.eggs.gd"},
		DependsOn:    []string{"CORE-144"},
	}

	EvaluateLaunch(dependent, root, []Task{prerequisite, *dependent})

	if dependent.LaunchEvaluation.Launchable {
		t.Fatal("dependent launched before prerequisite reached done")
	}
	if dependent.Status != "todo" {
		t.Fatalf("status = %q, want todo (dependency wait must not flip operator blocked)", dependent.Status)
	}
	if len(dependent.LaunchEvaluation.FailedGates) != 1 {
		t.Fatalf("failed gates = %#v", dependent.LaunchEvaluation.FailedGates)
	}
	got := dependent.LaunchEvaluation.FailedGates[0]
	if !strings.HasPrefix(got, "depends_on:") || !strings.Contains(got, "CORE-144") || !strings.Contains(got, `want "done"`) {
		t.Fatalf("failed gate = %q, want depends_on waiting on CORE-144 for done", got)
	}
	if dependent.LaunchEvaluation.Outcome != "not_launchable" {
		t.Fatalf("outcome = %q, want not_launchable", dependent.LaunchEvaluation.Outcome)
	}
}

func TestEvaluateLaunchAllowsDependsOnAfterPrerequisiteDone(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Fleet", "claude.md"), "# Claude\n")

	prerequisite := Task{
		Ref:    "CORE-144",
		Status: "done",
	}
	dependent := &Task{
		Ref:          "CORE-147",
		Status:       "todo",
		Assignee:     "claude",
		Repositories: []string{"core.eggs.gd"},
		DependsOn:    []string{"CORE-144"},
	}

	EvaluateLaunch(dependent, root, []Task{prerequisite, *dependent})

	if !dependent.LaunchEvaluation.Launchable {
		t.Fatalf("Launchable = false after prerequisite done: %#v", dependent.LaunchEvaluation.FailedGates)
	}
	if !slices.Contains(dependent.LaunchEvaluation.PassedGates, "depends_on") {
		t.Fatalf("missing depends_on pass in %#v", dependent.LaunchEvaluation.PassedGates)
	}
}

func TestEvaluateLaunchResolvesDependsOnByTaskID(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Fleet", "codex.md"), "# Codex\n")

	prerequisite := Task{
		Ref:    "CORE-144",
		ID:     "work-docs-deeplink",
		Status: "done",
	}
	dependent := &Task{
		Ref:          "CORE-147",
		Status:       "needs_rework",
		Assignee:     "codex",
		Repositories: []string{"core.eggs.gd"},
		DependsOn:    []string{"work-docs-deeplink"},
	}

	EvaluateLaunch(dependent, root, []Task{prerequisite})

	if !dependent.LaunchEvaluation.Launchable {
		t.Fatalf("id-resolved dependency should pass: %#v", dependent.LaunchEvaluation.FailedGates)
	}
}

func TestEvaluateLaunchBlocksMissingDependsOnRef(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Fleet", "claude.md"), "# Claude\n")

	dependent := &Task{
		Ref:          "CORE-147",
		Status:       "todo",
		Assignee:     "claude",
		Repositories: []string{"core.eggs.gd"},
		DependsOn:    []string{"CORE-144"},
	}

	EvaluateLaunch(dependent, root, nil)

	if dependent.LaunchEvaluation.Launchable {
		t.Fatal("missing dependency became launchable")
	}
	got := dependent.LaunchEvaluation.FailedGates[0]
	if !strings.Contains(got, "CORE-144 not found") {
		t.Fatalf("failed gate = %q, want not found", got)
	}
}

func TestTasksDependingOnFindsPickupDependents(t *testing.T) {
	completed := Task{Ref: "CORE-144", Status: "done"}
	known := []Task{
		{Ref: "CORE-147", Status: "todo", DependsOn: []string{"CORE-144"}},
		{Ref: "CORE-149", Status: "backlog", DependsOn: []string{"CORE-144"}},
		{Ref: "CORE-150", Status: "todo", DependsOn: []string{"CORE-999"}},
	}
	got := TasksDependingOn(known, completed)
	if len(got) != 1 || got[0].Ref != "CORE-147" {
		t.Fatalf("TasksDependingOn = %#v, want CORE-147 only", got)
	}
}
