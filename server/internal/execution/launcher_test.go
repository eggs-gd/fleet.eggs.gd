package execution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

const (
	codexAppServerBackend = BackendCodexAppServer
	cursorVisibleBackend  = BackendCursorVisible
)

type stubPlanner struct {
	plans map[string]Plan
	err   error
}

func (planner stubPlanner) Plan(name string, task TaskContext) (Plan, error) {
	if planner.err != nil {
		return Plan{}, planner.err
	}
	plan, ok := planner.plans[name]
	if !ok {
		return Plan{}, ErrUnknownAgent
	}
	return plan, nil
}

func installFakeLauncherExecutable(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return path
}

func launcherCommandHasArg(command []string, arg string) bool {
	for _, candidate := range command {
		if candidate == arg {
			return true
		}
	}
	return false
}

func TestBuildClaudeCommandUsesBackgroundRemoteControlByDefault(t *testing.T) {
	binary := installFakeLauncherExecutable(t, "claude")
	task := &Task{
		Ref:      "CORE-9",
		Assignee: "claude",
		LaunchEvaluation: tasklifecycle.LaunchEvaluation{
			Agent:      "claude",
			WorkingDir: "/tmp/repo",
		},
	}
	launcher := &AgentLauncher{Planner: stubPlanner{plans: map[string]Plan{
		"claude": {
			Agent:           "claude",
			Backend:         BackendBackgroundRemote,
			Command:         []string{binary, "--bg", "--remote-control", "--name", "CORE-9", "--permission-mode", "acceptEdits", "prompt"},
			WorkingDir:      "/tmp/repo",
			VisibilityClass: string(VisibilityAppVisible),
			LiveReady:       true,
		},
	}}}

	launched, err := launcher.Decorate(task)
	if err != nil {
		t.Fatal(err)
	}
	command := launched.LaunchEvaluation.Command
	if !launched.LaunchEvaluation.Launchable {
		t.Skipf("claude binary not resolvable in this environment: %#v", launched.LaunchEvaluation.FailedGates)
	}
	if launched.LaunchEvaluation.Backend != "background-remote" {
		t.Fatalf("backend = %q, want background-remote", launched.LaunchEvaluation.Backend)
	}
	if len(command) != 8 {
		t.Fatalf("command = %#v, want background remote-control claude command", command)
	}
	joined := strings.Join(command, " ")
	for _, want := range []string{"--bg", "--remote-control", "--name", "--permission-mode acceptEdits"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("command missing %q: %#v", want, command)
		}
	}
	if launcherCommandHasArg(command, "-p") || launcherCommandHasArg(command, "--output-format") || launcherCommandHasArg(command, "bypassPermissions") {
		t.Fatalf("command must not use Claude headless print flags: %#v", command)
	}
}

func TestBuildCodexCommandUsesAppServerMode(t *testing.T) {
	installFakeLauncherExecutable(t, "codex")
	task := &Task{
		Ref:      "CORE-57",
		Assignee: "codex",
		LaunchEvaluation: tasklifecycle.LaunchEvaluation{
			Agent:      "codex",
			WorkingDir: "/tmp/repo",
		},
	}
	launcher := &AgentLauncher{Planner: stubPlanner{plans: map[string]Plan{
		"codex": {
			Agent:           "codex",
			Backend:         BackendCodexAppServer,
			Command:         []string{"/tmp/codex", "app-server", "--stdio"},
			WorkingDir:      "/tmp/repo",
			VisibilityClass: string(VisibilityCoreVisible),
			LiveReady:       true,
			LiveNotes:       "core_visible",
		},
	}}}

	launched, err := launcher.Decorate(task)
	if err != nil {
		t.Fatal(err)
	}
	if !launched.LaunchEvaluation.Launchable {
		t.Fatalf("Codex app-server should be daemon-launchable by default (CORE-130): %#v", launched.LaunchEvaluation)
	}
	if launched.LaunchEvaluation.VisibilityMode != string(VisibilityCoreVisible) {
		t.Fatalf("visibility = %q, want %q", launched.LaunchEvaluation.VisibilityMode, VisibilityCoreVisible)
	}
	if launched.LaunchEvaluation.Backend != codexAppServerBackend {
		t.Fatalf("backend = %q, want %q", launched.LaunchEvaluation.Backend, codexAppServerBackend)
	}
	command := launched.LaunchEvaluation.Command
	if len(command) != 3 || command[1] != "app-server" || command[2] != "--stdio" {
		t.Fatalf("command = %#v, want codex app-server command", command)
	}
}

func TestBuildCodexCommandLegacyAllowCoreVisibleStillLaunchable(t *testing.T) {
	installFakeLauncherExecutable(t, "codex")
	task := &Task{
		Ref:      "CORE-70",
		Assignee: "codex",
		Launch: tasklifecycle.Launch{
			Mode: LaunchModeAllowCoreVisible,
		},
		LaunchEvaluation: tasklifecycle.LaunchEvaluation{
			Agent:      "codex",
			WorkingDir: "/tmp/repo",
		},
	}
	launcher := &AgentLauncher{Planner: stubPlanner{plans: map[string]Plan{
		"codex": {
			Agent:           "codex",
			Backend:         BackendCodexAppServer,
			Command:         []string{"/tmp/codex", "app-server", "--stdio"},
			WorkingDir:      "/tmp/repo",
			VisibilityClass: string(VisibilityCoreVisible),
			LiveReady:       true,
			LiveNotes:       "legacy override accepted",
		},
	}}}

	launched, err := launcher.Decorate(task)
	if err != nil {
		t.Fatal(err)
	}
	if !launched.LaunchEvaluation.Launchable {
		t.Fatalf("legacy allow_core_visible must remain launchable: %#v", launched.LaunchEvaluation)
	}
	if launched.LaunchEvaluation.VisibilityMode != string(VisibilityCoreVisible) {
		t.Fatalf("must remain core_visible, got %q", launched.LaunchEvaluation.VisibilityMode)
	}
	command := launched.LaunchEvaluation.Command
	if len(command) != 3 || command[1] != "app-server" || command[2] != "--stdio" {
		t.Fatalf("command = %#v, want codex app-server command", command)
	}
}

func TestHeadlessBackendRemainsBlockedFromDaemonAutoLaunch(t *testing.T) {
	task := &Task{
		Ref:      "CORE-130",
		Assignee: "codex",
		LaunchEvaluation: tasklifecycle.LaunchEvaluation{
			Agent:      "codex",
			WorkingDir: "/tmp/repo",
		},
	}
	launcher := &AgentLauncher{Planner: stubPlanner{plans: map[string]Plan{
		"codex": {
			Agent:           "codex",
			Backend:         "process",
			Command:         []string{"/tmp/codex", "exec", "prompt"},
			WorkingDir:      "/tmp/repo",
			VisibilityClass: string(VisibilityHeadless),
			LiveReady:       true,
			LiveNotes:       "headless must stay blocked",
		},
	}}}

	launched, err := launcher.Decorate(task)
	if err != nil {
		t.Fatal(err)
	}
	if launched.LaunchEvaluation.Launchable {
		t.Fatalf("headless must not be daemon-launchable: %#v", launched.LaunchEvaluation)
	}
	if len(launched.LaunchEvaluation.FailedGates) != 1 || !strings.HasPrefix(launched.LaunchEvaluation.FailedGates[0], "agent_visibility:") {
		t.Fatalf("failed gates = %#v, want agent_visibility", launched.LaunchEvaluation.FailedGates)
	}
}

func TestCursorLaunchUsesVisibleCliMode(t *testing.T) {
	installFakeLauncherExecutable(t, "cursor-agent")
	task := &Task{
		Ref:      "CORE-56",
		Assignee: "cursor",
		LaunchEvaluation: tasklifecycle.LaunchEvaluation{
			Agent:      "cursor",
			WorkingDir: "/tmp/repo",
		},
	}
	launcher := &AgentLauncher{Planner: stubPlanner{plans: map[string]Plan{
		"cursor": {
			Agent:           "cursor",
			Backend:         BackendCursorVisible,
			Command:         []string{"/tmp/cursor-agent", "--trust", "--workspace", "/tmp/repo", "prompt"},
			WorkingDir:      "/tmp/repo",
			VisibilityClass: string(VisibilityCLIVisible),
			LiveReady:       true,
		},
	}}}

	launched, err := launcher.Decorate(task)
	if err != nil {
		t.Fatal(err)
	}
	if !launched.LaunchEvaluation.Launchable {
		t.Fatalf("Cursor should be launchable through cursor-visible backend: %#v", launched.LaunchEvaluation)
	}
	if launched.LaunchEvaluation.VisibilityMode != string(VisibilityCLIVisible) {
		t.Fatalf("visibility = %q, want %q", launched.LaunchEvaluation.VisibilityMode, VisibilityCLIVisible)
	}
	if launched.LaunchEvaluation.Backend != cursorVisibleBackend {
		t.Fatalf("backend = %q, want %q", launched.LaunchEvaluation.Backend, cursorVisibleBackend)
	}
	command := launched.LaunchEvaluation.Command
	if len(command) == 0 {
		t.Fatal("cursor launch command is empty")
	}
	if launcherCommandHasArg(command, "--print") || launcherCommandHasArg(command, "-p") || launcherCommandHasArg(command, "--output-format") {
		t.Fatalf("Cursor launch must not use print/headless flags: %#v", command)
	}
	if !launcherCommandHasArg(command, "--trust") || !launcherCommandHasArg(command, "--workspace") {
		t.Fatalf("Cursor launch missing visible CLI workspace flags: %#v", command)
	}
}

func TestClaudeLaunchExposesAppVisibleClass(t *testing.T) {
	installFakeLauncherExecutable(t, "claude")
	task := &Task{
		Ref:      "CORE-70",
		Assignee: "claude",
		LaunchEvaluation: tasklifecycle.LaunchEvaluation{
			Agent:      "claude",
			WorkingDir: "/tmp/repo",
		},
	}
	launcher := &AgentLauncher{Planner: stubPlanner{plans: map[string]Plan{
		"claude": {
			Agent:           "claude",
			Backend:         BackendBackgroundRemote,
			Command:         []string{"/tmp/claude", "--bg", "--remote-control", "--name", "CORE-70", "--permission-mode", "acceptEdits", "prompt"},
			WorkingDir:      "/tmp/repo",
			VisibilityClass: string(VisibilityAppVisible),
			LiveReady:       true,
		},
	}}}

	launched, err := launcher.Decorate(task)
	if err != nil {
		t.Fatal(err)
	}
	if !launched.LaunchEvaluation.Launchable {
		t.Skipf("claude binary not resolvable in this environment: %#v", launched.LaunchEvaluation.FailedGates)
	}
	if launched.LaunchEvaluation.VisibilityMode != string(VisibilityAppVisible) {
		t.Fatalf("visibility = %q, want %q", launched.LaunchEvaluation.VisibilityMode, VisibilityAppVisible)
	}
}

func TestMissingExecutableLiveNotesUseAgentExecutableGate(t *testing.T) {
	task := &Task{
		Ref:      "CORE-150",
		Assignee: "codex",
		LaunchEvaluation: tasklifecycle.LaunchEvaluation{
			Agent:      "codex",
			WorkingDir: "/tmp/repo",
		},
	}
	launcher := &AgentLauncher{Planner: stubPlanner{plans: map[string]Plan{
		"codex": {
			Agent:           "codex",
			Backend:         "unsupported",
			WorkingDir:      "/tmp/repo",
			VisibilityClass: string(VisibilityUnknown),
			LiveReady:       false,
			LiveNotes:       "Codex CLI is not available on the daemon PATH. Install the standalone Codex CLI or configure the Codex binary path, then retry this task. PATH=/usr/bin",
		},
	}}}

	launched, err := launcher.Decorate(task)
	if err != nil {
		t.Fatal(err)
	}
	if launched.LaunchEvaluation.Launchable {
		t.Fatal("expected missing executable plan to be not launchable")
	}
	if len(launched.LaunchEvaluation.FailedGates) == 0 || !strings.HasPrefix(launched.LaunchEvaluation.FailedGates[0], "agent_executable:") {
		t.Fatalf("failed gates = %#v, want agent_executable", launched.LaunchEvaluation.FailedGates)
	}
	decision := ClassifyLaunchLog(*launched)
	if decision.Category != LaunchLogNotReady {
		t.Fatalf("category = %q, want not_ready", decision.Category)
	}
}
