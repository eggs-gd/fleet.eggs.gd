package server

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"time"

	"github.com/eggs-gd/core.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/core.eggs.gd/internal/taskprovider/markdown"
)

func TestExecutionFinalizerConsumesChannelIndependently(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-finalizer-channel.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-113C", "Contour 3 channel", "doing"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: true})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	app.Store.UpsertTask(task)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if app.FinalizerProc == nil {
		t.Fatal("finalizerProc not wired")
	}
	go app.FinalizerProc.Process(ctx)

	app.Exec.PublishExecutionResult(taskflow.ExecutionResult{
		TaskID:      task.RelativePath,
		ExecutionID: "claim-channel",
		Agent:       "codex",
		Outcome:     taskflow.ExecutionCompleted,
		Summary:     "channel-driven finalization",
	})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		loaded, loadErr := markdown.LoadTaskFile(root, taskPath)
		if loadErr == nil && loaded.Status == "needs_review" {
			if len(loaded.Comments) == 0 || !strings.Contains(loaded.Comments[0].Text, "channel-driven finalization") {
				t.Fatalf("comments = %#v", loaded.Comments)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("finalizer did not finalize task via ExecutionResult channel")
}

func TestRuntimeWiresExecutionFinalizationService(t *testing.T) {
	root := t.TempDir()
	app := Compose(ComposeConfig{CoreRoot: root, Interval: time.Second, DryRun: true})
	if app.Listener == nil || app.Exec == nil || app.FinalizerProc == nil {
		t.Fatalf("processors nil: listener=%v exec=%v finalizer=%v",
			app.Listener != nil, app.Exec != nil, app.FinalizerProc != nil)
	}
}
