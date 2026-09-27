package providers

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func installFakeExecutable(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if name == "codex" {
		// Prefer PATH resolution in tests; host FLEET_CODEX_BINARY must not win.
		t.Setenv("FLEET_CODEX_BINARY", "")
	}
	return path
}

func commandHasArg(command []string, arg string) bool {
	for _, candidate := range command {
		if candidate == arg {
			return true
		}
	}
	return false
}

func TestBuildPromptIncludesRequiredCoreFrameworkRules(t *testing.T) {
	prompt := BuildPrompt(TaskContext{
		ID:           "work-test",
		Ref:          "CORE-26",
		Title:        "Launch agents",
		Status:       "todo",
		Type:         "feature",
		Priority:     3,
		ProjectID:    "core-eggs-gd",
		WorkspaceID:  "core-eggs-gd",
		Repository:   "core.eggs.gd",
		RelativePath: "Work/core-eggs-gd/tasks/2026-07-31-real-agent-process-launcher.md",
		WorkingDir:   "/tmp/core.eggs.gd",
		Body:         "Implement the thing.",
	})

	for _, expected := range []string{
		"_docs/TASK_LIFECYCLE.md",
		"_docs/OPERATING_MODEL.md",
		"Fleet/ROUTING.md",
		"Fleet/LAUNCH_POLICY.md",
		"compact JSON result payload",
		`"outcome": "<completed|failed|needs_input|needs_rework|blocked>"`,
		"Do not manually edit generated/service index files such as Work/INDEX.md",
		"Fleet daemon/finalizer refreshes derived files and generated indexes",
		"no-auto-commit, no-auto-push, and no-auto-PR",
		"Never finalize task status yourself",
		"Task card content:",
		"Implement the thing.",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt missing %q\n%s", expected, prompt)
		}
	}
}

// TestBuildPromptDoesNotInstructWorkersToMutateTaskCards is the CORE-97
// acceptance check: daemon-launched workers must report outcomes through the
// JSON result payload, not by editing their own (or any) canonical task
// card's status/frontmatter directly.
func TestBuildPromptDoesNotInstructWorkersToMutateTaskCards(t *testing.T) {
	prompt := BuildPrompt(TaskContext{
		ID:           "work-test",
		Ref:          "CORE-97",
		Title:        "Route worker outcomes",
		Status:       "todo",
		Type:         "architecture",
		Priority:     1,
		ProjectID:    "core-eggs-gd",
		WorkspaceID:  "core-eggs-gd",
		Repository:   "core.eggs.gd",
		RelativePath: "Work/core-eggs-gd/tasks/2026-08-03-route-worker-outcomes-through-manager.md",
		WorkingDir:   "/tmp/core.eggs.gd",
		Body:         "Route worker outcomes through the manager.",
	})

	for _, forbidden := range []string{
		"set the task card's status",
		"set the task",
	} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("prompt still instructs workers to self-mutate task card status: contains %q\n%s", forbidden, prompt)
		}
	}
	for _, expected := range []string{
		"Do not edit this (or any) canonical task card's status or frontmatter",
		"Fleet reads your reported outcome and applies",
		"never mutate",
		"canonical task card yourself",
		"Never finalize task status yourself",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt missing %q\n%s", expected, prompt)
		}
	}
}

// TestBuildPromptOutcomeExampleIsNotAValidWorkerOutcome guards against the
// false-positive risk documented on tasklifecycle.ParseWorkerResult: some
// providers (confirmed for Claude's `claude logs`) echo this launch prompt
// back into the same transcript Core scans for a worker's reported outcome.
// If the prompt's own example used a real recognized outcome value (e.g.
// "completed"), Core could mistake the echoed instructions for an actual
// worker report and finalize the task before any real work happened.
func TestBuildPromptOutcomeExampleIsNotAValidWorkerOutcome(t *testing.T) {
	prompt := BuildPrompt(TaskContext{Ref: "CORE-97", WorkingDir: "/tmp/core.eggs.gd"})

	start := strings.Index(prompt, `{"outcome"`)
	if start < 0 {
		t.Fatal("prompt missing the compact JSON result payload example")
	}
	end := strings.Index(prompt[start:], "\n")
	if end < 0 {
		end = len(prompt) - start
	}
	example := prompt[start : start+end]

	for _, outcome := range []string{"completed", "blocked", "needs_rework", "failed", "waiting_input", "needs_input"} {
		if strings.Contains(example, `"outcome": "`+outcome+`"`) || strings.Contains(example, `"outcome":"`+outcome+`"`) {
			t.Fatalf("prompt example uses a real recognized outcome %q, which an echoed transcript could mistake for a real report: %s", outcome, example)
		}
	}
}

func TestClaudePlanDefaultsToBackgroundRemoteControl(t *testing.T) {
	binary := installFakeExecutable(t, "claude")

	plan, err := Claude{}.Plan(TaskContext{
		Ref:        "CORE-26",
		WorkingDir: "/tmp/core.eggs.gd",
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Agent != "claude" {
		t.Fatalf("agent = %q, want claude", plan.Agent)
	}
	if plan.Backend != "background-remote" {
		t.Fatalf("backend = %q, want background-remote", plan.Backend)
	}
	if len(plan.Command) != 8 || plan.Command[0] != binary {
		t.Fatalf("command = %#v, want background remote-control claude command starting with %q", plan.Command, binary)
	}
	if plan.WorkingDir != "/tmp/core.eggs.gd" {
		t.Fatalf("working dir = %q", plan.WorkingDir)
	}
	wantFlags := []string{"--bg", "--remote-control", "--name", "--permission-mode", "acceptEdits"}
	for _, flag := range wantFlags {
		found := false
		for _, arg := range plan.Command {
			if arg == flag {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("command missing %q: %#v", flag, plan.Command)
		}
	}
	if !strings.Contains(plan.Command[len(plan.Command)-1], "CORE-26") {
		t.Fatalf("prompt command arg missing task ref: %#v", plan.Command)
	}
	if !plan.LiveReady {
		t.Fatalf("Claude plan should be live-ready: %s", plan.LiveNotes)
	}
	if plan.VisibilityClass != "app_visible" {
		t.Fatalf("visibility = %q, want app_visible", plan.VisibilityClass)
	}
	if commandHasArg(plan.Command, "-p") || commandHasArg(plan.Command, "--output-format") || commandHasArg(plan.Command, "bypassPermissions") {
		t.Fatalf("default Claude command must not use headless print flags: %#v", plan.Command)
	}
}

// CORE-53: Claude's human-input mode now dispatches a `claude --bg
// --remote-control` background agent instead of the earlier PTY/Terminal
// automation attempts (which were confirmed not daemon-controllable). Core
// does not inject stdin/keystrokes into this session; it supervises
// completion via `claude agents --json` polling and surfaces the printed
// Remote Control URL so a human can continue the conversation from the phone.
func TestClaudePlanUsesBackgroundRemoteControlForHumanInputMode(t *testing.T) {
	binary := installFakeExecutable(t, "claude")

	plan, err := Claude{}.Plan(TaskContext{
		Ref:        "CORE-53",
		ID:         "work-53",
		WorkingDir: "/tmp/core.eggs.gd",
		LaunchMode: "live-human-input-smoke",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.LiveReady {
		t.Fatalf("human-input Claude mode should be live-ready via background remote-control: %s", plan.LiveNotes)
	}
	if plan.Backend != "background-remote" {
		t.Fatalf("backend = %q, want background-remote", plan.Backend)
	}
	if plan.Command[0] != binary {
		t.Fatalf("command[0] = %q, want %q", plan.Command[0], binary)
	}
	wantFlags := []string{"--bg", "--remote-control", "--name", "--permission-mode", "acceptEdits"}
	for _, flag := range wantFlags {
		found := false
		for _, arg := range plan.Command {
			if arg == flag {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("command missing %q: %#v", flag, plan.Command)
		}
	}
	if !strings.Contains(plan.Command[len(plan.Command)-1], "CORE-53") {
		t.Fatalf("prompt command arg missing task ref: %#v", plan.Command)
	}
	if !strings.Contains(plan.LiveNotes, "claude agents --json") {
		t.Fatalf("live notes should explain daemon supervision without stdin injection: %s", plan.LiveNotes)
	}
}

func TestClaudeLaunchModeCannotSelectHeadlessPrint(t *testing.T) {
	binary := installFakeExecutable(t, "claude")

	plan, err := Claude{}.Plan(TaskContext{
		Ref:        "CORE-54",
		ID:         "work-54",
		WorkingDir: "/tmp/career-wizard",
		LaunchMode: "headless-print",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.LiveReady {
		t.Fatalf("background-remote Claude mode should be live-ready: %s", plan.LiveNotes)
	}
	if plan.Backend != "background-remote" {
		t.Fatalf("backend = %q, want background-remote", plan.Backend)
	}
	if plan.Command[0] != binary {
		t.Fatalf("command[0] = %q, want %q", plan.Command[0], binary)
	}
	if !strings.Contains(plan.Command[len(plan.Command)-1], "CORE-54") {
		t.Fatalf("prompt command arg missing task ref: %#v", plan.Command)
	}
	if commandHasArg(plan.Command, "-p") || commandHasArg(plan.Command, "--output-format") || commandHasArg(plan.Command, "bypassPermissions") {
		t.Fatalf("launch.mode must not select Claude headless print flags: %#v", plan.Command)
	}
}

func TestRegistryPlansConcreteAgents(t *testing.T) {
	installFakeExecutable(t, "claude")
	installFakeExecutable(t, "codex")
	installFakeExecutable(t, "cursor-agent")
	registry := NewRegistry()
	for _, name := range []string{"claude", "codex", "cursor"} {
		plan, err := registry.Plan(name, TaskContext{Ref: "CORE-26", WorkingDir: "/tmp/core.eggs.gd"})
		if err != nil {
			t.Fatalf("%s plan failed: %v", name, err)
		}
		if plan.Agent != name {
			t.Fatalf("%s plan agent = %q", name, plan.Agent)
		}
		if len(plan.Command) == 0 {
			t.Fatalf("%s command is empty", name)
		}
	}
}

func TestCodexPlanUsesAppServerCommand(t *testing.T) {
	binary := installFakeExecutable(t, "codex")

	plan, err := Codex{}.Plan(TaskContext{
		Ref:        "CORE-57",
		WorkingDir: "/tmp/core.eggs.gd",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.LiveReady {
		t.Fatalf("Codex plan must be live-ready by default: %s", plan.LiveNotes)
	}
	if plan.VisibilityClass != "fleet_visible" {
		t.Fatalf("visibility = %q, want fleet_visible", plan.VisibilityClass)
	}
	if plan.Backend != "codex-app-server" {
		t.Fatalf("backend = %q, want codex-app-server", plan.Backend)
	}
	if plan.Command[0] != binary {
		t.Fatalf("command[0] = %q, want %q", plan.Command[0], binary)
	}
	wantArgs := []string{"app-server", "--stdio"}
	for i, want := range wantArgs {
		if plan.Command[i+1] != want {
			t.Fatalf("command[%d] = %q, want %q in %#v", i+1, plan.Command[i+1], want, plan.Command)
		}
	}
	if !strings.Contains(plan.Prompt, "CORE-57") {
		t.Fatalf("prompt missing task ref: %s", plan.Prompt)
	}
	if !strings.Contains(plan.LiveNotes, "fleet_visible") || !strings.Contains(plan.LiveNotes, "Codex Remote") {
		t.Fatalf("live notes should document the fleet_visible Remote/control path: %s", plan.LiveNotes)
	}
	if !strings.Contains(plan.LiveNotes, "Resolved Codex binary: "+binary) {
		t.Fatalf("live notes should expose resolved Codex binary path: %s", plan.LiveNotes)
	}
	if strings.Contains(plan.LiveNotes, "Normal daemon auto-launch is disabled") {
		t.Fatalf("live notes must not claim auto-launch is disabled: %s", plan.LiveNotes)
	}
}

func TestCodexPlanRequiresStandaloneCLIOnPATH(t *testing.T) {
	// Isolate PATH so LookPath cannot find a host `codex`, including any
	// ChatGPT.app bundle that may exist on the machine. CORE-151 forbids that
	// fallback.
	t.Setenv("PATH", t.TempDir())
	t.Setenv("FLEET_CODEX_BINARY", "")

	plan, err := Codex{}.Plan(TaskContext{
		Ref:        "CORE-151",
		WorkingDir: "/tmp/core.eggs.gd",
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.LiveReady {
		t.Fatalf("missing standalone codex must not be live-ready: %#v", plan)
	}
	if len(plan.Command) != 0 {
		t.Fatalf("missing standalone codex must not produce a command (got %#v); ChatGPT.app fallback is forbidden", plan.Command)
	}
	if strings.Contains(plan.LiveNotes, "ChatGPT.app") || strings.Contains(strings.Join(plan.Command, " "), "ChatGPT.app") {
		t.Fatalf("must not mention or use ChatGPT.app bundled codex: %s / %#v", plan.LiveNotes, plan.Command)
	}
	for _, want := range []string{
		"Codex CLI is not available on the daemon PATH",
		"standalone Codex CLI",
		"configure the Codex binary path",
	} {
		if !strings.Contains(plan.LiveNotes, want) {
			t.Fatalf("live notes missing %q: %s", want, plan.LiveNotes)
		}
	}
}

func TestCodexPlanUsesConfiguredBinaryPath(t *testing.T) {
	dir := t.TempDir()
	configured := filepath.Join(dir, "codex")
	if err := os.WriteFile(configured, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// PATH has no codex; only the explicit Core override should win.
	t.Setenv("PATH", t.TempDir())
	t.Setenv("FLEET_CODEX_BINARY", configured)

	plan, err := Codex{}.Plan(TaskContext{
		Ref:        "CORE-151",
		WorkingDir: "/tmp/core.eggs.gd",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.LiveReady {
		t.Fatalf("configured FLEET_CODEX_BINARY should be live-ready: %s", plan.LiveNotes)
	}
	if plan.Command[0] != configured {
		t.Fatalf("command[0] = %q, want configured %q", plan.Command[0], configured)
	}
	if !strings.Contains(plan.LiveNotes, "Resolved Codex binary: "+configured) {
		t.Fatalf("live notes should expose configured binary: %s", plan.LiveNotes)
	}
}

func TestCodexPlanRejectsInvalidConfiguredBinaryPath(t *testing.T) {
	for _, tc := range []struct {
		name       string
		configure  func(t *testing.T) string
		wantReason string
	}{
		{
			name: "relative",
			configure: func(t *testing.T) string {
				return "bin/codex"
			},
			wantReason: "not an absolute path",
		},
		{
			name: "not_executable",
			configure: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "codex")
				if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantReason: "not executable",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PATH", t.TempDir())
			t.Setenv("FLEET_CODEX_BINARY", tc.configure(t))

			plan, err := Codex{}.Plan(TaskContext{
				Ref:        "CORE-151",
				WorkingDir: "/tmp/core.eggs.gd",
			})
			if err != nil {
				t.Fatal(err)
			}
			if plan.LiveReady {
				t.Fatalf("invalid FLEET_CODEX_BINARY should not be live-ready: %#v", plan)
			}
			if len(plan.Command) != 0 {
				t.Fatalf("invalid FLEET_CODEX_BINARY should not produce a command: %#v", plan.Command)
			}
			if !strings.Contains(plan.LiveNotes, tc.wantReason) || !strings.Contains(plan.LiveNotes, "Codex CLI is not available on the daemon PATH") {
				t.Fatalf("live notes = %q, want setup diagnostic containing %q", plan.LiveNotes, tc.wantReason)
			}
		})
	}
}

func TestGeminiPlanIsTerminalResumable(t *testing.T) {
	binary := installFakeExecutable(t, "agy")

	plan, err := Gemini{}.Plan(TaskContext{Ref: "TEST-9", WorkingDir: "/tmp/repo"})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.LiveReady {
		t.Fatalf("Gemini plan must be live-ready: %s", plan.LiveNotes)
	}
	if plan.VisibilityClass != "cli_visible" {
		t.Fatalf("visibility = %q, want cli_visible", plan.VisibilityClass)
	}
	if plan.Command[0] != binary || plan.Backend != BackendGeminiHeadless {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestEveryAdapterStatesItsCapabilities(t *testing.T) {
	want := map[string]struct {
		visibility string
		reuse      string
	}{
		"claude": {"app_visible", "verified"},
		"codex":  {"fleet_visible", "unverified"},
		"cursor": {"cli_visible", "verified"},
		"gemini": {"cli_visible", "verified"},
	}
	agents := NewRegistry().Agents()
	if len(agents) != len(want) {
		t.Fatalf("registered %d adapters, expected %d", len(agents), len(want))
	}
	for _, agent := range agents {
		expect, ok := want[agent.Name()]
		if !ok {
			t.Fatalf("unexpected adapter %q", agent.Name())
		}
		caps := agent.Capabilities()
		if caps.Provider != agent.Name() || caps.Backend == "" {
			t.Errorf("%s: provider/backend not set: %#v", agent.Name(), caps)
		}
		if caps.OperatorVisibility != expect.visibility {
			t.Errorf("%s: visibility = %q, want %q", agent.Name(), caps.OperatorVisibility, expect.visibility)
		}
		reuse := agent.SessionReuse()
		if string(reuse.Level) != expect.reuse || reuse.Backend != caps.Backend {
			t.Errorf("%s: reuse = %#v, want level %s on backend %s", agent.Name(), reuse, expect.reuse, caps.Backend)
		}
		if !caps.CanExposeOperatorPath {
			t.Errorf("%s: an adapter that can be launched automatically must expose an operator path", agent.Name())
		}
	}
}

func TestCursorPlanBuildsVisibleCommandWithoutPrint(t *testing.T) {
	binary := installFakeExecutable(t, "cursor-agent")

	plan, err := Cursor{}.Plan(TaskContext{
		Ref:        "CORE-56",
		WorkingDir: "/tmp/core.eggs.gd",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.LiveReady {
		t.Fatalf("Cursor should be live-ready through the visible CLI backend: %s", plan.LiveNotes)
	}
	if plan.VisibilityClass != "cli_visible" {
		t.Fatalf("visibility = %q, want cli_visible", plan.VisibilityClass)
	}
	if plan.Backend != "cursor-visible" {
		t.Fatalf("backend = %q, want cursor-visible", plan.Backend)
	}
	wantPrefix := []string{binary, "--trust", "--workspace", "/tmp/core.eggs.gd"}
	if len(plan.Command) != len(wantPrefix)+1 {
		t.Fatalf("command = %#v, want cursor-agent visible command", plan.Command)
	}
	for i, want := range wantPrefix {
		if plan.Command[i] != want {
			t.Fatalf("command[%d] = %q, want %q in %#v", i, plan.Command[i], want, plan.Command)
		}
	}
	if commandHasArg(plan.Command, "--print") || commandHasArg(plan.Command, "-p") || commandHasArg(plan.Command, "--output-format") {
		t.Fatalf("Cursor daemon command must not use print/headless flags: %#v", plan.Command)
	}
	if !strings.Contains(plan.Command[len(plan.Command)-1], "CORE-56") {
		t.Fatalf("prompt command arg missing task ref: %#v", plan.Command)
	}
	if !strings.Contains(plan.LiveNotes, "create-chat") || !strings.Contains(plan.LiveNotes, "--resume <chat_id>") {
		t.Fatalf("live notes should document Cursor chat identity and resume: %s", plan.LiveNotes)
	}
}

func TestAgentInterfaceDoesNotOwnProcessStart(t *testing.T) {
	agentType := reflect.TypeOf((*Agent)(nil)).Elem()
	if _, ok := agentType.MethodByName("Start"); ok {
		t.Fatal("provider planner must not own process start; runtime owns process lifecycle")
	}
}
