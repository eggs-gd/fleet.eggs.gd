package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/execution"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider/markdown"
)

// TestRuntimeManualVerifyCodexSession is a manual live runtime verification,
// not part of the hermetic suite: it starts a real `codex app-server --stdio`
// process through Core's runtime launcher, waits for a low-stakes turn to
// complete, and verifies log capture plus runtime task finalization. Run
// explicitly from an unrestricted host account with Codex auth and network
// access available:
//
//	CORE_MANUAL_LIVE_VERIFY=1 go test ./internal/server/... -run TestRuntimeManualVerifyCodexSession -v -timeout 3m
func TestRuntimeManualVerifyCodexSession(t *testing.T) {
	if os.Getenv("CORE_MANUAL_LIVE_VERIFY") != "1" {
		t.Skip("set CORE_MANUAL_LIVE_VERIFY=1 to run this live-network manual check")
	}

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "_docs", "MANAGER.md"), "# Test Manager\n")
	writeTestFile(t, filepath.Join(root, "_docs", "OPERATING_MODEL.md"), "# Test Operating Model\n")
	writeTestFile(t, filepath.Join(root, "Fleet", "ROUTING.md"), "# Test Routing\n")
	writeTestFile(t, filepath.Join(root, "Fleet", "LAUNCH_POLICY.md"), "# Test Launch Policy\n")
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
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-codex-app-server-smoke.md")
	writeTestFile(t, taskPath, testTaskMarkdownWithID("work-codex-app-server-smoke", "CORE-CODEX-SMOKE", "Manual Codex app-server verify", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: 45 * time.Second})

	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}

	body := `Low-stakes Codex app-server smoke test.

Do not edit files. Do not run commands. Do not commit, push, or open a pull request.
Return exactly this compact JSON result payload and no other final text:
{"outcome":"completed","summary":"CORE_CODEX_APP_SERVER_SMOKE_OK","artifacts":["_registry/sessions/<claim_id>.log"],"tests":["CORE_MANUAL_LIVE_VERIFY=1 go test ./internal/server/... -run TestRuntimeManualVerifyCodexSession -v -timeout 3m"]}`
	plan, err := execution.DefaultPlanner().Plan("codex", execution.TaskContext{
		Ref:          "CORE-CODEX-SMOKE",
		ID:           task.ID,
		Title:        task.Title,
		Status:       task.Status,
		Type:         task.Type,
		Priority:     task.Priority,
		ProjectID:    task.ProjectID,
		WorkspaceID:  task.WorkspaceID,
		Repository:   "core.eggs.gd",
		RelativePath: task.RelativePath,
		WorkingDir:   root,
		Assignee:     "codex",
		LaunchMode:   execution.LaunchModeAllowCoreVisible,
		Body:         body,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.LiveReady {
		t.Skipf("Codex app-server is not live-ready in this environment: %s", plan.LiveNotes)
	}
	t.Logf("command: %#v visibility=%s", plan.Command, plan.VisibilityClass)

	task.Launch.Mode = execution.LaunchModeAllowCoreVisible
	task.LaunchEvaluation.Agent = "codex"
	task.LaunchEvaluation.Backend = plan.Backend
	task.LaunchEvaluation.VisibilityMode = plan.VisibilityClass
	task.LaunchEvaluation.Command = plan.Command
	task.LaunchEvaluation.WorkingDir = root
	task.LaunchEvaluation.Repository = "core.eggs.gd"
	task.LaunchEvaluation.Prompt = plan.Prompt

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	app.Exec.StartLaunchCandidate(ctx, task)

	deadline := time.Now().Add(140 * time.Second)
	for time.Now().Before(deadline) {
		if len(app.State().RuntimeSessions) == 0 {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if sessions := app.State().RuntimeSessions; len(sessions) != 0 {
		t.Fatalf("runtime still has active sessions after deadline: %#v", sessions)
	}

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	logText := readOnlySessionLog(t, root)
	if loaded.Status != "needs_review" {
		t.Fatalf("final task status = %q, want needs_review; comments=%#v\n\n%s", loaded.Status, loaded.Comments, logText)
	}

	for _, want := range []string{
		`"method":"initialize"`,
		`"method":"thread/start"`,
		`"method":"thread/name/set"`,
		`"method":"turn/start"`,
		`"method":"turn/completed"`,
		"CORE_CODEX_APP_SERVER_SMOKE_OK",
		`"outcome":"completed"`,
	} {
		if !strings.Contains(logText, want) {
			t.Fatalf("session log missing %s\n\n%s", want, logText)
		}
	}
}

func readOnlySessionLog(t *testing.T, root string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "_registry", "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	var logs []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".log") {
			content, err := os.ReadFile(filepath.Join(root, "_registry", "sessions", entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("session log %s:\n%s", entry.Name(), string(content))
			logs = append(logs, string(content))
			continue
		}
		if strings.HasSuffix(entry.Name(), ".json") {
			content, err := os.ReadFile(filepath.Join(root, "_registry", "sessions", entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("session record %s:\n%s", entry.Name(), string(content))
		}
	}
	if len(logs) != 1 {
		t.Fatalf("session logs = %d, want one", len(logs))
	}
	return logs[0]
}
