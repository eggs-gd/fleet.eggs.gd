package execution

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLaunchableDetectionRequiresPickupStatusAssigneeProfileAndRepository(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root+"/Fleet/claude.md", "# Claude\n")

	task := &Task{
		Ref:          "CORE-5",
		Status:       "todo",
		Assignee:     "claude",
		Repositories: []string{"core.eggs.gd"},
	}

	validator := &Validator{Root: root}
	launcher := &AgentLauncher{Planner: stubPlanner{plans: map[string]Plan{
		"claude": {
			Agent:           "claude",
			Backend:         BackendBackgroundRemote,
			Command:         []string{"/tmp/claude", "--bg", "--remote-control", "prompt"},
			WorkingDir:      "/tmp/repo",
			VisibilityClass: string(VisibilityAppVisible),
			LiveReady:       true,
		},
	}}}

	validated, err := validator.Decorate(task)
	if err != nil {
		t.Fatal(err)
	}
	launched, err := launcher.Decorate(validated)
	if err != nil {
		t.Fatal(err)
	}

	if !launched.LaunchEvaluation.Launchable {
		t.Fatalf("Launchable = false, failed gates = %#v", launched.LaunchEvaluation.FailedGates)
	}
	for _, step := range []string{"status:pickup", "assignee", "depends_on", "fleet_profile", "repository", "command"} {
		if !slices.Contains(launched.LaunchEvaluation.PassedGates, step) {
			t.Fatalf("missing passed step %q in %#v", step, launched.LaunchEvaluation.PassedGates)
		}
	}
}

func TestLaunchableDetectionBlocksBacklogTasks(t *testing.T) {
	task := &Task{
		Ref:          "CORE-5",
		Status:       "backlog",
		Assignee:     "codex",
		Repositories: []string{"core.eggs.gd"},
	}

	validator := &Validator{Root: t.TempDir()}
	validated, err := validator.Decorate(task)
	if err != nil {
		t.Fatal(err)
	}
	if validated.LaunchEvaluation.Launchable {
		t.Fatal("backlog task became launchable")
	}
	if len(validated.LaunchEvaluation.FailedGates) != 1 || validated.LaunchEvaluation.FailedGates[0] != `status:pickup: status is "backlog", expected "needs_rework" or "todo"` {
		t.Fatalf("failed gates = %#v", validated.LaunchEvaluation.FailedGates)
	}
}

func TestLaunchableDetectionBlocksMultiRepositoryTasks(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root+"/Fleet/claude.md", "# Claude\n")

	task := &Task{
		Ref:          "CORE-46",
		Status:       "todo",
		Assignee:     "claude",
		Repositories: []string{"core.eggs.gd", "another.repo"},
	}

	validator := &Validator{Root: root}
	validated, err := validator.Decorate(task)
	if err != nil {
		t.Fatal(err)
	}
	if validated.LaunchEvaluation.Launchable {
		t.Fatal("multi-repository task became launchable")
	}
	if len(validated.LaunchEvaluation.FailedGates) != 1 || validated.LaunchEvaluation.FailedGates[0] != "repository: automatic launch requires exactly one repository" {
		t.Fatalf("failed gates = %#v", validated.LaunchEvaluation.FailedGates)
	}
}
