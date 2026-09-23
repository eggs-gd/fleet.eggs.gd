package server

import (
	"path/filepath"
	"strings"
	"testing"

	"time"

	"github.com/eggs-gd/core.eggs.gd/internal/execution"
	"github.com/eggs-gd/core.eggs.gd/internal/taskprovider/markdown"
)

func TestOperatorReleaseStaleClaudeSessionByClaimID(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-84-claude.md")
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
	writeTestFile(t, taskPath, testTaskMarkdownWithID("work-core-84-claude", "CORE-84-CLAUDE", "Stale Claude execution", "doing"))
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	app.Exec.UpsertSession(execution.RuntimeSession{
		ClaimID:    "core-84-claude-claim",
		TaskRef:    task.Ref,
		TaskID:     task.ID,
		TaskPath:   task.RelativePath,
		ProjectID:  task.ProjectID,
		Repository: "core.eggs.gd",
		Agent:      "claude",
		Backend:    "background-remote",
		LogPath:    "_registry/sessions/core-84-claude-claim.log",
		ClaudeSessionDetails: execution.ClaudeSessionDetails{
			BackgroundID:     "bg-stale-84",
			RemoteControlURL: "https://claude.ai/code/session_STALE84",
		},
		ClaimedAt:       "2026-08-02T10:00:00+03:00",
		StartedAt:       "2026-08-02T10:01:00+03:00",
		Status:          "resumable",
		ExecutionStatus: "resumable",
	})

	if _, ok := app.Exec.ActiveSessionForTask(task); !ok {
		t.Fatal("expected active stale Claude session before release")
	}

	if err := app.ControlSession(execution.SessionControlPatch{
		ClaimID:      "core-84-claude-claim",
		Action:       "release",
		TargetStatus: "needs_review",
	}); err != nil {
		t.Fatal(err)
	}

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "needs_review" {
		t.Fatalf("task status = %q, want needs_review", loaded.Status)
	}
	if len(loaded.Comments) == 0 || !strings.Contains(loaded.Comments[0].Text, "Operator released runtime execution core-84-claude-claim") {
		t.Fatalf("comments = %#v, want operator release comment", loaded.Comments)
	}

	session, ok := app.Exec.Session("core-84-claude-claim")
	if !ok {
		t.Fatal("expected persisted released session")
	}
	if session.IsActive() || session.ExecutionStatus != "released" {
		t.Fatalf("session = %#v, want non-active released", session)
	}
	if _, ok := app.Exec.ActiveSessionForTask(loaded); ok {
		t.Fatal("active execution guard should be cleared after release")
	}

	assertEventLogContains(t, root, `"type":"runtime_session_operator_released"`)
	assertEventLogContains(t, root, `"type":"runtime_slot_released"`)
}

func TestOperatorReleaseStaleCodexSessionByClaimID(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-84-codex.md")
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
	writeTestFile(t, taskPath, testTaskMarkdownWithID("work-core-84-codex", "CORE-84-CODEX", "Stale Codex execution", "doing"))
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	app.Exec.UpsertSession(execution.RuntimeSession{
		ClaimID:    "core-84-codex-claim",
		TaskRef:    task.Ref,
		TaskID:     task.ID,
		TaskPath:   task.RelativePath,
		ProjectID:  task.ProjectID,
		Repository: "core.eggs.gd",
		Agent:      "codex",
		Backend:    execution.BackendCodexAppServer,
		LogPath:    "_registry/sessions/core-84-codex-claim.log",
		CodexSessionDetails: execution.CodexSessionDetails{
			CodexThreadID: "thread_stale_84",
		},
		ClaimedAt:       "2026-08-02T10:00:00+03:00",
		StartedAt:       "2026-08-02T10:01:00+03:00",
		Status:          "resumable",
		ExecutionStatus: "resumable",
	})

	if err := app.ControlSession(execution.SessionControlPatch{
		ClaimID:      "core-84-codex-claim",
		Action:       "mark_dead",
		TargetStatus: "blocked",
	}); err != nil {
		t.Fatal(err)
	}

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "blocked" {
		t.Fatalf("task status = %q, want blocked", loaded.Status)
	}
	if len(loaded.Comments) == 0 || !strings.Contains(loaded.Comments[0].Text, "Operator marked runtime execution core-84-codex-claim dead") {
		t.Fatalf("comments = %#v, want mark_dead comment", loaded.Comments)
	}

	session, ok := app.Exec.Session("core-84-codex-claim")
	if !ok || session.IsActive() || session.ExecutionStatus != "dead" {
		t.Fatalf("session = %#v ok=%v, want non-active dead", session, ok)
	}
	assertEventLogContains(t, root, `"action":"mark_dead"`)
}

func TestOperatorReleaseWakesLaunchQueue(t *testing.T) {
	root, _ := writeRuntimeRequeueFixture(t)
	installFakeCodexAppServer(t, root)

	stalePath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-84-stale.md")
	nextPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-84-next.md")
	writeTestFile(t, stalePath, testTaskMarkdownWithID("work-core-84-stale", "CORE-84-STALE", "Stale slot holder", "doing"))
	writeTestFile(t, nextPath, allowCoreVisibleLaunch(testTaskMarkdownWithID("work-core-84-next", "CORE-84-NEXT", "Next after release", "todo")))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	stale, err := markdown.LoadTaskFile(root, stalePath)
	if err != nil {
		t.Fatal(err)
	}
	app.Exec.UpsertSession(execution.RuntimeSession{
		ClaimID:    "core-84-stale-claim",
		TaskRef:    stale.Ref,
		TaskID:     stale.ID,
		TaskPath:   stale.RelativePath,
		ProjectID:  stale.ProjectID,
		Repository: "core.eggs.gd",
		Agent:      "codex",
		Backend:    execution.BackendCodexAppServer,
		LogPath:    "_registry/sessions/core-84-stale-claim.log",
		CodexSessionDetails: execution.CodexSessionDetails{
			CodexThreadID: "thread_stale_queue",
		},
		ClaimedAt:       "2026-08-02T11:00:00+03:00",
		Status:          "resumable",
		ExecutionStatus: "resumable",
	})

	if err := app.ControlSession(execution.SessionControlPatch{
		ClaimID:      "core-84-stale-claim",
		Action:       "release",
		TargetStatus: "needs_review",
	}); err != nil {
		t.Fatal(err)
	}

	waitForTaskStatus(t, root, nextPath, "needs_review")
	assertEventLogContains(t, root, `"type":"runtime_slot_released"`)
	assertEventLogContains(t, root, `"type":"launch_requeue_selected"`)
}

func TestOperatorReleaseRejectsMissingClaimID(t *testing.T) {
	app := Compose(ComposeConfig{CoreRoot: t.TempDir()})
	err := app.ControlSession(execution.SessionControlPatch{Action: "release"})
	if err == nil || !strings.Contains(err.Error(), "claim_id") {
		t.Fatalf("error = %v, want claim_id required", err)
	}
}

func TestOperatorReleaseRejectsInvalidTargetStatus(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-84-invalid.md")
	writeTestFile(t, filepath.Join(root, "Work", "core-eggs-gd", "PROJECT.md"), `---
id: core-eggs-gd
title: Core
---
`)
	writeTestFile(t, taskPath, testTaskMarkdownWithID("work-core-84-invalid", "CORE-84-INVALID", "Invalid target", "doing"))
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	app.Exec.UpsertSession(execution.RuntimeSession{
		ClaimID:         "core-84-invalid-claim",
		TaskRef:         task.Ref,
		TaskID:          task.ID,
		TaskPath:        task.RelativePath,
		ProjectID:       task.ProjectID,
		Agent:           "claude",
		Backend:         "background-remote",
		Status:          "resumable",
		ExecutionStatus: "resumable",
		ClaimedAt:       "2026-08-02T10:00:00+03:00",
	})
	err = app.ControlSession(execution.SessionControlPatch{
		ClaimID:      "core-84-invalid-claim",
		Action:       "release",
		TargetStatus: "done",
	})
	if err == nil || !strings.Contains(err.Error(), "target_status") {
		t.Fatalf("error = %v, want target_status validation", err)
	}
}

func TestOperatorReleaseClearsActiveExecutionGuard(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-84-guard.md")
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
	writeTestFile(t, taskPath, testTaskMarkdownWithID("work-core-84-guard", "CORE-84-GUARD", "Guard release", "doing"))
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	app.Exec.UpsertSession(execution.RuntimeSession{
		ClaimID:         "core-84-guard-claim",
		TaskRef:         task.Ref,
		TaskID:          task.ID,
		TaskPath:        task.RelativePath,
		ProjectID:       task.ProjectID,
		Repository:      "core.eggs.gd",
		Agent:           "claude",
		Backend:         "background-remote",
		Status:          "resumable",
		ExecutionStatus: "resumable",
		ClaimedAt:       "2026-08-02T10:00:00+03:00",
	})

	if err := app.ControlSession(execution.SessionControlPatch{
		ClaimID:      "core-84-guard-claim",
		Action:       "cancel_without_provider_control",
		TargetStatus: "needs_rework",
	}); err != nil {
		t.Fatal(err)
	}

	// After operator release there is no active session, so observe must keep
	// needs_rework instead of forcing the task back to doing.
	observed := observeTaskFileChange(t, app, taskPath)
	if observed.GuardApplied {
		t.Fatal("did not expect active execution guard after operator release")
	}
	if observed.After.Status != "needs_rework" {
		t.Fatalf("observed status = %q, want needs_rework (not forced back to doing)", observed.After.Status)
	}
}
