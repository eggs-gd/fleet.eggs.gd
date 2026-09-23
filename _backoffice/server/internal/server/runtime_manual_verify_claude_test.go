package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"time"

	"github.com/eggs-gd/core.eggs.gd/internal/execution"
	"github.com/eggs-gd/core.eggs.gd/internal/taskprovider/markdown"
)

// TestRuntimeManualVerifyClaudeBackgroundRemoteSession is a one-time manual
// runtime verification, not part of the hermetic suite: it dispatches a real
// `claude --bg --remote-control` process and waits for it to finish. Run
// explicitly with:
//
//	CORE_MANUAL_LIVE_VERIFY=1 go test ./internal/server/... -run TestRuntimeManualVerifyClaudeBackgroundRemoteSession -v -timeout 90s
func TestRuntimeManualVerifyClaudeBackgroundRemoteSession(t *testing.T) {
	if os.Getenv("CORE_MANUAL_LIVE_VERIFY") != "1" {
		t.Skip("set CORE_MANUAL_LIVE_VERIFY=1 to run this live-network manual check")
	}

	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-01-a.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-DIAG", "Manual verify", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: 60 * time.Second})

	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}

	plan, err := execution.DefaultPlanner().Plan("claude", execution.TaskContext{
		Ref:        "CORE-DIAG",
		ID:         task.ID,
		WorkingDir: root,
		LaunchMode: "live-human-input-smoke",
		Body:       "Reply with exactly the word OK and take no other action. Do not read or write any files.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.LiveReady {
		t.Fatalf("plan not live-ready: %s", plan.LiveNotes)
	}
	t.Logf("command: %#v", plan.Command)

	task.LaunchEvaluation.Agent = "claude"
	task.LaunchEvaluation.Backend = plan.Backend
	task.LaunchEvaluation.Command = plan.Command
	task.LaunchEvaluation.WorkingDir = root
	task.LaunchEvaluation.Repository = "core.eggs.gd"

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
	defer cancel()

	app.Exec.StartLaunchCandidate(ctx, task)

	deadline := time.Now().Add(75 * time.Second)
	for time.Now().Before(deadline) {
		sessions := app.State().RuntimeSessions
		if len(sessions) == 0 {
			break
		}
		t.Logf("session state: %#v", sessions[0])
		time.Sleep(2 * time.Second)
	}

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("final task status=%s comments=%#v", loaded.Status, loaded.Comments)

	logBytes, _ := os.ReadFile(filepath.Join(root, "_registry", "sessions"))
	_ = logBytes
	entries, _ := os.ReadDir(filepath.Join(root, "_registry", "sessions"))
	for _, entry := range entries {
		content, _ := os.ReadFile(filepath.Join(root, "_registry", "sessions", entry.Name()))
		t.Logf("session log %s:\n%s", entry.Name(), string(content))
	}
}
