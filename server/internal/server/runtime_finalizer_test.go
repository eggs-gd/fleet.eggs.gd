package server

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/execution"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider/markdown"
)

func TestRuntimeFinalizerReportExecutionOutcomePaths(t *testing.T) {
	cases := []struct {
		name           string
		result         *tasklifecycle.WorkerResult
		wantStatus     string
		wantCommentSub string
	}{
		{
			name:           "completed",
			result:         &tasklifecycle.WorkerResult{Outcome: "completed", Summary: "shipped", Artifacts: []string{"a.go"}, Tests: []string{"go test"}},
			wantStatus:     "needs_review",
			wantCommentSub: "Agent execution completed.",
		},
		{
			name:           "failed",
			result:         &tasklifecycle.WorkerResult{Outcome: "failed", Summary: "could not build", Error: "compile error"},
			wantStatus:     "blocked",
			wantCommentSub: "Agent execution reported failure.",
		},
		{
			name:           "needs_input",
			result:         &tasklifecycle.WorkerResult{Outcome: "needs_input", Summary: "need token", Question: "What is the staging token?"},
			wantStatus:     "blocked",
			wantCommentSub: "waiting on operator input",
		},
		{
			name:           "waiting_input_legacy",
			result:         &tasklifecycle.WorkerResult{Outcome: "waiting_input", Summary: "need approval", Blockers: []string{"missing sign-off"}},
			wantStatus:     "blocked",
			wantCommentSub: "waiting on operator input",
		},
		{
			name:           "needs_rework",
			result:         &tasklifecycle.WorkerResult{Outcome: "needs_rework", Summary: "partial"},
			wantStatus:     "needs_rework",
			wantCommentSub: "needs rework",
		},
		{
			name:           "blocked",
			result:         &tasklifecycle.WorkerResult{Outcome: "blocked", Summary: "hard stop"},
			wantStatus:     "blocked",
			wantCommentSub: "reported blocked",
		},
		{
			name:           "nil_defaults_completed",
			result:         nil,
			wantStatus:     "needs_review",
			wantCommentSub: "Agent execution completed.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-"+tc.name+".md")
			writeTestFile(t, taskPath, testTaskMarkdown("CORE-105", "Finalizer path "+tc.name, "doing"))

			app := Compose(ComposeConfig{CoreRoot: root, DryRun: true})
			task, err := markdown.LoadTaskFile(root, taskPath)
			if err != nil {
				t.Fatal(err)
			}
			app.Store.UpsertTask(task)

			updated, err := app.Exec.ReportWorkerExecution(task, tc.result, "_registry/sessions/test.log")
			if err != nil {
				t.Fatal(err)
			}
			if updated.Status != tc.wantStatus {
				t.Fatalf("status = %q, want %q", updated.Status, tc.wantStatus)
			}
			if len(updated.Comments) == 0 || !strings.Contains(updated.Comments[len(updated.Comments)-1].Text, tc.wantCommentSub) {
				t.Fatalf("comments = %#v, want substring %q", updated.Comments, tc.wantCommentSub)
			}
			if !strings.Contains(updated.Comments[len(updated.Comments)-1].Text, "Execution outcome:") {
				t.Fatalf("comment missing TaskService ReportExecution marker: %#v", updated.Comments)
			}
		})
	}
}

func TestRuntimeFinalizerTimedOutAndCancelledPaths(t *testing.T) {
	for _, tc := range []struct {
		outcome    taskflow.ExecutionOutcome
		wantStatus string
		sub        string
	}{
		{taskflow.ExecutionTimedOut, "blocked", "timed out"},
		{taskflow.ExecutionCancelled, "blocked", "cancelled"},
	} {
		t.Run(string(tc.outcome), func(t *testing.T) {
			root := t.TempDir()
			taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-"+string(tc.outcome)+".md")
			writeTestFile(t, taskPath, testTaskMarkdown("CORE-105", "Runtime "+string(tc.outcome), "doing"))

			app := Compose(ComposeConfig{CoreRoot: root, DryRun: true})
			task, err := markdown.LoadTaskFile(root, taskPath)
			if err != nil {
				t.Fatal(err)
			}
			app.Store.UpsertTask(task)

			updated, err := app.Exec.ReportExecutionResult(task, execution.ExecutionResultFromRuntimeStatus(
				task, tc.outcome, execution.WorkerOutcomeHeadline(tc.outcome, string(tc.outcome)), "boom", "_registry/sessions/x.log",
			))
			if err != nil {
				t.Fatal(err)
			}
			if updated.Status != tc.wantStatus {
				t.Fatalf("status = %q, want %q", updated.Status, tc.wantStatus)
			}
			if len(updated.Comments) == 0 || !strings.Contains(strings.ToLower(updated.Comments[0].Text), tc.sub) {
				t.Fatalf("comments = %#v, want %q", updated.Comments, tc.sub)
			}
		})
	}
}

func TestRuntimeClaimUsesTaskService(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-claim.md")
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
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-105C", "Claim via TaskService", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if app.FinalizerProc == nil {
		t.Fatal("finalizerProc not wired")
	}
	go app.FinalizerProc.Process(ctx)

	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "codex",
		Repository: "core.eggs.gd",
		WorkingDir: root,
		Command:    []string{"/usr/bin/true"},
		Launchable: true,
		Outcome:    "launchable",
	}
	app.Store.UpsertTask(task)

	app.Exec.StartLaunchCandidate(t.Context(), task)
	waitForNoActiveRuntimeSessions(t, app)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		loaded, err := markdown.LoadTaskFile(root, taskPath)
		if err != nil {
			t.Fatal(err)
		}
		// Claim + successful exit with no worker JSON defaults to needs_review via Contour 3.
		if loaded.Status == "needs_review" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Fatalf("status = %q, want needs_review after TaskService claim + Contour 3 finalizer", loaded.Status)
}

func TestExecutionResultFromWorkerMapsLegacyFields(t *testing.T) {
	task := tasklifecycle.Task{RelativePath: "Work/x/tasks/y.md", Path: "/tmp/y.md"}
	got := execution.ExecutionResultFromWorker(task, &tasklifecycle.WorkerResult{
		Outcome:             "waiting_input",
		Summary:             "need help",
		Blockers:            []string{"access"},
		ReviewNotes:         "ping Alex",
		SuggestedNextStatus: "todo",
		Artifacts:           []string{"a.go"},
		Tests:               []string{"go test"},
	}, "sessions/a.log")
	if got.Outcome != taskflow.ExecutionNeedsInput {
		t.Fatalf("outcome = %q", got.Outcome)
	}
	if got.Question != "access" {
		t.Fatalf("question = %q", got.Question)
	}
	if !strings.Contains(got.Summary, "ping Alex") || !strings.Contains(got.Summary, "Worker-suggested next status") {
		t.Fatalf("summary = %q", got.Summary)
	}
	if !strings.Contains(got.Summary, "Session log:") {
		t.Fatalf("summary missing session log: %q", got.Summary)
	}
}

func TestRuntimeHITLNeedsInputPausesAndHoldsLaunchSlot(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-hitl.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-HITL", "Ask human", "doing"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: true})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	app.Store.UpsertTask(task)

	session := execution.RuntimeSession{
		ClaimID:         "hitl-claim",
		TaskRef:         task.Ref,
		TaskID:          task.ID,
		TaskPath:        task.RelativePath,
		ProjectID:       firstNonEmpty(task.ProjectID, "core-eggs-gd"),
		Repository:      "core.eggs.gd",
		Agent:           "codex",
		Status:          "running",
		ExecutionStatus: "running",
	}
	app.Exec.UpsertSession(session)

	app.Exec.ApplyWorkerReportedResult(&session, task, &tasklifecycle.WorkerResult{
		Outcome:  "needs_input",
		Summary:  "need token",
		Question: "What is the staging token?",
	})

	reloaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Status != "doing" {
		t.Fatalf("status = %q, want doing (HITL is a pause)", reloaded.Status)
	}
	if len(reloaded.Comments) == 0 || !strings.Contains(reloaded.Comments[len(reloaded.Comments)-1].Text, "waiting on operator input") {
		t.Fatalf("comments = %#v, want HITL question comment", reloaded.Comments)
	}

	live, ok := app.Exec.Session("hitl-claim")
	if !ok || !live.IsActive() || live.ExecutionStatus != "waiting_input" {
		t.Fatalf("session = %#v, want active waiting_input", live)
	}

	app.Exec.CompleteProviderSession(live, reloaded, "process exited after HITL", nil)
	afterComplete, ok := app.Exec.Session("hitl-claim")
	if !ok || !afterComplete.IsActive() {
		t.Fatal("CompleteProviderSession released the HITL pause")
	}

	_, _, reserved := app.Exec.TryReserveSession(execution.RuntimeSession{
		ClaimID:    "next-claim",
		ProjectID:  session.ProjectID,
		Repository: session.Repository,
		Agent:      session.Agent,
		Status:     "starting",
	})
	if reserved {
		t.Fatal("1-1-1 allowed a second session while HITL still holds the slot")
	}

	if _, err := app.PatchTask(tasklifecycle.TaskPatch{Path: taskPath, Status: "blocked"}); err != nil {
		t.Fatal(err)
	}
	released, ok := app.Exec.Session("hitl-claim")
	if ok && released.IsActive() {
		t.Fatalf("session still active after operator left doing: %#v", released)
	}
	_, _, reserved = app.Exec.TryReserveSession(execution.RuntimeSession{
		ClaimID:    "after-human",
		ProjectID:  session.ProjectID,
		Repository: session.Repository,
		Agent:      session.Agent,
		Status:     "starting",
	})
	if !reserved {
		t.Fatal("1-1-1 should free after the human moves the HITL task off doing")
	}
}
