package server

import (
	"testing"

	"github.com/eggs-gd/core.eggs.gd/internal/execution"
	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
)

func TestAnnotateTaskClearsStaleAssigneeOrphanWaitWhenBlockerTaskResolved(t *testing.T) {
	root := t.TempDir()
	app := Compose(ComposeConfig{CoreRoot: root, DryRun: true})
	blocker := taskForLaunchWaitTest("CORE-57", "work-core-57", "needs_review")
	candidate := taskForLaunchWaitTest("CORE-79", "work-core-79", "needs_rework")
	candidate.MarkLaunchWaiting(tasklifecycle.LaunchWait{
		Scope:           "assignee_orphan",
		Resource:        "codex",
		ConflictClaimID: "orphan:CORE-57",
		Reason:          `assignee "codex" has unresolved orphaned doing task CORE-57; move the orphan to blocked, needs_rework, todo, done, or another valid state to release this launch slot`,
		WaitingSince:    "2026-08-02T01:00:00Z",
	})

	app.Store.UpsertTask(blocker)
	app.Store.UpsertTask(candidate)
	app.Exec.UpsertOrphan(execution.OrphanedTask{
		TaskRef:    blocker.Ref,
		TaskID:     blocker.ID,
		TaskPath:   blocker.RelativePath,
		Assignee:   "codex",
		DetectedAt: "2026-08-02T01:00:00Z",
		Reason:     "task was doing on startup",
	})

	got := app.Exec.AnnotateTask(candidate)
	if got.LaunchEvaluation.Waiting != nil {
		t.Fatalf("waiting = %#v, want stale CORE-57 orphan wait cleared", got.LaunchEvaluation.Waiting)
	}
}

func TestAnnotateTaskClearsStaleAssigneeOrphanWaitWhenOrphanResolved(t *testing.T) {
	root := t.TempDir()
	app := Compose(ComposeConfig{CoreRoot: root, DryRun: true})
	blocker := taskForLaunchWaitTest("CORE-57", "work-core-57", "doing")
	candidate := taskForLaunchWaitTest("CORE-79", "work-core-79", "needs_rework")
	candidate.MarkLaunchWaiting(tasklifecycle.LaunchWait{
		Scope:           "assignee_orphan",
		Resource:        "codex",
		ConflictClaimID: "orphan:CORE-57",
		Reason:          `assignee "codex" has unresolved orphaned doing task CORE-57; move the orphan to blocked, needs_rework, todo, done, or another valid state to release this launch slot`,
		WaitingSince:    "2026-08-02T01:00:00Z",
	})

	app.Store.UpsertTask(blocker)
	app.Store.UpsertTask(candidate)

	got := app.Exec.AnnotateTask(candidate)
	if got.LaunchEvaluation.Waiting != nil {
		t.Fatalf("waiting = %#v, want stale wait cleared after orphan record resolves", got.LaunchEvaluation.Waiting)
	}
}

func TestAnnotateTaskKeepsStillActiveAssigneeOrphanWait(t *testing.T) {
	root := t.TempDir()
	app := Compose(ComposeConfig{CoreRoot: root, DryRun: true})
	blocker := taskForLaunchWaitTest("CORE-57", "work-core-57", "doing")
	candidate := taskForLaunchWaitTest("CORE-79", "work-core-79", "needs_rework")
	wait := tasklifecycle.LaunchWait{
		Scope:           "assignee_orphan",
		Resource:        "codex",
		ConflictClaimID: "orphan:CORE-57",
		Reason:          `assignee "codex" has unresolved orphaned doing task CORE-57; move the orphan to blocked, needs_rework, todo, done, or another valid state to release this launch slot`,
		WaitingSince:    "2026-08-02T01:00:00Z",
	}
	candidate.MarkLaunchWaiting(wait)

	app.Store.UpsertTask(blocker)
	app.Store.UpsertTask(candidate)
	app.Exec.UpsertOrphan(execution.OrphanedTask{
		TaskRef:    blocker.Ref,
		TaskID:     blocker.ID,
		TaskPath:   blocker.RelativePath,
		Assignee:   "codex",
		DetectedAt: "2026-08-02T01:00:00Z",
		Reason:     "task was doing on startup",
	})

	got := app.Exec.AnnotateTask(candidate)
	if got.LaunchEvaluation.Waiting == nil {
		t.Fatal("waiting = nil, want active orphan wait preserved")
	}
	if got.LaunchEvaluation.Waiting.Scope != wait.Scope || got.LaunchEvaluation.Waiting.Resource != wait.Resource {
		t.Fatalf("waiting = %#v, want scope/resource from %#v", got.LaunchEvaluation.Waiting, wait)
	}
}

func taskForLaunchWaitTest(ref string, id string, status string) tasklifecycle.Task {
	return tasklifecycle.Task{
		Ref:          ref,
		ID:           id,
		Status:       status,
		Assignee:     "codex",
		RelativePath: "Work/core-eggs-gd/tasks/" + ref + ".md",
		Path:         "Work/core-eggs-gd/tasks/" + ref + ".md",
	}
}

func TestUpsertSessionPreservesExistingWorkerResult(t *testing.T) {
	root := t.TempDir()
	app := Compose(ComposeConfig{CoreRoot: root, DryRun: true})
	app.Exec.UpsertSession(execution.RuntimeSession{
		ClaimID:         "core-99-result",
		TaskRef:         "CORE-99",
		Status:          "running",
		ExecutionStatus: "running",
		ClaimedAt:       "2026-08-03T20:00:00Z",
		Result:          &tasklifecycle.WorkerResult{Outcome: "completed", Summary: "keep me"},
	})
	app.Exec.UpsertSession(execution.RuntimeSession{
		ClaimID:         "core-99-result",
		TaskRef:         "CORE-99",
		Status:          "running",
		ExecutionStatus: "running",
		ClaimedAt:       "2026-08-03T20:00:00Z",
		LastEvent:       "transcript_updated",
		LastMessage:     "transcript bytes=12",
	})

	session, ok := app.Exec.Session("core-99-result")
	if !ok {
		t.Fatal("missing session")
	}
	if session.Result == nil || session.Result.Outcome != "completed" || session.Result.Summary != "keep me" {
		t.Fatalf("session.Result = %#v, want previously parsed result preserved across heartbeat upsert", session.Result)
	}
	records, err := execution.LoadRuntimeSessionRecords(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Result == nil || records[0].Result.Summary != "keep me" {
		t.Fatalf("persisted records = %#v, want result kept on disk", records)
	}
}

func TestStatusRuntimeSessionsIncludesOnlyActiveSessions(t *testing.T) {
	app := Compose(ComposeConfig{CoreRoot: t.TempDir(), DryRun: true})
	app.Exec.UpsertSession(execution.RuntimeSession{
		ClaimID:         "active-1",
		TaskRef:         "CORE-98-A",
		Status:          "running",
		ExecutionStatus: "running",
		ClaimedAt:       "2026-08-03T20:00:00Z",
	})
	app.Exec.UpsertSession(execution.RuntimeSession{
		ClaimID:         "terminal-1",
		TaskRef:         "CORE-96",
		Status:          "exited",
		ExecutionStatus: "provider_error",
		ClaimedAt:       "2026-08-03T19:00:00Z",
		ProviderError:   &tasklifecycle.ProviderError{Provider: "claude", Kind: "rate_limited", Reason: "stale"},
	})
	app.Exec.UpsertSession(execution.RuntimeSession{
		ClaimID:         "succeeded-1",
		TaskRef:         "CORE-97",
		Status:          "exited",
		ExecutionStatus: "succeeded",
		ClaimedAt:       "2026-08-03T18:00:00Z",
	})

	status := app.Exec.Status()
	if len(status.RuntimeSessions) != 1 {
		t.Fatalf("runtime sessions = %#v, want only the active session", status.RuntimeSessions)
	}
	if status.RuntimeSessions[0].ClaimID != "active-1" {
		t.Fatalf("runtime session = %#v, want active-1", status.RuntimeSessions[0])
	}
	foundTerminal := false
	for _, group := range status.SessionGroups {
		for _, session := range group.Sessions {
			if session.ClaimID == "terminal-1" || session.ClaimID == "succeeded-1" {
				foundTerminal = true
				if session.IsActive() {
					t.Fatalf("historical session still active: %#v", session)
				}
			}
		}
	}
	if !foundTerminal {
		t.Fatalf("session_groups = %#v, want terminal/succeeded history retained", status.SessionGroups)
	}
}
