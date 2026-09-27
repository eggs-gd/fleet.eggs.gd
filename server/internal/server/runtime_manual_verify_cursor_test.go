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

// TestRuntimeManualVerifyCursorSession is a manual runtime smoke check, not
// part of the hermetic suite: it starts a real visible `cursor-agent`
// session, creates/persists a Cursor chat id, and waits for the CLI process
// to finish. Run explicitly from an unrestricted host account with Cursor
// auth and writable local state:
//
//	FLEET_MANUAL_LIVE_VERIFY=1 go test ./internal/server/... -run TestRuntimeManualVerifyCursorSession -v -timeout 3m
func TestRuntimeManualVerifyCursorSession(t *testing.T) {
	if os.Getenv("FLEET_MANUAL_LIVE_VERIFY") != "1" {
		t.Skip("set FLEET_MANUAL_LIVE_VERIFY=1 to run this live-network manual check")
	}

	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-01-cursor-smoke.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-CURSOR-SMOKE", "Manual Cursor verify", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: 90 * time.Second})

	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}

	body := `Low-stakes Cursor smoke test.

Create a file named cursor-smoke-output.txt in the workspace containing exactly CORE_CURSOR_SMOKE_READY.
Then update this task card status to needs_review.
Do not commit, push, or open a pull request.`
	plan, err := execution.DefaultPlanner().Plan("cursor", execution.TaskContext{
		Ref:          "CORE-CURSOR-SMOKE",
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
		Assignee:     "cursor",
		Body:         body,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.LiveReady {
		t.Skipf("Cursor visible mode is not live-ready in this environment: %s", plan.LiveNotes)
	}
	t.Logf("command: %#v", plan.Command)

	task.LaunchEvaluation.Agent = "cursor"
	task.LaunchEvaluation.Backend = plan.Backend
	task.LaunchEvaluation.Command = plan.Command
	task.LaunchEvaluation.WorkingDir = root
	task.LaunchEvaluation.Repository = "core.eggs.gd"

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	app.Exec.StartLaunchCandidate(ctx, task)

	waitForLaunchToFinish(t, app, root, taskPath, 110*time.Second)
	if sessions := app.State().RuntimeSessions; len(sessions) != 0 {
		t.Fatalf("runtime still has active sessions after deadline: %#v", sessions)
	}

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(filepath.Join(root, "_registry", "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("no session log was written")
	}
	for _, entry := range entries {
		content, _ := os.ReadFile(filepath.Join(root, "_registry", "sessions", entry.Name()))
		t.Logf("session log %s:\n%s", entry.Name(), string(content))
		if !strings.Contains(string(content), "CORE_CURSOR_SMOKE_READY") {
			t.Fatalf("session log %s does not contain smoke marker", entry.Name())
		}
	}

	if loaded.Status != "needs_review" {
		t.Fatalf("final task status = %q, want needs_review; comments=%#v", loaded.Status, loaded.Comments)
	}

	artifact, err := os.ReadFile(filepath.Join(root, "cursor-smoke-output.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(artifact)) != "CORE_CURSOR_SMOKE_READY" {
		t.Fatalf("artifact = %q, want CORE_CURSOR_SMOKE_READY", string(artifact))
	}
}
