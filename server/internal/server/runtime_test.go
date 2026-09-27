package server

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/audit"

	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/execution"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider/markdown"
)

const (
	codexAppServerBackend = execution.BackendCodexAppServer
	cursorVisibleBackend  = execution.BackendCursorVisible
	capabilityYes         = execution.CapabilityYes
	capabilityUnknown     = execution.CapabilityUnknown
)

func TestDryRunBlocksProcessStart(t *testing.T) {
	if !(ComposeConfig{DryRun: true}).DryRun {
		t.Fatal("dry-run config allowed process start")
	}
	if (ComposeConfig{DryRun: false}).DryRun {
		t.Fatal("non-dry-run config blocked process start")
	}
}

func TestStoreDetectsActiveSessionConflict(t *testing.T) {
	app := Compose(ComposeConfig{CoreRoot: t.TempDir(), DryRun: true})
	app.Exec.UpsertSession(execution.RuntimeSession{
		ClaimID:    "claim-1",
		ProjectID:  "core-eggs-gd",
		Repository: "core.eggs.gd",
		Agent:      "claude",
		Status:     "running",
		ClaimedAt:  "2026-08-01T10:00:00+03:00",
	})

	session, scope, ok := app.Exec.TryReserveSession(execution.RuntimeSession{
		ClaimID:   "claim-2",
		ProjectID: "core-eggs-gd",
		Status:    "starting",
	})
	if ok {
		t.Fatal("expected project session conflict")
	}
	if scope != "project" || session.ClaimID != "claim-1" {
		t.Fatalf("conflict = %#v scope=%q", session, scope)
	}

	session, scope, ok = app.Exec.TryReserveSession(execution.RuntimeSession{
		ClaimID:    "claim-3",
		Repository: "core.eggs.gd",
		Status:     "starting",
	})
	if ok {
		t.Fatal("expected repository session conflict")
	}
	if scope != "repository" || session.ClaimID != "claim-1" {
		t.Fatalf("conflict = %#v scope=%q", session, scope)
	}

	session, scope, ok = app.Exec.TryReserveSession(execution.RuntimeSession{
		ClaimID: "claim-4",
		Agent:   "claude",
		Status:  "starting",
	})
	if ok {
		t.Fatal("expected assignee session conflict")
	}
	if scope != "assignee" || session.ClaimID != "claim-1" {
		t.Fatalf("assignee conflict = %#v scope=%q", session, scope)
	}
}

func TestStoreDetectsAssigneeConflictDuringProviderStartup(t *testing.T) {
	app := Compose(ComposeConfig{CoreRoot: t.TempDir(), DryRun: true})
	if _, _, ok := app.Exec.TryReserveSession(execution.RuntimeSession{
		ClaimID:         "claim-startup",
		ProjectID:       "project-a",
		Repository:      "repo-a",
		Agent:           "claude",
		Status:          "starting",
		ExecutionStatus: "waiting_for_visible_session",
		ClaimedAt:       "2026-09-08T12:00:00+03:00",
	}); !ok {
		t.Fatal("first reservation failed")
	}

	conflict, scope, ok := app.Exec.TryReserveSession(execution.RuntimeSession{
		ClaimID:    "claim-other",
		ProjectID:  "project-b",
		Repository: "repo-b",
		Agent:      "claude",
		Status:     "starting",
	})
	if ok {
		t.Fatal("second claude reservation during provider startup succeeded")
	}
	if scope != "assignee" || conflict.ClaimID != "claim-startup" {
		t.Fatalf("conflict = %#v scope=%q", conflict, scope)
	}
	if !conflict.IsActive() || !conflict.IsProviderStartup() {
		t.Fatalf("startup session should remain active: %#v", conflict)
	}
}

func TestStoreTryReserveSessionIsAtomic(t *testing.T) {
	app := Compose(ComposeConfig{CoreRoot: t.TempDir(), DryRun: true})
	session := execution.RuntimeSession{
		ClaimID:    "claim-1",
		ProjectID:  "core-eggs-gd",
		Repository: "core.eggs.gd",
		Status:     "starting",
		ClaimedAt:  "2026-08-01T10:00:00+03:00",
	}

	if _, _, ok := app.Exec.TryReserveSession(session); !ok {
		t.Fatal("first reservation failed")
	}
	conflict, scope, ok := app.Exec.TryReserveSession(execution.RuntimeSession{
		ClaimID:    "claim-2",
		ProjectID:  "core-eggs-gd",
		Repository: "core.eggs.gd",
		Status:     "starting",
	})
	if ok {
		t.Fatal("second reservation for same project/repository succeeded")
	}
	if conflict.ClaimID != "claim-1" || scope != "project" {
		t.Fatalf("conflict = %#v scope=%q", conflict, scope)
	}
}

func TestStoreSessionUpsertRecordsActivityTimestamps(t *testing.T) {
	app := Compose(ComposeConfig{CoreRoot: t.TempDir(), DryRun: true})
	app.Exec.UpsertSession(execution.RuntimeSession{
		ClaimID:         "claim-activity",
		Status:          "running",
		ExecutionStatus: "running",
		LastEvent:       "started",
	})
	sessions := app.Exec.Status().RuntimeSessions
	if len(sessions) != 1 {
		t.Fatalf("sessions = %#v, want 1", sessions)
	}
	first := sessions[0]
	if first.LastSeenAt == "" || first.LastEventAt == "" || first.LastStatusAt == "" {
		t.Fatalf("initial session missing activity timestamps: %#v", first)
	}

	time.Sleep(time.Millisecond)
	first.LastMessage = "output"
	app.Exec.UpsertSession(first)
	second := app.Exec.Status().RuntimeSessions[0]
	if second.LastOutputAt == "" {
		t.Fatalf("session missing last_output_at after output: %#v", second)
	}
	if second.LastEventAt != first.LastEventAt {
		t.Fatalf("last_event_at changed without event change: before=%q after=%q", first.LastEventAt, second.LastEventAt)
	}
}

func TestExecutionStateExposesCodexRemoteIdentity(t *testing.T) {
	session := execution.RuntimeSession{
		ClaimID:  "core-80-1",
		Agent:    "codex",
		Backend:  codexAppServerBackend,
		HostID:   "host-1",
		HostName: "core-host",
		CodexSessionDetails: execution.CodexSessionDetails{
			CodexThreadID:    "thread-80",
			CodexTurnID:      "turn-80",
			CodexThreadTitle: "CORE-80 · Set meaningful Codex Remote thread titles",
		},
		Status:          "running",
		ExecutionStatus: "running",
	}

	state := execution.ExecutionStateForSession(session)

	if state.HostID != "host-1" || state.HostName != "core-host" {
		t.Fatalf("host identity = %q/%q", state.HostID, state.HostName)
	}
	if state.ThreadID != "thread-80" || state.TurnID != "turn-80" {
		t.Fatalf("thread/turn = %q/%q", state.ThreadID, state.TurnID)
	}
	if state.ThreadTitle != "CORE-80 · Set meaningful Codex Remote thread titles" {
		t.Fatalf("thread title = %q", state.ThreadTitle)
	}
}

func TestRuntimeBootstrapDetectsDoingTaskAsOrphan(t *testing.T) {
	root := t.TempDir()
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
	writeTestFile(t, filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-a.md"), testTaskMarkdown("CORE-46", "Orphan", "doing"))

	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}

	state := app.State()
	if len(state.OrphanedTasks) != 1 {
		t.Fatalf("orphans = %#v, want one", state.OrphanedTasks)
	}
	if state.OrphanedTasks[0].TaskRef != "CORE-46" {
		t.Fatalf("orphan ref = %q, want CORE-46", state.OrphanedTasks[0].TaskRef)
	}
}

func TestRuntimeBootstrapRehydratesPersistedActiveSession(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-01-core-64.md")
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
	writeTestFile(t, taskPath, testTaskMarkdownWithID("work-core-64", "CORE-64", "Restarted session", "doing"))
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := execution.PersistRuntimeSession(root, execution.RuntimeSession{
		ClaimID:    "core-64-claim",
		TaskRef:    task.Ref,
		TaskID:     task.ID,
		TaskPath:   task.RelativePath,
		ProjectID:  task.ProjectID,
		Repository: "core.eggs.gd",
		Agent:      "codex",
		Launcher:   "core",
		Backend:    codexAppServerBackend,
		LogPath:    "_registry/sessions/core-64-claim.log",
		ProcessID:  os.Getpid(),
		CodexSessionDetails: execution.CodexSessionDetails{
			CodexThreadID: "thread-1",
			CodexTurnID:   "turn-1",
		},
		ClaimedAt:       "2026-08-01T23:50:00+03:00",
		StartedAt:       "2026-08-01T23:51:00+03:00",
		Status:          "running",
		ExecutionStatus: "running",
	}); err != nil {
		t.Fatal(err)
	}

	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}

	state := app.State()
	if len(state.OrphanedTasks) != 0 {
		t.Fatalf("orphans = %#v, want none while persisted session is resumable", state.OrphanedTasks)
	}
	if len(state.RuntimeSessions) != 1 {
		t.Fatalf("runtime sessions = %#v, want one rehydrated session", state.RuntimeSessions)
	}
	session := state.RuntimeSessions[0]
	if session.ClaimID != "core-64-claim" || session.ExecutionStatus != "resumable" {
		t.Fatalf("session = %#v, want recovered resumable core-64-claim", session)
	}
	if session.CodexThreadID != "thread-1" || session.LogPath == "" || session.Backend != codexAppServerBackend {
		t.Fatalf("session metadata not preserved: %#v", session)
	}
	if task := state.Tasks[0]; task.Status != "doing" || task.Execution.State != "resumable" {
		t.Fatalf("task execution = status %q %#v, want doing + resumable", task.Status, task.Execution)
	}
}

func TestRuntimeBootstrapMarksDoingTaskWithDeadCodexSessionAsExecutionDead(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-69.md")
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
	writeTestFile(t, taskPath, testTaskMarkdownWithID("work-core-69", "CORE-69", "Dead Codex session", "doing"))
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := execution.PersistRuntimeSession(root, execution.RuntimeSession{
		ClaimID:    "core-69-1785618118035062000",
		TaskRef:    task.Ref,
		TaskID:     task.ID,
		TaskPath:   task.RelativePath,
		ProjectID:  task.ProjectID,
		Repository: "core.eggs.gd",
		Agent:      "codex",
		Backend:    codexAppServerBackend,
		LogPath:    "_registry/sessions/core-69-1785618118035062000.log",
		ProcessID:  999999999,
		CodexSessionDetails: execution.CodexSessionDetails{
			CodexThreadID: "019fbf22-35f7-72e2-848f-e889d5ae6e10",
			CodexTurnID:   "turn-1",
		},
		ClaimedAt:       "2026-08-02T00:00:00+03:00",
		StartedAt:       "2026-08-02T00:01:00+03:00",
		LastSeenAt:      "2026-08-02T00:02:00+03:00",
		Status:          "running",
		ExecutionStatus: "running",
	}); err != nil {
		t.Fatal(err)
	}

	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}

	state := app.State()
	if len(state.RuntimeSessions) != 0 {
		t.Fatalf("runtime sessions = %#v, want no active session for dead process", state.RuntimeSessions)
	}
	if len(state.OrphanedTasks) != 1 {
		t.Fatalf("orphans = %#v, want one resumable orphan", state.OrphanedTasks)
	}
	orphan := state.OrphanedTasks[0]
	// CORE-74: dead local process with a known Codex thread id is orphaned-but-resumable,
	// not a live session and not an opaque unresolved orphan.
	if orphan.ExecutionState != "resumable" || orphan.ClaimID != "core-69-1785618118035062000" {
		t.Fatalf("orphan = %#v, want resumable orphan with original claim id", orphan)
	}
	if orphan.ThreadID != "019fbf22-35f7-72e2-848f-e889d5ae6e10" || orphan.LogPath == "" || orphan.ProcessID != 999999999 {
		t.Fatalf("orphan identity not preserved: %#v", orphan)
	}
	if orphan.Capabilities.CanResumeSession != capabilityYes || orphan.Capabilities.CanDetectRunningSession != capabilityYes {
		t.Fatalf("orphan capabilities = %#v, want explicit Codex recovery contract", orphan.Capabilities)
	}
	if !strings.Contains(orphan.BlockingReason, "blocks later codex launches") {
		t.Fatalf("blocking reason = %q, want launch-slot explanation", orphan.BlockingReason)
	}
	if !strings.Contains(orphan.Reason, "orphaned but resumable") {
		t.Fatalf("reason = %q, want orphaned-but-resumable classification", orphan.Reason)
	}
	if len(state.Tasks) != 1 {
		t.Fatalf("tasks = %#v, want one task", state.Tasks)
	}
	apiTask := state.Tasks[0]
	if apiTask.Status != "doing" {
		t.Fatalf("task status = %q, want canonical doing unchanged", apiTask.Status)
	}
	if apiTask.Execution.State != "resumable" || apiTask.Execution.ClaimID != orphan.ClaimID || apiTask.Execution.ThreadID != orphan.ThreadID {
		t.Fatalf("task execution = %#v, want resumable orphan execution identity", apiTask.Execution)
	}
}

func TestRuntimeBootstrapClosesLeftoverSessionAfterTaskLeftDoing(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-79.md")
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
	writeTestFile(t, taskPath, testTaskMarkdownWithID("work-core-79", "CORE-79", "Leftover Codex session", "archived"))
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := execution.PersistRuntimeSession(root, execution.RuntimeSession{
		ClaimID:    "core-79-leftover",
		TaskRef:    task.Ref,
		TaskID:     task.ID,
		TaskPath:   task.RelativePath,
		ProjectID:  task.ProjectID,
		Repository: "core.eggs.gd",
		Agent:      "codex",
		Backend:    codexAppServerBackend,
		LogPath:    "_registry/sessions/core-79-leftover.log",
		ProcessID:  999999997,
		CodexSessionDetails: execution.CodexSessionDetails{
			CodexThreadID: "019fc15b-affd-7a01-9419-88f56d6b1a0e",
			CodexTurnID:   "turn-leftover",
		},
		ClaimedAt:       "2026-08-02T10:23:59+03:00",
		StartedAt:       "2026-08-02T10:23:59+03:00",
		Status:          "running",
		ExecutionStatus: "running",
	}); err != nil {
		t.Fatal(err)
	}

	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	state := app.State()
	if len(state.RuntimeSessions) != 0 {
		t.Fatalf("runtime sessions = %#v, want leftover off the live strip", state.RuntimeSessions)
	}
	if len(state.OrphanedTasks) != 0 {
		t.Fatalf("orphans = %#v, want none after the task left doing", state.OrphanedTasks)
	}
	if len(state.Tasks) != 1 || state.Tasks[0].Status != "archived" {
		t.Fatalf("task = %#v, want archived task left unchanged", state.Tasks)
	}
	if state.Tasks[0].Execution.State != "none" {
		t.Fatalf("archived task execution = %#v, want none", state.Tasks[0].Execution)
	}
	records, err := execution.LoadRuntimeSessionRecords(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].IsActive() || records[0].ExecutionStatus != "exited" {
		t.Fatalf("persisted session = %#v, want closed so the next restart does not list it", records)
	}
}

func TestRuntimeBootstrapMarksDeadCodexSessionWithoutThreadAsDead(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-74-dead.md")
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
	writeTestFile(t, taskPath, testTaskMarkdownWithID("work-core-74-dead", "CORE-74A", "Dead Codex without thread", "doing"))
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := execution.PersistRuntimeSession(root, execution.RuntimeSession{
		ClaimID:         "core-74-dead-claim",
		TaskRef:         task.Ref,
		TaskID:          task.ID,
		TaskPath:        task.RelativePath,
		ProjectID:       task.ProjectID,
		Repository:      "core.eggs.gd",
		Agent:           "codex",
		Backend:         codexAppServerBackend,
		LogPath:         "_registry/sessions/core-74-dead-claim.log",
		ProcessID:       999999998,
		ClaimedAt:       "2026-08-02T00:00:00+03:00",
		StartedAt:       "2026-08-02T00:01:00+03:00",
		Status:          "running",
		ExecutionStatus: "running",
	}); err != nil {
		t.Fatal(err)
	}

	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	state := app.State()
	if len(state.OrphanedTasks) != 1 {
		t.Fatalf("orphans = %#v, want one dead orphan", state.OrphanedTasks)
	}
	orphan := state.OrphanedTasks[0]
	if orphan.ExecutionState != "dead" || orphan.ClaimID != "core-74-dead-claim" {
		t.Fatalf("orphan = %#v, want dead orphan without resume identity", orphan)
	}
	if orphan.Capabilities.CanResumeSession != capabilityYes {
		t.Fatalf("capabilities should still describe Codex resume support: %#v", orphan.Capabilities)
	}
}

func TestRuntimeBootstrapClassifiesUnknownProviderCapability(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-74-unknown.md")
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
	writeTestFile(t, taskPath, testTaskMarkdownWithID("work-core-74-unknown", "CORE-74B", "Unknown provider session", "doing"))
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := execution.PersistRuntimeSession(root, execution.RuntimeSession{
		ClaimID:         "core-74-unknown-claim",
		TaskRef:         task.Ref,
		TaskID:          task.ID,
		TaskPath:        task.RelativePath,
		ProjectID:       task.ProjectID,
		Repository:      "core.eggs.gd",
		Agent:           "mystery",
		Backend:         "mystery-backend",
		LogPath:         "_registry/sessions/core-74-unknown-claim.log",
		ProcessID:       999999997,
		ClaimedAt:       "2026-08-02T00:00:00+03:00",
		StartedAt:       "2026-08-02T00:01:00+03:00",
		Status:          "running",
		ExecutionStatus: "running",
	}); err != nil {
		t.Fatal(err)
	}

	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	state := app.State()
	if len(state.RuntimeSessions) != 0 {
		t.Fatalf("runtime sessions = %#v, want none for unknown provider", state.RuntimeSessions)
	}
	if len(state.OrphanedTasks) != 1 {
		t.Fatalf("orphans = %#v, want one unknown orphan", state.OrphanedTasks)
	}
	orphan := state.OrphanedTasks[0]
	if orphan.ExecutionState != "unknown" {
		t.Fatalf("orphan = %#v, want unknown recovery state", orphan)
	}
	if orphan.Capabilities.CanDetectRunningSession != capabilityUnknown || orphan.Capabilities.CanResumeSession != capabilityUnknown {
		t.Fatalf("capabilities = %#v, want unknown recovery contract", orphan.Capabilities)
	}
	if _, blocked := app.Exec.ActiveOrphanConflict("mystery", execution.RuntimeSession{Agent: "mystery", TaskRef: "CORE-OTHER"}); !blocked {
		t.Fatal("unknown orphan should still block same-assignee launches")
	}
}

func TestRuntimeBootstrapIgnoresTerminalTaskDespitePersistedActiveSession(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-74-terminal.md")
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
	writeTestFile(t, taskPath, testTaskMarkdownWithID("work-core-74-terminal", "CORE-74C", "Already reviewed", "needs_review"))
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := execution.PersistRuntimeSession(root, execution.RuntimeSession{
		ClaimID:    "core-74-terminal-claim",
		TaskRef:    task.Ref,
		TaskID:     task.ID,
		TaskPath:   task.RelativePath,
		ProjectID:  task.ProjectID,
		Repository: "core.eggs.gd",
		Agent:      "codex",
		Backend:    codexAppServerBackend,
		ProcessID:  os.Getpid(),
		CodexSessionDetails: execution.CodexSessionDetails{
			CodexThreadID: "thread-terminal",
		},
		ClaimedAt:       "2026-08-02T00:00:00+03:00",
		Status:          "running",
		ExecutionStatus: "running",
	}); err != nil {
		t.Fatal(err)
	}

	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	state := app.State()
	if len(state.RuntimeSessions) != 0 {
		t.Fatalf("runtime sessions = %#v, want none for terminal task", state.RuntimeSessions)
	}
	if len(state.OrphanedTasks) != 0 {
		t.Fatalf("orphans = %#v, want none for terminal task", state.OrphanedTasks)
	}
}

func TestRuntimeBootstrapBlocksStaleClaudeBackgroundSessionWithDeadControlSocket(t *testing.T) {
	root := t.TempDir()
	claude := writeExecutable(t, root, "fake-claude-dead-socket", `#!/bin/sh
if [ "$1" = "logs" ]; then
  echo "connect ECONNREFUSED /tmp/claude-daemon.sock" >&2
  exit 1
fi
if [ "$1" = "agents" ]; then
  echo "connect ECONNREFUSED /tmp/claude-daemon.sock" >&2
  exit 1
fi
exit 1
`)
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-82.md")
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
	writeTestFile(t, taskPath, testTaskMarkdownWithID("work-core-82", "CORE-82", "Stale Claude session", "doing"))
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := execution.PersistRuntimeSession(root, execution.RuntimeSession{
		ClaimID:    "core-13-claude",
		TaskRef:    task.Ref,
		TaskID:     task.ID,
		TaskPath:   task.RelativePath,
		ProjectID:  task.ProjectID,
		Repository: "core.eggs.gd",
		Agent:      "claude",
		Backend:    "background-remote",
		Command:    []string{claude},
		LogPath:    "_registry/sessions/core-13-claude.log",
		ClaudeSessionDetails: execution.ClaudeSessionDetails{
			BackgroundID:     "8920ca20",
			RemoteControlURL: "https://claude.ai/code/session_01P9yfovFvSPYVZRDPcoGLjF",
		},
		ClaimedAt:       "2026-08-02T00:00:00+03:00",
		StartedAt:       "2026-08-02T00:01:00+03:00",
		Status:          "running",
		ExecutionStatus: "resumable",
	}); err != nil {
		t.Fatal(err)
	}

	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "blocked" {
		t.Fatalf("task status = %q, want blocked for stale Claude session", loaded.Status)
	}
	if len(loaded.Comments) == 0 {
		t.Fatal("missing stale-session review comment")
	}
	comment := loaded.Comments[0].Text
	for _, expected := range []string{"control_socket_unavailable", "8920ca20", "no longer guarded by a stale resumable session"} {
		if !strings.Contains(comment, expected) {
			t.Fatalf("comment missing %q:\n%s", expected, comment)
		}
	}
	state := app.State()
	if len(state.RuntimeSessions) != 0 {
		t.Fatalf("runtime sessions = %#v, want active strip empty for terminal provider-error session", state.RuntimeSessions)
	}
	session, ok := findSessionInGroups(state, "CORE-82")
	if !ok {
		t.Fatalf("session_groups = %#v, want terminal provider-error session in history", state.SessionGroups)
	}
	if session.IsActive() || session.ExecutionStatus != "control_socket_unavailable" {
		t.Fatalf("session = %#v, want non-active control_socket_unavailable", session)
	}
	if session.ProviderError == nil || session.ProviderError.Kind != "control_socket_unavailable" {
		t.Fatalf("provider error = %#v, want control_socket_unavailable", session.ProviderError)
	}
	if len(state.OrphanedTasks) != 0 {
		t.Fatalf("orphans = %#v, want none after task leaves doing", state.OrphanedTasks)
	}
	apiTask := state.Tasks[0]
	if apiTask.Status != "blocked" || apiTask.Execution.State != "none" {
		t.Fatalf("api task = status %q execution %#v, want blocked/no active execution", apiTask.Status, apiTask.Execution)
	}
}

func TestRuntimeBootstrapRehydratesHealthyClaudeBackgroundSessionAfterProviderVerification(t *testing.T) {
	root := t.TempDir()
	claude := writeExecutable(t, root, "fake-claude-healthy", `#!/bin/sh
if [ "$1" = "logs" ]; then
  echo "Continue here, on your phone, or at https://claude.ai/code/session_HEALTHY"
  exit 0
fi
if [ "$1" = "agents" ]; then
  echo '[{"id":"bghealthy","state":"running"}]'
  exit 0
fi
exit 1
`)
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-82-healthy.md")
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
	writeTestFile(t, taskPath, testTaskMarkdownWithID("work-core-82-healthy", "CORE-82-HEALTHY", "Healthy Claude session", "doing"))
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := execution.PersistRuntimeSession(root, execution.RuntimeSession{
		ClaimID:    "core-82-healthy-claim",
		TaskRef:    task.Ref,
		TaskID:     task.ID,
		TaskPath:   task.RelativePath,
		ProjectID:  task.ProjectID,
		Repository: "core.eggs.gd",
		Agent:      "claude",
		Backend:    "background-remote",
		Command:    []string{claude},
		LogPath:    "_registry/sessions/core-82-healthy-claim.log",
		ClaudeSessionDetails: execution.ClaudeSessionDetails{
			BackgroundID:     "bghealthy",
			RemoteControlURL: "https://claude.ai/code/session_HEALTHY",
		},
		ClaimedAt:       "2026-08-02T00:00:00+03:00",
		StartedAt:       "2026-08-02T00:01:00+03:00",
		Status:          "running",
		ExecutionStatus: "running",
	}); err != nil {
		t.Fatal(err)
	}

	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}

	state := app.State()
	if len(state.OrphanedTasks) != 0 {
		t.Fatalf("orphans = %#v, want none for provider-verified Claude session", state.OrphanedTasks)
	}
	if len(state.RuntimeSessions) != 1 {
		t.Fatalf("runtime sessions = %#v, want rehydrated Claude session", state.RuntimeSessions)
	}
	session := state.RuntimeSessions[0]
	if session.ClaimID != "core-82-healthy-claim" || session.ExecutionStatus != "resumable" || !session.IsActive() {
		t.Fatalf("session = %#v, want active resumable provider-verified Claude session", session)
	}
	if session.BackgroundID != "bghealthy" || session.Capabilities.TerminalStateDetection == "" {
		t.Fatalf("session metadata/capabilities missing: %#v", session)
	}
	if task := state.Tasks[0]; task.Status != "doing" || task.Execution.State != "resumable" {
		t.Fatalf("task execution = status %q %#v, want doing + resumable", task.Status, task.Execution)
	}
}

// TestRuntimeBootstrapRecoversWorkerResultFromTerminalClaudeSessionTranscript
// is the CORE-97 rework regression test: the live Claude runner in
// `internal/execution` already finalizes a task from a worker's reported outcome
// without waiting for the background session to end, but restart
// reconciliation (runtime_bootstrap.go) previously skipped that check
// entirely for a session already found terminal, falling back to a generic
// "stale session" blocked comment even when the transcript it had just
// fetched to verify reachability contained a valid, unread worker outcome
// payload. This is exactly what happened to CORE-97's own first session: it
// reported "completed" but the daemon restarted before the poll loop read it,
// and reconciliation discarded the report instead of finalizing from it.
func TestRuntimeBootstrapRecoversWorkerResultFromTerminalClaudeSessionTranscript(t *testing.T) {
	root := t.TempDir()
	claude := writeExecutable(t, root, "fake-claude-terminal-with-result", `#!/bin/sh
if [ "$1" = "logs" ]; then
  echo "Continue here, on your phone, or at https://claude.ai/code/session_RECOVERED"
  echo "Done working. Also covered the 429 rate-limit failure mode."
  echo '{"outcome":"completed","summary":"Implemented the outcome protocol.","artifacts":["a.go"]}'
  exit 0
fi
if [ "$1" = "agents" ]; then
  echo '[{"id":"bgrecovered","state":"done"}]'
  exit 0
fi
exit 1
`)
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-03-core-97-recovered.md")
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
	writeTestFile(t, taskPath, testTaskMarkdownWithID("work-core-97-recovered", "CORE-97-RECOVERED", "Recover worker result after restart", "doing"))
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := execution.PersistRuntimeSession(root, execution.RuntimeSession{
		ClaimID:    "core-97-recovered-claim",
		TaskRef:    task.Ref,
		TaskID:     task.ID,
		TaskPath:   task.RelativePath,
		ProjectID:  task.ProjectID,
		Repository: "core.eggs.gd",
		Agent:      "claude",
		Backend:    "background-remote",
		Command:    []string{claude},
		LogPath:    "_registry/sessions/core-97-recovered-claim.log",
		ClaudeSessionDetails: execution.ClaudeSessionDetails{
			BackgroundID:     "bgrecovered",
			RemoteControlURL: "https://claude.ai/code/session_RECOVERED",
		},
		ClaimedAt:       "2026-08-03T15:03:00+03:00",
		StartedAt:       "2026-08-03T15:03:05+03:00",
		Status:          "running",
		ExecutionStatus: "resumable",
	}); err != nil {
		t.Fatal(err)
	}

	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "needs_review" {
		t.Fatalf("task status = %q, want needs_review recovered from the transcript's worker outcome, not a generic stale-session block", loaded.Status)
	}
	if len(loaded.Comments) == 0 {
		t.Fatal("missing finalization comment")
	}
	comment := loaded.Comments[len(loaded.Comments)-1].Text
	for _, expected := range []string{"Implemented the outcome protocol.", "a.go"} {
		if !strings.Contains(comment, expected) {
			t.Fatalf("comment missing %q, want it to reflect the worker's reported outcome, not a stale-session comment:\n%s", expected, comment)
		}
	}
	if strings.Contains(comment, "no longer guarded by a stale resumable session") {
		t.Fatalf("comment fell back to the generic stale-session path instead of the recovered worker result:\n%s", comment)
	}
	if strings.Contains(comment, "Provider blocker detected") || strings.Contains(comment, "rate_limited") {
		t.Fatalf("restart recovery misclassified rate-limit prose in a successful transcript:\n%s", comment)
	}
	state := app.State()
	if len(state.OrphanedTasks) != 0 {
		t.Fatalf("orphans = %#v, want none once the task is finalized from the recovered result", state.OrphanedTasks)
	}
	if len(state.RuntimeSessions) != 0 {
		t.Fatalf("runtime sessions = %#v, want no active strip entry after terminal recovery", state.RuntimeSessions)
	}

	records, err := execution.LoadRuntimeSessionRecords(root)
	if err != nil {
		t.Fatal(err)
	}
	var recovered execution.RuntimeSession
	for _, record := range records {
		if record.ClaimID == "core-97-recovered-claim" {
			recovered = record
			break
		}
	}
	if recovered.ClaimID == "" {
		t.Fatal("expected recovered session registry record")
	}
	if recovered.Result == nil || recovered.Result.Outcome != "completed" {
		t.Fatalf("persisted session.Result = %#v, want completed worker result on disk", recovered.Result)
	}
	if recovered.Status != "exited" || recovered.ExecutionStatus != "succeeded" {
		t.Fatalf("persisted session status = %s/%s, want exited/succeeded", recovered.Status, recovered.ExecutionStatus)
	}
}

func TestCursorVisibleRuntimeSessionCreatesChatAndLaunchesWithoutPrint(t *testing.T) {
	root := t.TempDir()
	binDir := t.TempDir()
	cursorPath := filepath.Join(binDir, "cursor-agent")
	script := `#!/bin/sh
if [ "$1" = "create-chat" ]; then
  echo chat_CORE_79_123
  exit 0
fi
printf '%s\n' "$@" > cursor-visible-argv.txt
echo CORE_CURSOR_VISIBLE_SMOKE_OK
exit 0
`
	if err := os.WriteFile(cursorPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-cursor-visible.md")
	writeTestFile(t, taskPath, testTaskMarkdownWithID("work-core-79-smoke", "CORE-79-SMOKE", "Cursor visible smoke", "todo"))
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "cursor",
		Backend:    cursorVisibleBackend,
		Command:    []string{cursorPath, "--trust", "--workspace", root, "write smoke marker"},
		WorkingDir: root,
		Repository: "core.eggs.gd",
		Launchable: true,
		Outcome:    "launchable",
	}

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	app.Exec.StartLaunchCandidate(ctx, task)

	waitForLaunchToFinish(t, app, root, taskPath, 5*time.Second)
	if sessions := app.State().RuntimeSessions; len(sessions) != 0 {
		t.Fatalf("runtime still has active sessions: %#v", sessions)
	}

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "needs_review" {
		t.Fatalf("task status = %q, want needs_review; comments=%#v", loaded.Status, loaded.Comments)
	}
	argv, err := os.ReadFile(filepath.Join(root, "cursor-visible-argv.txt"))
	if err != nil {
		t.Fatal(err)
	}
	argvText := string(argv)
	if strings.Contains(argvText, "--print") || strings.Contains(argvText, "--output-format") {
		t.Fatalf("cursor visible launch used print/headless flags: %s", argvText)
	}
	if !strings.Contains(argvText, "--resume\nchat_CORE_79_123") {
		t.Fatalf("cursor visible launch did not resume created chat id: %s", argvText)
	}

	records, err := execution.LoadRuntimeSessionRecords(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("session records = %#v, want one", records)
	}
	record := records[0]
	if record.Backend != cursorVisibleBackend || record.CursorChatID != "chat_CORE_79_123" || record.VisibilityMode != "cli_visible" {
		t.Fatalf("cursor session metadata not persisted: %#v", record)
	}
	if record.OperatorCommand == "" || !strings.Contains(record.OperatorCommand, "--resume chat_CORE_79_123") {
		t.Fatalf("operator command not persisted: %#v", record)
	}
	if !record.Capabilities.CanStartVisibleSession || record.Capabilities.CanResumeSession != capabilityYes {
		t.Fatalf("cursor capabilities not persisted: %#v", record.Capabilities)
	}
	logText := execution.ReadSessionLog(root, record.LogPath)
	if !strings.Contains(logText, "CORE_CURSOR_VISIBLE_SMOKE_OK") || !strings.Contains(logText, "operator resume command") {
		t.Fatalf("cursor session log missing expected markers:\n%s", logText)
	}
}

// TestCursorVisibleRuntimeSessionAppliesWorkerReportedBlockedOutcome is the
// CORE-97 regression test for Cursor: before this fix, cursor-agent's
// captured stdout was never parsed for the worker's compact JSON result
// payload, so any clean process exit (including one where the worker
// reported "blocked") was finalized as an implicit "completed" and the task
// moved to needs_review regardless of what the worker actually said.
func TestCursorVisibleRuntimeSessionAppliesWorkerReportedBlockedOutcome(t *testing.T) {
	root := t.TempDir()
	binDir := t.TempDir()
	cursorPath := filepath.Join(binDir, "cursor-agent")
	script := `#!/bin/sh
if [ "$1" = "create-chat" ]; then
  echo chat_CORE_97_BLOCKED
  exit 0
fi
echo "I could not find the credentials I need."
echo '{"outcome":"blocked","summary":"Need staging credentials.","blockers":["missing CORE_STAGING_TOKEN"]}'
exit 0
`
	if err := os.WriteFile(cursorPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-03-cursor-blocked.md")
	writeTestFile(t, taskPath, testTaskMarkdownWithID("work-core-97-cursor-blocked", "CORE-97-CURSOR", "Cursor reports blocked outcome", "todo"))
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "cursor",
		Backend:    cursorVisibleBackend,
		Command:    []string{cursorPath, "--trust", "--workspace", root, "do the task"},
		WorkingDir: root,
		Repository: "core.eggs.gd",
		Launchable: true,
		Outcome:    "launchable",
	}

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	app.Exec.StartLaunchCandidate(ctx, task)

	waitForLaunchToFinish(t, app, root, taskPath, 5*time.Second)
	if sessions := app.State().RuntimeSessions; len(sessions) != 0 {
		t.Fatalf("runtime still has active sessions: %#v", sessions)
	}

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "blocked" {
		t.Fatalf("task status = %q, want blocked; comments=%#v", loaded.Status, loaded.Comments)
	}
	if len(loaded.Comments) == 0 || !strings.Contains(loaded.Comments[len(loaded.Comments)-1].Text, "missing CORE_STAGING_TOKEN") {
		t.Fatalf("comments = %#v, want the worker's reported blocker", loaded.Comments)
	}
}

func TestRuntimeBootstrapIgnoresTerminalSessionForReviewedTask(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-01-core-64.md")
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
	writeTestFile(t, taskPath, testTaskMarkdownWithID("work-core-64", "CORE-64", "Reviewed session", "needs_review"))
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := execution.PersistRuntimeSession(root, execution.RuntimeSession{
		ClaimID:         "core-64-claim",
		TaskRef:         task.Ref,
		TaskID:          task.ID,
		TaskPath:        task.RelativePath,
		ProjectID:       task.ProjectID,
		Repository:      "core.eggs.gd",
		Agent:           "codex",
		Backend:         codexAppServerBackend,
		LogPath:         "_registry/sessions/core-64-claim.log",
		ClaimedAt:       "2026-08-01T23:50:00+03:00",
		StartedAt:       "2026-08-01T23:51:00+03:00",
		ExitedAt:        "2026-08-02T00:05:00+03:00",
		Status:          "exited",
		ExecutionStatus: "succeeded",
		Result:          &tasklifecycle.WorkerResult{Outcome: "completed", Summary: "done"},
	}); err != nil {
		t.Fatal(err)
	}

	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}

	state := app.State()
	if len(state.RuntimeSessions) != 0 {
		t.Fatalf("runtime sessions = %#v, want terminal persisted session ignored", state.RuntimeSessions)
	}
	if len(state.OrphanedTasks) != 0 {
		t.Fatalf("orphans = %#v, want reviewed task ignored", state.OrphanedTasks)
	}
}

func TestRuntimeStartupOrphanBlocksSameAssigneeLaunchSlot(t *testing.T) {
	root := t.TempDir()
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
	orphanPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-01-core-64.md")
	candidatePath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-01-core-65.md")
	writeTestFile(t, orphanPath, testTaskMarkdownWithID("work-core-64", "CORE-64", "Orphaned Codex task", "doing"))
	writeTestFile(t, candidatePath, testTaskMarkdownWithID("work-core-65", "CORE-65", "Next Codex task", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}

	candidate := launchCandidateForTest(t, root, candidatePath)
	candidate.LaunchEvaluation.Agent = "codex"
	app.Exec.StartLaunchCandidate(t.Context(), candidate)

	loaded, err := markdown.LoadTaskFile(root, candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "todo" {
		t.Fatalf("CORE-65 status = %q, want todo while CORE-64 orphan blocks Codex", loaded.Status)
	}
	if sessions := app.State().RuntimeSessions; len(sessions) != 0 {
		t.Fatalf("sessions = %#v, want no launch while orphan blocks", sessions)
	}
	stateTask, ok := app.Store.TaskByPath(candidate.Path)
	if !ok {
		t.Fatal("CORE-65 missing from runtime store")
	}
	wait := stateTask.LaunchEvaluation.Waiting
	if wait == nil {
		t.Fatalf("CORE-65 launch evaluation = %#v, want waiting on orphan", stateTask.LaunchEvaluation)
	}
	if wait.Scope != "assignee_orphan" || wait.Resource != "codex" {
		t.Fatalf("wait = %#v, want assignee_orphan/codex", wait)
	}
	if !strings.Contains(wait.Reason, "unresolved orphaned doing task CORE-64") {
		t.Fatalf("wait reason = %q, want CORE-64 orphan reason", wait.Reason)
	}
}

func TestResolvingStartupOrphanReleasesAssigneeLaunchSlot(t *testing.T) {
	root := t.TempDir()
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
	orphanPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-01-core-64.md")
	candidatePath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-01-core-65.md")
	writeTestFile(t, orphanPath, testTaskMarkdownWithID("work-core-64", "CORE-64", "Orphaned Codex task", "doing"))
	writeTestFile(t, candidatePath, testTaskMarkdownWithID("work-core-65", "CORE-65", "Next Codex task", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if _, err := app.PatchTask(tasklifecycle.TaskPatch{
		Path:   orphanPath,
		Status: "blocked",
	}); err != nil {
		t.Fatal(err)
	}

	candidate := launchCandidateForTest(t, root, candidatePath)
	candidate.LaunchEvaluation.Agent = "codex"
	candidate.LaunchEvaluation.Command = []string{"/bin/sh", "-c", "echo launched"}
	app.Exec.StartLaunchCandidate(t.Context(), candidate)
	waitForLaunchToFinish(t, app, root, candidatePath, 15*time.Second)

	loaded, err := markdown.LoadTaskFile(root, candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "needs_review" {
		t.Fatalf("CORE-65 status = %q, want needs_review after orphan resolution", loaded.Status)
	}
	if len(app.State().OrphanedTasks) != 0 {
		t.Fatalf("orphans after resolution = %#v, want none", app.State().OrphanedTasks)
	}
}

func TestStartupOrphanDoesNotBlockDifferentAssignee(t *testing.T) {
	app := Compose(ComposeConfig{CoreRoot: t.TempDir(), DryRun: true})
	task := tasklifecycle.Task{
		Ref:          "CORE-64",
		ID:           "work-core-64",
		Status:       "doing",
		RelativePath: "Work/core-eggs-gd/tasks/2026-08-01-core-64.md",
		ProjectID:    "core-eggs-gd",
		Repositories: []string{"core.eggs.gd"},
		Assignee:     "codex",
	}
	app.Store.UpsertTask(task)
	app.Exec.UpsertOrphan(execution.OrphanedTask{
		TaskRef:    task.Ref,
		TaskID:     task.ID,
		TaskPath:   task.RelativePath,
		ProjectID:  task.ProjectID,
		Repository: "core.eggs.gd",
		Assignee:   task.Assignee,
		DetectedAt: "2026-08-01T23:00:00+03:00",
		Reason:     "task is doing but runtime has no active session after startup",
	})

	if _, _, ok := app.Exec.TryReserveSession(execution.RuntimeSession{
		ClaimID:    "core-65-claude",
		TaskRef:    "CORE-65",
		TaskID:     "work-core-65",
		TaskPath:   "Work/core-eggs-gd/tasks/2026-08-01-core-65.md",
		ProjectID:  "core-eggs-gd",
		Repository: "core.eggs.gd",
		Agent:      "claude",
		Status:     "starting",
	}); !ok {
		t.Fatal("Codex orphan blocked unrelated Claude assignee")
	}
}

func TestRuntimeClearsOrphanWhenTaskMovesToNeedsReviewThenDone(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-a.md")
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
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-55", "Unified serve command", "doing"))

	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if len(app.State().OrphanedTasks) != 1 {
		t.Fatalf("orphans after bootstrap = %#v, want one", app.State().OrphanedTasks)
	}

	needsReview, err := app.PatchTask(tasklifecycle.TaskPatch{
		Path:   taskPath,
		Status: "needs_review",
	})
	if err != nil {
		t.Fatal(err)
	}
	if needsReview.Status != "needs_review" {
		t.Fatalf("status = %q, want needs_review", needsReview.Status)
	}
	if len(app.State().OrphanedTasks) != 0 {
		t.Fatalf("orphans after needs_review = %#v, want none", app.State().OrphanedTasks)
	}

	done, err := app.PatchTask(tasklifecycle.TaskPatch{
		Path:   taskPath,
		Status: "done",
	})
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != "done" {
		t.Fatalf("status = %q, want done", done.Status)
	}
	if len(app.State().OrphanedTasks) != 0 {
		t.Fatalf("orphans after done = %#v, want none", app.State().OrphanedTasks)
	}

	data, err := audit.EventLogText(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(data, `"type":"runtime_orphan_detected"`) {
		t.Fatalf("event log lost historical orphan event:\n%s", data)
	}
}

func TestStoreClearsOrphanWhenTaskIsReclaimedIntoActiveSession(t *testing.T) {
	app := Compose(ComposeConfig{CoreRoot: t.TempDir(), DryRun: true})
	task := tasklifecycle.Task{
		Ref:          "CORE-55",
		ID:           "work-core-55",
		Status:       "doing",
		RelativePath: "Work/core-eggs-gd/tasks/2026-08-01-core-55.md",
		ProjectID:    "core-eggs-gd",
		Repositories: []string{"core.eggs.gd"},
		Assignee:     "claude",
	}
	app.Store.UpsertTask(task)
	app.Exec.UpsertOrphan(execution.OrphanedTask{
		TaskRef:    task.Ref,
		TaskID:     task.ID,
		TaskPath:   task.RelativePath,
		ProjectID:  task.ProjectID,
		Repository: "core.eggs.gd",
		Assignee:   task.Assignee,
		DetectedAt: "2026-08-01T20:16:04+03:00",
		Reason:     "task is doing but runtime has no active session",
	})
	if len(app.Exec.Status().OrphanedTasks) != 1 {
		t.Fatalf("orphans before reclaim = %#v, want one", app.Exec.Status().OrphanedTasks)
	}

	if _, _, ok := app.Exec.TryReserveSession(execution.RuntimeSession{
		ClaimID:    "core-55-1785604604465932000",
		TaskRef:    task.Ref,
		TaskID:     task.ID,
		TaskPath:   task.RelativePath,
		ProjectID:  task.ProjectID,
		Repository: "core.eggs.gd",
		Agent:      task.Assignee,
		Status:     "starting",
	}); !ok {
		t.Fatal("session reservation failed")
	}
	if len(app.Exec.Status().OrphanedTasks) != 0 {
		t.Fatalf("orphans after active session = %#v, want none", app.Exec.Status().OrphanedTasks)
	}
}

func TestStoreSnapshotExposesOnlyCurrentDoingOrphans(t *testing.T) {
	app := Compose(ComposeConfig{CoreRoot: t.TempDir(), DryRun: true})
	for _, status := range []string{"needs_review", "done"} {
		task := tasklifecycle.Task{
			Ref:          "CORE-" + status,
			ID:           "work-" + status,
			Status:       status,
			RelativePath: "Work/core-eggs-gd/tasks/" + status + ".md",
			ProjectID:    "core-eggs-gd",
			Assignee:     "codex",
		}
		app.Store.UpsertTask(task)
		app.Exec.UpsertOrphan(execution.OrphanedTask{
			TaskRef:  task.Ref,
			TaskID:   task.ID,
			TaskPath: task.RelativePath,
			Assignee: task.Assignee,
		})
	}
	if len(app.Exec.Status().OrphanedTasks) != 0 {
		t.Fatalf("stale completed orphans exposed = %#v, want none", app.Exec.Status().OrphanedTasks)
	}
}

func TestConcurrentLaunchCandidatesReserveSameRepositoryOnce(t *testing.T) {
	root := t.TempDir()
	firstPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-a.md")
	secondPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-b.md")
	writeTestFile(t, firstPath, testTaskMarkdownWithID("work-core-50", "CORE-50", "Launch first", "todo"))
	writeTestFile(t, secondPath, testTaskMarkdownWithID("work-core-51", "CORE-51", "Launch second", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false})
	first := launchCandidateForTest(t, root, firstPath)
	second := launchCandidateForTest(t, root, secondPath)
	first.LaunchEvaluation.Command = []string{"/bin/sleep", "5"}
	second.LaunchEvaluation.Command = []string{"/bin/sleep", "5"}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		app.Exec.StartLaunchCandidate(ctx, first)
	}()
	go func() {
		defer wg.Done()
		app.Exec.StartLaunchCandidate(ctx, second)
	}()
	wg.Wait()

	firstLoaded, err := markdown.LoadTaskFile(root, firstPath)
	if err != nil {
		t.Fatal(err)
	}
	secondLoaded, err := markdown.LoadTaskFile(root, secondPath)
	if err != nil {
		t.Fatal(err)
	}
	doing := 0
	todo := 0
	for _, task := range []tasklifecycle.Task{firstLoaded, secondLoaded} {
		switch task.Status {
		case "doing":
			doing++
		case "todo":
			todo++
		default:
			t.Fatalf("unexpected status after concurrent launch: %s", task.Status)
		}
	}
	if doing != 1 || todo != 1 {
		t.Fatalf("statuses after concurrent launch: doing=%d todo=%d", doing, todo)
	}
	state := waitForRuntimeState(t, app, func(state State) bool {
		active := 0
		for _, session := range state.RuntimeSessions {
			if session.IsActive() {
				active++
			}
		}
		waiting := 0
		for _, task := range state.Tasks {
			if task.LaunchEvaluation.Waiting != nil {
				waiting++
			}
		}
		return active == 1 && waiting == 1
	})
	if sessions := state.RuntimeSessions; len(sessions) != 1 {
		t.Fatalf("sessions = %#v, want exactly one active session", sessions)
	}
	waiting := 0
	for _, task := range state.Tasks {
		if task.LaunchEvaluation.Waiting != nil {
			waiting++
		}
	}
	if waiting != 1 {
		t.Fatalf("waiting launch tasks = %d, want 1", waiting)
	}

	cancel()
	waitForNoRuntimeSessions(t, app)
}

func TestRuntimeRequeueCandidatesMatchReleasedSlotScopes(t *testing.T) {
	root := t.TempDir()
	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false})
	task := tasklifecycle.Task{
		Ref:          "CORE-51",
		ID:           "work-test",
		Status:       "todo",
		ProjectID:    "core-eggs-gd",
		Repositories: []string{"core.eggs.gd"},
		Assignee:     "codex",
	}
	task.MarkLaunchWaiting(tasklifecycle.LaunchWait{
		Scope:           "project",
		Resource:        "core-eggs-gd",
		ConflictClaimID: "claim-1",
		Reason:          "active runtime session claim-1 already owns project",
		WaitingSince:    "2026-08-01T10:00:00+03:00",
	})
	app.Store.UpsertTask(task)

	candidates := app.Exec.RequeueCandidatesForReleasedSlot(execution.RuntimeSession{
		ClaimID:    "claim-1",
		ProjectID:  "core-eggs-gd",
		Repository: "core.eggs.gd",
	})

	if len(candidates) != 1 || candidates[0].Ref != "CORE-51" {
		t.Fatalf("candidates = %#v, want CORE-51", candidates)
	}
}

func TestRuntimeLaunchesNextCodexTaskAfterTerminalSuccessWithoutFileEvent(t *testing.T) {
	root, repo := writeRuntimeRequeueFixture(t)
	installFakeCodexAppServer(t, root)
	firstPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-57.md")
	secondPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-80.md")
	writeTestFile(t, firstPath, testTaskMarkdownWithID("work-core-57", "CORE-57", "First Codex task", "todo"))
	writeTestFile(t, secondPath, testTaskMarkdownWithID("work-core-80", "CORE-80", "Second Codex task", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	first := launchCandidateForTest(t, root, firstPath)
	first.LaunchEvaluation.Agent = "codex"
	first.LaunchEvaluation.WorkingDir = repo
	first.LaunchEvaluation.Command = []string{"/bin/sh", "-c", "echo first done"}

	app.Exec.StartLaunchCandidate(t.Context(), first)
	waitForTaskStatus(t, root, secondPath, "needs_review")

	assertEventLogContains(t, root, `"type":"runtime_slot_released"`)
	assertEventLogContains(t, root, `"type":"launch_requeue_pending"`)
	assertEventLogContains(t, root, `"type":"launch_requeue_selected"`)
}

func TestRuntimeLaunchesNextCodexTaskAfterTerminalFailureWithoutFileEvent(t *testing.T) {
	root, repo := writeRuntimeRequeueFixture(t)
	installFakeCodexAppServer(t, root)
	firstPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-57.md")
	secondPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-74.md")
	writeTestFile(t, firstPath, testTaskMarkdownWithID("work-core-57", "CORE-57", "Failing Codex task", "todo"))
	writeTestFile(t, secondPath, testTaskMarkdownWithID("work-core-74", "CORE-74", "Next Codex task", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	first := launchCandidateForTest(t, root, firstPath)
	first.LaunchEvaluation.Agent = "codex"
	first.LaunchEvaluation.WorkingDir = repo
	first.LaunchEvaluation.Command = []string{"/bin/sh", "-c", "echo first failed; exit 1"}

	app.Exec.StartLaunchCandidate(t.Context(), first)
	waitForTaskStatus(t, root, firstPath, "blocked")
	waitForTaskStatus(t, root, secondPath, "needs_review")

	assertEventLogContains(t, root, `"type":"runtime_slot_released"`)
	assertEventLogContains(t, root, `"type":"launch_requeue_selected"`)
}

func TestRuntimeLaunchesNextCodexTaskAfterStartupOrphanRelease(t *testing.T) {
	root, _ := writeRuntimeRequeueFixture(t)
	installFakeCodexAppServer(t, root)
	orphanPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-57.md")
	nextPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-core-70.md")
	writeTestFile(t, orphanPath, testTaskMarkdownWithID("work-core-57", "CORE-57", "Dead Codex task", "doing"))
	writeTestFile(t, nextPath, testTaskMarkdownWithID("work-core-70", "CORE-70", "Next after orphan", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	if len(app.State().OrphanedTasks) != 1 {
		t.Fatalf("orphans = %#v, want one startup orphan", app.State().OrphanedTasks)
	}
	if _, err := app.PatchTask(tasklifecycle.TaskPatch{
		Path:   orphanPath,
		Status: "blocked",
	}); err != nil {
		t.Fatal(err)
	}

	waitForTaskStatus(t, root, nextPath, "needs_review")
	assertEventLogContains(t, root, `"type":"runtime_slot_released"`)
	assertEventLogContains(t, root, `"orphan_released"`)
}

// TestRuntimeProviderSessionExitRequeuesWaitingLaunchableTask proves CORE-127:
// Cursor/Codex/Claude finish through CompleteProviderSession, which must wake
// waiting launchable tasks without requiring an unrelated task file event.
func TestRuntimeProviderSessionExitRequeuesWaitingLaunchableTask(t *testing.T) {
	root, repo := writeRuntimeRequeueFixture(t)
	installFakeCodexAppServer(t, root)
	firstPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-core-123.md")
	secondPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-core-125.md")
	writeTestFile(t, firstPath, testTaskMarkdownWithID("work-core-123", "CORE-123", "Provider session holder", "todo"))
	writeTestFile(t, secondPath, testTaskMarkdownWithPriority("work-core-125", "CORE-125", "Needs rework after slot free", "needs_rework", 1))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}

	first := launchCandidateForTest(t, root, firstPath)
	first.LaunchEvaluation.Agent = "codex"
	first.LaunchEvaluation.Backend = execution.BackendCodexAppServer
	first.LaunchEvaluation.Repository = "core.eggs.gd"
	first.LaunchEvaluation.WorkingDir = repo
	first.LaunchEvaluation.Command = []string{"codex", "app-server", "--stdio"}

	app.Exec.StartLaunchCandidate(t.Context(), first)
	waitForTaskStatus(t, root, secondPath, "needs_review")

	assertEventLogContains(t, root, `"type":"runtime_slot_released"`)
	assertEventLogContains(t, root, `"type":"launch_requeue_selected"`)
	assertEventLogContains(t, root, `"task_ref":"CORE-125"`)
}

// TestRuntimeBootstrapLaunchPendingPicksTodoAndNeedsRework proves existing
// pickup-status tasks are claimed after restart without a new task event.
func TestRuntimeBootstrapLaunchPendingPicksTodoAndNeedsRework(t *testing.T) {
	root, _ := writeRuntimeRequeueFixture(t)
	installFakeCodexAppServer(t, root)
	reworkPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-core-125-boot.md")
	todoPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-core-124-boot.md")
	writeTestFile(t, reworkPath, testTaskMarkdownWithPriority("work-core-125-boot", "CORE-125", "Rework first", "needs_rework", 1))
	writeTestFile(t, todoPath, testTaskMarkdownWithPriority("work-core-124-boot", "CORE-124", "Todo second", "todo", 2))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	app.Exec.LaunchPendingTasks(t.Context())

	waitForTaskStatus(t, root, reworkPath, "needs_review")
	assertEventLogContains(t, root, `"type":"launch_queued"`)
	assertEventLogContains(t, root, `"type":"launch_claimed"`)
	assertEventLogContains(t, root, `"task_ref":"CORE-125"`)

	// After CORE-125 finishes, slot release should pick CORE-124 without a file event.
	waitForTaskStatus(t, root, todoPath, "needs_review")
	assertEventLogContains(t, root, `"task_ref":"CORE-124"`)
}

// TestRuntimeRequeuePrefersNeedsReworkOverTodo covers CORE-125-style ordering.
func TestRuntimeRequeuePrefersNeedsReworkOverTodo(t *testing.T) {
	root, _ := writeRuntimeRequeueFixture(t)
	app := Compose(ComposeConfig{CoreRoot: root, DryRun: true})
	todo := tasklifecycle.Task{
		Ref:          "CORE-124",
		ID:           "work-core-124",
		Status:       "todo",
		Priority:     2,
		ProjectID:    "core-eggs-gd",
		Repositories: []string{"core.eggs.gd"},
		Assignee:     "codex",
		RelativePath: "Work/core-eggs-gd/tasks/core-124.md",
	}
	rework := tasklifecycle.Task{
		Ref:          "CORE-125",
		ID:           "work-core-125",
		Status:       "needs_rework",
		Priority:     1,
		ProjectID:    "core-eggs-gd",
		Repositories: []string{"core.eggs.gd"},
		Assignee:     "codex",
		RelativePath: "Work/core-eggs-gd/tasks/core-125.md",
	}
	todo.MarkLaunchWaiting(tasklifecycle.LaunchWait{
		Scope:           "project",
		Resource:        "core-eggs-gd",
		ConflictClaimID: "claim-holder",
		Reason:          "waiting",
		WaitingSince:    "2026-08-04T10:00:00+03:00",
	})
	rework.MarkLaunchWaiting(tasklifecycle.LaunchWait{
		Scope:           "project",
		Resource:        "core-eggs-gd",
		ConflictClaimID: "claim-holder",
		Reason:          "waiting",
		WaitingSince:    "2026-08-04T10:00:00+03:00",
	})
	app.Store.UpsertTask(todo)
	app.Store.UpsertTask(rework)

	candidates := app.Exec.RequeueCandidatesForReleasedSlot(execution.RuntimeSession{
		ClaimID:    "claim-holder",
		Agent:      "codex",
		ProjectID:  "core-eggs-gd",
		Repository: "core.eggs.gd",
	})
	if len(candidates) < 2 {
		t.Fatalf("candidates = %#v, want both tasks", candidates)
	}
	if candidates[0].Ref != "CORE-125" {
		t.Fatalf("first candidate = %s, want CORE-125 needs_rework before todo", candidates[0].Ref)
	}
	if candidates[1].Ref != "CORE-124" {
		t.Fatalf("second candidate = %s, want CORE-124", candidates[1].Ref)
	}
}

// TestAnnotateTaskStaleWaitClearRequeuesLaunchableTask proves clearing wait
// metadata alone cannot leave a launchable task idle forever.
func TestAnnotateTaskStaleWaitClearRequeuesLaunchableTask(t *testing.T) {
	root, _ := writeRuntimeRequeueFixture(t)
	installFakeCodexAppServer(t, root)
	nextPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-core-124-stale.md")
	writeTestFile(t, nextPath, testTaskMarkdownWithPriority("work-core-124-stale", "CORE-124", "Stale wait idle", "todo", 1))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	task, err := markdown.LoadTaskFile(root, nextPath)
	if err != nil {
		t.Fatal(err)
	}
	task.MarkLaunchWaiting(tasklifecycle.LaunchWait{
		Scope:           "project",
		Resource:        "core-eggs-gd",
		ConflictClaimID: "gone-claim",
		Reason:          "active runtime session gone-claim already owns project",
		WaitingSince:    "2026-08-04T10:00:00+03:00",
	})
	app.Store.UpsertTask(task)

	got := app.Exec.AnnotateTask(task)
	if got.LaunchEvaluation.Waiting != nil {
		t.Fatalf("waiting = %#v, want cleared", got.LaunchEvaluation.Waiting)
	}

	waitForTaskStatus(t, root, nextPath, "needs_review")
	assertEventLogContains(t, root, `"type":"launch_waiting_stale_cleared"`)
	assertEventLogContains(t, root, `"type":"launch_wait_requeued"`)
	assertEventLogContains(t, root, `"type":"launch_claimed"`)
}

func TestStartLaunchCandidateFailureMovesTaskToBlocked(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-a.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-46", "Launch failure", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Repository: "core.eggs.gd",
		WorkingDir: root,
		Command:    []string{"definitely-missing-core-launcher-binary"},
		Launchable: true,
		Outcome:    "launchable",
	}

	app.Exec.StartLaunchCandidate(t.Context(), task)

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", loaded.Status)
	}
	if len(loaded.Comments) == 0 {
		t.Fatal("missing launch failure comment")
	}
	comment := loaded.Comments[0].Text
	// CORE-150: missing launcher binary is a classified setup failure, not a
	// vague "Launch failed:" string.
	for _, expected := range []string{"missing_executable", "Claude CLI", "manual"} {
		if !strings.Contains(comment, expected) {
			t.Fatalf("comments = %#v, want classified missing_executable blocker containing %q", loaded.Comments, expected)
		}
	}
	if len(app.State().RuntimeSessions) != 0 {
		t.Fatalf("sessions = %#v, want none after start failure", app.State().RuntimeSessions)
	}
}

func TestStartLaunchCandidateWritesSessionLog(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-a.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-51", "Log session", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Repository: "core.eggs.gd",
		WorkingDir: root,
		Command:    []string{"/bin/sh", "-c", "echo stdout-line; echo stderr-line >&2"},
		Launchable: true,
		Outcome:    "launchable",
	}

	app.Exec.StartLaunchCandidate(t.Context(), task)
	waitForNoActiveRuntimeSessions(t, app)

	logText := firstSessionLog(t, root)
	for _, expected := range []string{"stdout-line", "stderr-line", "[core] starting agent=claude"} {
		if !strings.Contains(logText, expected) {
			t.Fatalf("session log missing %q:\n%s", expected, logText)
		}
	}
}

func TestSuccessfulStartedProcessMovesTaskToNeedsReviewAfterExit(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-01-success.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-57", "Successful session", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "codex",
		Repository: "core.eggs.gd",
		WorkingDir: root,
		Command:    []string{"/bin/sh", "-c", "echo done"},
		Launchable: true,
		Outcome:    "launchable",
	}

	app.Exec.StartLaunchCandidate(t.Context(), task)
	waitForNoActiveRuntimeSessions(t, app)

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "needs_review" {
		t.Fatalf("status = %q, want needs_review", loaded.Status)
	}
	if len(loaded.Comments) == 0 || !strings.Contains(loaded.Comments[0].Text, "Agent execution completed.") {
		t.Fatalf("comments = %#v, want runtime completion comment", loaded.Comments)
	}
}

func TestActiveExecutionDefersWorkerLifecycleFinalization(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-01-active.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-57", "Active session", "doing"))

	app := Compose(ComposeConfig{CoreRoot: root})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	app.Store.UpsertTask(task)
	app.Exec.UpsertSession(execution.RuntimeSession{
		ClaimID:         "core-57-active",
		TaskRef:         task.Ref,
		TaskID:          task.ID,
		TaskPath:        task.RelativePath,
		ProjectID:       task.ProjectID,
		Repository:      "core.eggs.gd",
		Status:          "running",
		ExecutionStatus: "running",
	})

	if _, err := app.TaskStore.Mutate(tasklifecycle.TaskPatch{
		Path:                task.RelativePath,
		Status:              "needs_review",
		CommentAuthor:       "codex",
		Comment:             `{"outcome":"completed","summary":"worker tried to finalize early"}`,
		AllowStatusOverride: true,
	}); err != nil {
		t.Fatal(err)
	}

	observed := observeTaskFileChange(t, app, taskPath)
	if !observed.GuardApplied || observed.After.Status != "doing" {
		t.Fatalf("guarded observe = %#v, want status doing", observed)
	}
	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "doing" {
		t.Fatalf("status = %q, want doing until execution is terminal", loaded.Status)
	}
	if len(loaded.Comments) < 2 || !strings.Contains(loaded.Comments[len(loaded.Comments)-1].Text, "active execution") {
		t.Fatalf("comments = %#v, want active execution guard comment", loaded.Comments)
	}
}

func TestFailedStartedProcessMovesTaskToBlocked(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-a.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-52", "Failed session", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Repository: "core.eggs.gd",
		WorkingDir: root,
		Command:    []string{"/bin/sh", "-c", "echo failing; exit 1"},
		Launchable: true,
		Outcome:    "launchable",
	}

	app.Exec.StartLaunchCandidate(t.Context(), task)
	waitForLaunchToFinish(t, app, root, taskPath, 15*time.Second)

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", loaded.Status)
	}
	if len(loaded.Comments) == 0 || !strings.Contains(loaded.Comments[0].Text, "Launch process failed: exit status 1") {
		t.Fatalf("comments = %#v, want process failure comment", loaded.Comments)
	}
	if !strings.Contains(loaded.Comments[0].Text, "_registry/sessions/") {
		t.Fatalf("failure comment missing session log path: %#v", loaded.Comments)
	}
}

func TestBackgroundClaudeQuotaErrorMovesTaskToBlockedWithProviderComment(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-quota.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-76", "Claude quota failure", "todo"))
	claude := writeExecutable(t, root, "fake-claude", `#!/bin/sh
if [ "$1" = "--bg" ]; then
  echo "backgrounded · bgquota · launched"
  exit 0
fi
if [ "$1" = "agents" ]; then
  echo '[{"id":"bgquota","state":"failed"}]'
  exit 0
fi
if [ "$1" = "logs" ]; then
  echo "Quota exceeded: usage limit exceeded for this billing period."
  exit 0
fi
exit 1
`)

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Backend:    "background-remote",
		Repository: "core.eggs.gd",
		WorkingDir: root,
		Command:    []string{claude, "--bg", "--remote-control", "--name", "CORE-76", "prompt"},
		Launchable: true,
		Outcome:    "launchable",
	}

	app.Exec.StartLaunchCandidate(t.Context(), task)
	waitForNoActiveRuntimeSessions(t, app)

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", loaded.Status)
	}
	if len(loaded.Comments) == 0 {
		t.Fatal("missing provider blocker comment")
	}
	comment := loaded.Comments[0].Text
	for _, expected := range []string{"Claude quota exhausted", "quota_exceeded", "bgquota", "_registry/sessions/", "manual"} {
		if !strings.Contains(comment, expected) {
			t.Fatalf("provider comment missing %q:\n%s", expected, comment)
		}
	}
	state := app.State()
	if len(state.RuntimeSessions) != 0 {
		t.Fatalf("runtime sessions = %#v, want active strip empty after provider-error finalization", state.RuntimeSessions)
	}
	session, ok := findSessionInGroups(state, "CORE-76")
	if !ok {
		t.Fatalf("session_groups = %#v, want terminal provider-error session in history", state.SessionGroups)
	}
	if session.IsActive() {
		t.Fatalf("provider error session is still active: %#v", session)
	}
	if session.ProviderError == nil || session.ProviderError.Kind != "quota_exceeded" {
		t.Fatalf("provider error = %#v, want quota_exceeded", session.ProviderError)
	}
	if session.ProviderError.RetryPolicy != tasklifecycle.RetryPolicyManual {
		t.Fatalf("retry_policy = %#v, want manual", session.ProviderError)
	}
}

// TestBackgroundClaudeCompletedOutcomeWithRateLimitProseNeedsReview is the
// CORE-98 / CORE-96 regression: a successful Claude background session whose
// transcript mentions a "429 rate-limit failure mode" test case must finalize
// as needs_review, not as a false-positive provider rate_limited blocker.
func TestBackgroundClaudeCompletedOutcomeWithRateLimitProseNeedsReview(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-03-core-98.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-98", "False positive rate limit prose", "todo"))
	claude := writeExecutable(t, root, "fake-claude-core98", `#!/bin/sh
if [ "$1" = "--bg" ]; then
  echo "backgrounded · bgcore98 · launched"
  exit 0
fi
if [ "$1" = "agents" ]; then
  echo '[{"id":"bgcore98","state":"done"}]'
  exit 0
fi
if [ "$1" = "logs" ]; then
  echo "Continue here, on your phone, or at https://claude.ai/code/session_CORE98"
  echo "Added coverage for the 429 rate-limit failure mode and auth/quota docs."
  echo '{"outcome":"completed","summary":"Plane provider shipped.","artifacts":["internal/taskprovider/provider.go"],"tests":["go test ./internal/taskprovider/..."]}'
  exit 0
fi
exit 1
`)

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Backend:    "background-remote",
		Repository: "core.eggs.gd",
		WorkingDir: root,
		Command:    []string{claude, "--bg", "--remote-control", "--name", "CORE-98", "prompt"},
		Launchable: true,
		Outcome:    "launchable",
	}

	app.Exec.StartLaunchCandidate(t.Context(), task)
	waitForNoActiveRuntimeSessions(t, app)

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "needs_review" {
		t.Fatalf("status = %q, want needs_review (got blocked on rate-limit prose?)", loaded.Status)
	}
	if len(loaded.Comments) == 0 || !strings.Contains(loaded.Comments[len(loaded.Comments)-1].Text, "Plane provider shipped.") {
		t.Fatalf("comments = %#v, want worker summary", loaded.Comments)
	}
	for _, comment := range loaded.Comments {
		if strings.Contains(comment.Text, "Provider blocker detected") || strings.Contains(comment.Text, "rate_limited") {
			t.Fatalf("unexpected provider blocker comment: %s", comment.Text)
		}
	}
	state := app.State()
	if len(state.RuntimeSessions) != 0 {
		t.Fatalf("runtime sessions = %#v, want no stale active/running slot after success", state.RuntimeSessions)
	}
}

func TestBackgroundRemoteIdleSessionNeedsOperatorAttentionWithoutBlockingTask(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-idle.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-71", "Idle background session", "todo"))
	claude := writeExecutable(t, root, "fake-claude-idle", `#!/bin/sh
if [ "$1" = "--bg" ]; then
  echo "backgrounded · bgidle · launched"
  exit 0
fi
if [ "$1" = "agents" ]; then
  echo '[{"id":"bgidle","state":"running"}]'
  exit 0
fi
if [ "$1" = "logs" ]; then
  echo "Continue here, on your phone, or at https://claude.ai/code/session_IDLE"
  exit 0
fi
if [ "$1" = "stop" ]; then
  exit 0
fi
exit 1
`)

	restorePoll := execution.SetBackgroundRemotePollIntervalForTest(5 * time.Millisecond)
	defer restorePoll()

	ctx, cancel := context.WithCancel(t.Context())
	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: 25 * time.Millisecond})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Backend:    "background-remote",
		Repository: "core.eggs.gd",
		WorkingDir: root,
		Command:    []string{claude, "--bg", "--remote-control", "--name", "CORE-71", "prompt"},
		Launchable: true,
		Outcome:    "launchable",
	}

	app.Exec.StartLaunchCandidate(ctx, task)
	waitForSessionState(t, app, "operator_attention")

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "doing" {
		t.Fatalf("status = %q, want doing while idle session awaits operator attention", loaded.Status)
	}
	session := app.State().RuntimeSessions[0]
	if !session.IsActive() || session.BlockingReason == "" || session.LastEventAt == "" {
		t.Fatalf("idle session not exposed as active attention state: %#v", session)
	}
	if session.RemoteControlURL == "" {
		t.Fatalf("idle attention path requires an announced remote-control URL: %#v", session)
	}
	cancel()
	waitForNoActiveRuntimeSessions(t, app)
}

// TestBackgroundRemoteHollowSessionFailsWhenRemoteControlNeverRegisters covers
// CORE-142: dispatch succeeds and a background id exists, but Claude never
// publishes an app-visible remote-control URL. Core must fail as
// remote_control_unavailable instead of sitting in idle attention for 10m.
func TestBackgroundRemoteHollowSessionFailsWhenRemoteControlNeverRegisters(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-05-hollow.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-142", "Hollow Claude session", "todo"))
	claude := writeExecutable(t, root, "fake-claude-hollow", `#!/bin/sh
if [ "$1" = "--bg" ]; then
  echo "backgrounded · bghollow · CORE-142"
  exit 0
fi
if [ "$1" = "agents" ]; then
  echo '[{"id":"bghollow","state":"running"}]'
  exit 0
fi
if [ "$1" = "logs" ]; then
  echo "background agent started"
  exit 0
fi
if [ "$1" = "stop" ]; then
  exit 0
fi
exit 1
`)

	restorePoll := execution.SetBackgroundRemotePollIntervalForTest(5 * time.Millisecond)
	defer restorePoll()
	restoreReady := execution.SetBackgroundRemoteControlReadyTimeoutForTest(20 * time.Millisecond)
	defer restoreReady()

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Minute})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Backend:    "background-remote",
		Repository: "core.eggs.gd",
		WorkingDir: root,
		Command:    []string{claude, "--bg", "--remote-control", "--name", "CORE-142", "prompt"},
		Launchable: true,
		Outcome:    "launchable",
	}

	app.Exec.StartLaunchCandidate(t.Context(), task)
	waitForNoActiveRuntimeSessions(t, app)

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "blocked" {
		t.Fatalf("status = %q, want blocked for remote_control_unavailable", loaded.Status)
	}
	found := false
	for _, comment := range loaded.Comments {
		if strings.Contains(comment.Text, "remote_control_unavailable") || strings.Contains(comment.Text, "never published a remote-control URL") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("comments = %#v, want remote_control_unavailable blocker", loaded.Comments)
	}
	for _, session := range app.State().RuntimeSessions {
		if session.IsActive() {
			t.Fatalf("hollow session still active: %#v", session)
		}
	}
}

// TestBackgroundRemoteHollowSessionClassifiesClaudeLoginFailure covers CORE-150:
// expired/logged-out Claude startup must surface as auth_required, not a vague
// remote_control_unavailable launcher error.
func TestBackgroundRemoteHollowSessionClassifiesClaudeLoginFailure(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-09-08-auth.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-150", "Claude login expired", "todo"))
	claude := writeExecutable(t, root, "fake-claude-auth", `#!/bin/sh
if [ "$1" = "--bg" ]; then
  echo "backgrounded · bgauth · CORE-150"
  exit 0
fi
if [ "$1" = "agents" ]; then
  echo '[{"id":"bgauth","state":"running"}]'
  exit 0
fi
if [ "$1" = "logs" ]; then
  printf '\033[31mError: not logged in. Please run /login\033[0m\n'
  exit 0
fi
if [ "$1" = "stop" ]; then
  exit 0
fi
exit 1
`)

	restorePoll := execution.SetBackgroundRemotePollIntervalForTest(5 * time.Millisecond)
	defer restorePoll()
	restoreReady := execution.SetBackgroundRemoteControlReadyTimeoutForTest(20 * time.Millisecond)
	defer restoreReady()

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Minute})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Backend:    "background-remote",
		Repository: "core.eggs.gd",
		WorkingDir: root,
		Command:    []string{claude, "--bg", "--remote-control", "--name", "CORE-150", "prompt"},
		Launchable: true,
		Outcome:    "launchable",
	}

	app.Exec.StartLaunchCandidate(t.Context(), task)
	waitForNoActiveRuntimeSessions(t, app)

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "blocked" {
		t.Fatalf("status = %q, want blocked for auth_required", loaded.Status)
	}
	found := false
	for _, comment := range loaded.Comments {
		if strings.Contains(comment.Text, "auth_required") && strings.Contains(comment.Text, "Claude login required") {
			found = true
			if strings.Contains(comment.Text, "\x1b[") {
				t.Fatalf("comment dumped ANSI escapes:\n%s", comment.Text)
			}
			break
		}
	}
	if !found {
		t.Fatalf("comments = %#v, want auth_required actionable blocker", loaded.Comments)
	}
	session, ok := findSessionInGroups(app.State(), "CORE-150")
	if !ok {
		t.Fatalf("session_groups = %#v, want terminal auth session", app.State().SessionGroups)
	}
	if session.ProviderError == nil || session.ProviderError.Kind != "auth_required" {
		t.Fatalf("provider error = %#v, want auth_required", session.ProviderError)
	}
	if session.ProviderError.RetryPolicy != tasklifecycle.RetryPolicyManual {
		t.Fatalf("retry_policy = %#v, want manual", session.ProviderError)
	}
	if session.ProviderError.SuggestedAction == "" {
		t.Fatal("expected suggested_action on provider error")
	}
}

// TestMissingCodexExecutableBlocksTaskWithProviderDiagnostic covers CORE-150:
// unresolved Codex CLI on the daemon PATH blocks with missing_executable.
func TestMissingCodexExecutableBlocksTaskWithProviderDiagnostic(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-09-08-missing-codex.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-150B", "Missing Codex CLI", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.Assignee = "codex"
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:       "codex",
		FailedGates: []string{"agent_executable: Codex CLI is not available on the daemon PATH. Install the standalone Codex CLI or configure the Codex binary path, then retry this task."},
	}

	app.Exec.HandleLaunchResult(t.Context(), task)
	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "blocked" {
		t.Fatalf("status = %q, want blocked for missing executable", loaded.Status)
	}
	if len(loaded.Comments) == 0 {
		t.Fatal("missing provider comment")
	}
	comment := loaded.Comments[0].Text
	for _, expected := range []string{"missing_executable", "Codex CLI", "manual"} {
		if !strings.Contains(comment, expected) {
			t.Fatalf("comment missing %q:\n%s", expected, comment)
		}
	}
}

// TestAssigneeSlotReservedDuringClaudeVisibleSessionWait covers CORE-149:
// two Claude tasks on different projects must not both start while the first is
// still waiting for remote-control URL registration.
func TestAssigneeSlotReservedDuringClaudeVisibleSessionWait(t *testing.T) {
	// CORE-145 gets requeued and re-evaluated from disk once CORE-144's slot
	// releases; that path resolves WorkingDir as filepath.Join(coreRoot, "..",
	// repository) (real repos live as siblings of the Core root on disk), so
	// coreRoot must be a subdirectory of the temp dir, with repo-a/repo-b
	// created as its siblings — same layout writeRuntimeRequeueFixture uses.
	base := t.TempDir()
	root := filepath.Join(base, "core")
	for _, repo := range []string{"repo-a", "repo-b"} {
		if err := os.MkdirAll(filepath.Join(base, repo), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	firstPath := filepath.Join(root, "Work", "project-a", "tasks", "2026-09-08-core-144.md")
	secondPath := filepath.Join(root, "Work", "project-b", "tasks", "2026-09-08-core-145.md")
	// testTaskForProjectMarkdown hardcodes "id: work-test" for every task it
	// generates. sameTaskIdentity treats matching TaskID alone as proof two
	// sessions belong to the same task (OR across path/id/ref) — with both
	// fixtures sharing that id, CORE-144's still-active session gets
	// mistaken for CORE-145's own, so the assignee-conflict wait keeps
	// getting revalidated away and immediately re-detected, looping forever.
	// Real tasks always get unique ids (work-<date>-<slug>), so give each
	// fixture its own id here rather than touching sameTaskIdentity itself.
	writeTestFile(t, firstPath, strings.NewReplacer("assignee: codex", "assignee: claude", "id: work-test", "id: work-test-144").Replace(testTaskForProjectMarkdown("CORE-144", "First Claude startup", "todo", "project-a", "repo-a")))
	writeTestFile(t, secondPath, strings.NewReplacer("assignee: codex", "assignee: claude", "id: work-test", "id: work-test-145").Replace(testTaskForProjectMarkdown("CORE-145", "Second Claude startup", "todo", "project-b", "repo-b")))
	// CORE-145 gets requeued and re-evaluated from disk once CORE-144's slot
	// releases; the fleet_profile gate needs a real Fleet/claude.md to exist
	// for that re-evaluation to pass, same as the fixture used elsewhere in
	// this file for codex/cursor.
	writeTestFile(t, filepath.Join(root, "Fleet", "claude.md"), "# Claude\n")

	// CORE-144 is launched directly with an explicit LaunchEvaluation.Command
	// below, but CORE-145 gets requeued and re-evaluated from scratch once
	// CORE-144's slot releases — that path calls resolveClaudeBinary(), which
	// does exec.LookPath("claude") for real. Name the fake executable
	// "claude" and put its directory first on PATH so both the direct call
	// and the automatic re-evaluation resolve to this fake, never the real
	// Claude CLI that may also be installed on the machine running the test.
	claude := writeExecutable(t, root, "claude", `#!/bin/sh
if [ "$1" = "--bg" ]; then
  echo "backgrounded · bgwait · launched"
  exit 0
fi
if [ "$1" = "agents" ]; then
  echo '[{"id":"bgwait","state":"running"}]'
  exit 0
fi
if [ "$1" = "logs" ]; then
  echo "background agent started; remote control pending"
  exit 0
fi
if [ "$1" = "stop" ]; then
  exit 0
fi
exit 1
`)
	t.Setenv("PATH", filepath.Dir(claude)+string(os.PathListSeparator)+os.Getenv("PATH"))

	restorePoll := execution.SetBackgroundRemotePollIntervalForTest(5 * time.Millisecond)
	defer restorePoll()
	// CORE-144's fake claude script never announces a remote-control URL, by
	// design: this test observes CORE-145 correctly queuing behind CORE-144
	// while it is still mid-startup. waitForRuntimeState below has its own
	// 2s budget; the ready timeout must stay well above that or CORE-144's
	// hollow-session fail-fast can release its slot mid-test and race with
	// the assertions this test is actually trying to make.
	restoreReady := execution.SetBackgroundRemoteControlReadyTimeoutForTest(30 * time.Second)
	defer restoreReady()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Minute})

	first, err := markdown.LoadTaskFile(root, firstPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := markdown.LoadTaskFile(root, secondPath)
	if err != nil {
		t.Fatal(err)
	}
	first.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Backend:    "background-remote",
		Repository: "repo-a",
		WorkingDir: root,
		Command:    []string{claude, "--bg", "--remote-control", "--name", "CORE-144", "prompt"},
		Launchable: true,
		Outcome:    "launchable",
	}
	second.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Backend:    "background-remote",
		Repository: "repo-b",
		WorkingDir: root,
		Command:    []string{claude, "--bg", "--remote-control", "--name", "CORE-145", "prompt"},
		Launchable: true,
		Outcome:    "launchable",
	}

	app.Exec.StartLaunchCandidate(ctx, first)
	state := waitForRuntimeState(t, app, func(state State) bool {
		for _, session := range state.RuntimeSessions {
			if session.IsActive() && session.ExecutionStatus == "waiting_for_visible_session" {
				return true
			}
		}
		return false
	})
	if got := state.RuntimeSessions[0].ExecutionStatus; got != "waiting_for_visible_session" {
		t.Fatalf("handshake status = %q, want waiting_for_visible_session", got)
	}

	app.Exec.StartLaunchCandidate(ctx, second)
	state = waitForRuntimeState(t, app, func(state State) bool {
		waiting := 0
		for _, task := range state.Tasks {
			if task.LaunchEvaluation.Waiting != nil && task.Ref == "CORE-145" {
				waiting++
			}
		}
		active := 0
		for _, session := range state.RuntimeSessions {
			if session.IsActive() {
				active++
			}
		}
		return waiting == 1 && active == 1
	})
	secondLoaded, err := markdown.LoadTaskFile(root, secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if secondLoaded.Status != "todo" {
		t.Fatalf("second task status = %q, want todo (not claimed during first startup)", secondLoaded.Status)
	}
	var wait *tasklifecycle.LaunchWait
	for _, task := range state.Tasks {
		if task.Ref == "CORE-145" {
			wait = task.LaunchEvaluation.Waiting
			break
		}
	}
	if wait == nil || wait.Scope != "assignee" {
		t.Fatalf("second task wait = %#v, want assignee scope", wait)
	}
	if !strings.Contains(wait.Reason, "provider startup") {
		t.Fatalf("wait reason = %q, want provider startup explanation", wait.Reason)
	}
	assertEventLogContains(t, root, `"type":"runtime_slot_reserved"`)
	assertEventLogContains(t, root, `"type":"launch_waiting"`)
	assertEventLogContains(t, root, `"provider_startup":true`)
	cancel()
	// CORE-145's own background-remote session may have only just dispatched
	// via requeue when ctx was cancelled; its poll goroutine can take a beat
	// to notice ctx.Done() and finish. Give it a moment, then explicitly
	// remove whatever is left rather than assume it lands within
	// waitForNoActiveRuntimeSessions's fixed poll budget.
	time.Sleep(250 * time.Millisecond)
	for _, session := range app.State().RuntimeSessions {
		if session.IsActive() {
			app.Exec.RemoveSession(session.ClaimID)
		}
	}
	waitForSessionsRemoved(t, app)
	time.Sleep(150 * time.Millisecond)
}

// TestProviderStartupFailureReleasesAssigneeSlot covers CORE-149: after Claude
// remote-control registration fails, the assignee slot is released and a later
// task can reserve/start.
func TestProviderStartupFailureReleasesAssigneeSlot(t *testing.T) {
	root := t.TempDir()
	firstPath := filepath.Join(root, "Work", "project-a", "tasks", "2026-09-08-core-144-fail.md")
	secondPath := filepath.Join(root, "Work", "project-b", "tasks", "2026-09-08-core-145-retry.md")
	writeTestFile(t, firstPath, testTaskForProjectMarkdown("CORE-144", "Hollow first", "todo", "project-a", "repo-a"))
	writeTestFile(t, secondPath, testTaskForProjectMarkdown("CORE-145", "Retry after failure", "todo", "project-b", "repo-b"))

	claude := writeExecutable(t, root, "fake-claude-fail-then-retry", `#!/bin/sh
if [ "$1" = "--bg" ]; then
  name="bg"
  for arg in "$@"; do
    case "$arg" in
      CORE-144) name="bgfail" ;;
      CORE-145) name="bgretry" ;;
    esac
  done
  echo "backgrounded · $name · launched"
  exit 0
fi
if [ "$1" = "agents" ]; then
  id="$2"
  if [ -z "$id" ]; then id="bgfail"; fi
  # claude agents --json has no id arg; always list both ids as running until stop
  echo '[{"id":"bgfail","state":"running"},{"id":"bgretry","state":"running"}]'
  exit 0
fi
if [ "$1" = "logs" ]; then
  echo "background agent started; no remote control url"
  exit 0
fi
if [ "$1" = "stop" ]; then
  exit 0
fi
exit 1
`)

	restorePoll := execution.SetBackgroundRemotePollIntervalForTest(5 * time.Millisecond)
	defer restorePoll()
	restoreReady := execution.SetBackgroundRemoteControlReadyTimeoutForTest(20 * time.Millisecond)
	defer restoreReady()

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Minute})
	first, err := markdown.LoadTaskFile(root, firstPath)
	if err != nil {
		t.Fatal(err)
	}
	first.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Backend:    "background-remote",
		Repository: "repo-a",
		WorkingDir: root,
		Command:    []string{claude, "--bg", "--remote-control", "--name", "CORE-144", "prompt"},
		Launchable: true,
		Outcome:    "launchable",
	}
	app.Exec.StartLaunchCandidate(t.Context(), first)
	waitForNoActiveRuntimeSessions(t, app)
	assertEventLogContains(t, root, `"type":"runtime_slot_released_after_provider_startup_failure"`)

	second, err := markdown.LoadTaskFile(root, secondPath)
	if err != nil {
		t.Fatal(err)
	}
	second.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Backend:    "background-remote",
		Repository: "repo-b",
		WorkingDir: root,
		Command:    []string{claude, "--bg", "--remote-control", "--name", "CORE-145", "prompt"},
		Launchable: true,
		Outcome:    "launchable",
	}
	if _, _, ok := app.Exec.TryReserveSession(execution.RuntimeSession{
		ClaimID:    "post-failure-probe",
		ProjectID:  "project-b",
		Repository: "repo-b",
		Agent:      "claude",
		Status:     "starting",
	}); !ok {
		t.Fatal("assignee slot was not released after provider startup failure")
	}
	app.Exec.RemoveSession("post-failure-probe")

	app.Exec.StartLaunchCandidate(t.Context(), second)
	state := waitForRuntimeState(t, app, func(state State) bool {
		for _, session := range state.RuntimeSessions {
			if session.IsActive() && session.TaskRef == "CORE-145" {
				return true
			}
		}
		loaded, err := markdown.LoadTaskFile(root, secondPath)
		return err == nil && loaded.Status == "doing"
	})
	found := false
	for _, session := range state.RuntimeSessions {
		if session.TaskRef == "CORE-145" && session.IsActive() {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected CORE-145 active after slot release: %#v", state.RuntimeSessions)
	}
	waitForNoActiveRuntimeSessions(t, app)
}

// TestCursorCreateChatStartupReservesAssigneeSlot covers CORE-149 for a
// non-Claude provider: Cursor create-chat handshake owns the assignee slot.
func TestCursorCreateChatStartupReservesAssigneeSlot(t *testing.T) {
	// CORE-149B gets requeued and re-evaluated from disk once CORE-149A's
	// slot releases; that path resolves WorkingDir as filepath.Join(coreRoot,
	// "..", repository) (real repos live as siblings of the Core root on
	// disk), so coreRoot must be a subdirectory of the temp dir, with
	// repo-a/repo-b created as its siblings.
	base := t.TempDir()
	root := filepath.Join(base, "core")
	for _, repo := range []string{"repo-a", "repo-b"} {
		if err := os.MkdirAll(filepath.Join(base, repo), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	firstPath := filepath.Join(root, "Work", "project-a", "tasks", "2026-09-08-cursor-a.md")
	secondPath := filepath.Join(root, "Work", "project-b", "tasks", "2026-09-08-cursor-b.md")
	// Give each fixture its own id: testTaskForProjectMarkdown hardcodes
	// "id: work-test" for every task, and sameTaskIdentity treats matching
	// TaskID alone as proof two sessions are the same task — with both
	// sharing that id, CORE-149A's active session gets mistaken for
	// CORE-149B's own and the assignee-conflict wait never sticks. Real
	// tasks always get unique ids (work-<date>-<slug>).
	writeTestFile(t, firstPath, strings.NewReplacer("assignee: codex", "assignee: cursor", "id: work-test", "id: work-test-149a").Replace(testTaskForProjectMarkdown("CORE-149A", "Cursor first", "todo", "project-a", "repo-a")))
	writeTestFile(t, secondPath, strings.NewReplacer("assignee: codex", "assignee: cursor", "id: work-test", "id: work-test-149b").Replace(testTaskForProjectMarkdown("CORE-149B", "Cursor second", "todo", "project-b", "repo-b")))
	// CORE-149B gets requeued and re-evaluated from disk once CORE-149A's
	// slot releases; the fleet_profile gate needs a real Fleet/cursor.md to
	// exist for that re-evaluation to pass.
	writeTestFile(t, filepath.Join(root, "Fleet", "cursor.md"), "# Cursor\n")

	gate := filepath.Join(root, "cursor-create-gate")
	// CORE-149A launches with an explicit LaunchEvaluation.Command below, but
	// CORE-149B gets requeued and re-evaluated from scratch once CORE-149A's
	// slot releases — that path calls resolveCursorAgentBinary(), which does
	// exec.LookPath("cursor-agent") for real. Name the fake executable
	// "cursor-agent" and put its directory first on PATH so both the direct
	// call and the automatic re-evaluation resolve to this fake, never the
	// real cursor-agent CLI that may also be installed on the machine
	// running the test.
	cursor := writeExecutable(t, root, "cursor-agent", `#!/bin/sh
gate="`+gate+`"
if [ "$1" = "create-chat" ]; then
  : > "$gate"
  while [ -f "$gate" ]; do sleep 0.01; done
  echo "chat-startup-1"
  exit 0
fi
if [ "$1" = "--resume" ]; then
  sleep 0.05
  exit 0
fi
exit 1
`)
	t.Setenv("PATH", filepath.Dir(cursor)+string(os.PathListSeparator)+os.Getenv("PATH"))

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Minute})

	first, err := markdown.LoadTaskFile(root, firstPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := markdown.LoadTaskFile(root, secondPath)
	if err != nil {
		t.Fatal(err)
	}
	first.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "cursor",
		Backend:    cursorVisibleBackend,
		Repository: "repo-a",
		WorkingDir: root,
		Command:    []string{cursor, "--trust", "--workspace", root, "prompt-a"},
		Launchable: true,
		Outcome:    "launchable",
	}
	second.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "cursor",
		Backend:    cursorVisibleBackend,
		Repository: "repo-b",
		WorkingDir: root,
		Command:    []string{cursor, "--trust", "--workspace", root, "prompt-b"},
		Launchable: true,
		Outcome:    "launchable",
	}

	app.Exec.StartLaunchCandidate(ctx, first)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(gate); err == nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if _, err := os.Stat(gate); err != nil {
		t.Fatal("cursor create-chat did not start")
	}

	state := waitForRuntimeState(t, app, func(state State) bool {
		for _, session := range state.RuntimeSessions {
			if session.IsActive() && session.ExecutionStatus == "waiting_for_visible_session" {
				return true
			}
		}
		return false
	})
	if !state.RuntimeSessions[0].IsProviderStartup() {
		t.Fatalf("cursor handshake session not marked startup-active: %#v", state.RuntimeSessions[0])
	}

	app.Exec.StartLaunchCandidate(ctx, second)
	state = waitForRuntimeState(t, app, func(state State) bool {
		for _, task := range state.Tasks {
			if task.Ref == "CORE-149B" && task.LaunchEvaluation.Waiting != nil {
				return true
			}
		}
		return false
	})
	secondLoaded, err := markdown.LoadTaskFile(root, secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if secondLoaded.Status != "todo" {
		t.Fatalf("second cursor task status = %q, want todo", secondLoaded.Status)
	}
	var wait *tasklifecycle.LaunchWait
	for _, task := range state.Tasks {
		if task.Ref == "CORE-149B" {
			wait = task.LaunchEvaluation.Waiting
			break
		}
	}
	if wait == nil || wait.Scope != "assignee" {
		t.Fatalf("cursor wait = %#v, want assignee scope", wait)
	}

	// Releasing the gate lets CORE-149A's create-chat return, which frees the
	// assignee slot and requeues CORE-149B for a real launch attempt of its
	// own (also gated behind the same shared "gate" file, already gone by
	// then). Give that requeue a moment to actually dispatch before tearing
	// everything down, then cancel and explicitly remove whatever session(s)
	// remain rather than assume the fake script reaches a terminal state
	// within waitForNoActiveRuntimeSessions's fixed poll budget.
	_ = os.Remove(gate)
	time.Sleep(250 * time.Millisecond)
	cancel()
	for _, session := range app.State().RuntimeSessions {
		if session.IsActive() {
			app.Exec.RemoveSession(session.ClaimID)
		}
	}
	waitForSessionsRemoved(t, app)
	// A session goroutine racing this test's own teardown can still be
	// mid-atomic-write (temp file created, rename pending) in Work/ when the
	// function returns, which makes t.TempDir()'s own RemoveAll fail with
	// "directory not empty". Give it a moment to finish before returning.
	time.Sleep(150 * time.Millisecond)
}

func TestBackgroundRemoteActiveSessionContinuesPastIdleThreshold(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-active.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-71", "Active background session", "todo"))
	countPath := filepath.Join(root, "logs-count")
	claude := writeExecutable(t, root, "fake-claude-active", `#!/bin/sh
count_path="`+countPath+`"
if [ "$1" = "--bg" ]; then
  echo "backgrounded · bgactive · launched"
  exit 0
fi
if [ "$1" = "agents" ]; then
  echo '[{"id":"bgactive","state":"running"}]'
  exit 0
fi
if [ "$1" = "logs" ]; then
  count=0
  if [ -f "$count_path" ]; then count=$(cat "$count_path"); fi
  count=$((count + 1))
  echo "$count" > "$count_path"
  echo "https://claude.ai/code/session_ACTIVE"
  echo "heartbeat $count"
  exit 0
fi
if [ "$1" = "stop" ]; then
  exit 0
fi
exit 1
`)

	restorePoll := execution.SetBackgroundRemotePollIntervalForTest(5 * time.Millisecond)
	defer restorePoll()

	ctx, cancel := context.WithCancel(t.Context())
	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: 25 * time.Millisecond})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Backend:    "background-remote",
		Repository: "core.eggs.gd",
		WorkingDir: root,
		Command:    []string{claude, "--bg", "--remote-control", "--name", "CORE-71", "prompt"},
		Launchable: true,
		Outcome:    "launchable",
	}

	app.Exec.StartLaunchCandidate(ctx, task)

	// The launch starts a few shell processes, so how long it takes depends on
	// the machine. Wait for the session instead of guessing a delay.
	state := waitForRuntimeState(t, app, func(s State) bool { return len(s.RuntimeSessions) == 1 })
	session := state.RuntimeSessions[0]
	if session.ExecutionStatus == "operator_attention" || !session.IsActive() {
		t.Fatalf("active heartbeat session was treated as idle: %#v", session)
	}
	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "doing" {
		t.Fatalf("status = %q, want doing for active long-running session", loaded.Status)
	}
	cancel()
	waitForNoActiveRuntimeSessions(t, app)
}

// TestBackgroundRemoteSessionFinalizesTaskFromTranscript is the CORE-97
// regression test for the outcome-routing gap: a Claude background-remote
// session's task must be finalized from the worker's compact JSON result in the
// polled transcript, without waiting for the provider to end the session. The
// session is then stopped for reuse and keeps the result in its record.
func TestBackgroundRemoteSessionFinalizesTaskFromTranscript(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-03-result.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-97", "Report outcome from transcript", "todo"))
	countPath := filepath.Join(root, "logs-count")
	claude := writeExecutable(t, root, "fake-claude-result", `#!/bin/sh
count_path="`+countPath+`"
if [ "$1" = "--bg" ]; then
  echo "backgrounded · bgresult · launched"
  exit 0
fi
if [ "$1" = "agents" ]; then
  echo '[{"id":"bgresult","state":"running"}]'
  exit 0
fi
if [ "$1" = "logs" ]; then
  count=0
  if [ -f "$count_path" ]; then count=$(cat "$count_path"); fi
  count=$((count + 1))
  echo "$count" > "$count_path"
  echo "Continue here, on your phone, or at https://claude.ai/code/session_RESULT"
  if [ "$count" -ge 2 ]; then
    echo "Done working."
    echo '{"outcome":"completed","summary":"Implemented the outcome protocol.","artifacts":["a.go"]}'
  fi
  exit 0
fi
if [ "$1" = "stop" ]; then
  exit 0
fi
exit 1
`)

	restorePoll := execution.SetBackgroundRemotePollIntervalForTest(5 * time.Millisecond)
	defer restorePoll()

	ctx, cancel := context.WithCancel(t.Context())
	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Backend:    "background-remote",
		Repository: "core.eggs.gd",
		WorkingDir: root,
		Command:    []string{claude, "--bg", "--remote-control", "--name", "CORE-97", "prompt"},
		Launchable: true,
		Outcome:    "launchable",
	}

	app.Exec.StartLaunchCandidate(ctx, task)

	deadline := time.Now().Add(2 * time.Second)
	var loaded tasklifecycle.Task
	for time.Now().Before(deadline) {
		loaded, err = markdown.LoadTaskFile(root, taskPath)
		if err == nil && loaded.Status == "needs_review" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if loaded.Status != "needs_review" {
		t.Fatalf("status = %q, want needs_review", loaded.Status)
	}
	if len(loaded.Comments) == 0 || !strings.Contains(loaded.Comments[len(loaded.Comments)-1].Text, "Implemented the outcome protocol.") {
		t.Fatalf("comments = %#v, want the worker's reported summary", loaded.Comments)
	}

	// The task moved because of the result in the transcript, not because the
	// session ended. Once the outcome is applied Fleet stops the background
	// session so the next task for the same project can resume it (session
	// reuse), so the session is gone soon after and its record keeps the result.
	waitForNoActiveRuntimeSessions(t, app)

	records, err := execution.LoadRuntimeSessionRecords(root)
	if err != nil {
		t.Fatal(err)
	}
	foundPersistedResult := false
	for _, record := range records {
		if record.Result != nil && record.Result.Outcome == "completed" && strings.Contains(record.Result.Summary, "Implemented the outcome protocol.") {
			foundPersistedResult = true
			break
		}
	}
	if !foundPersistedResult {
		t.Fatalf("session registry records = %#v, want persisted worker result after live completion", records)
	}

	cancel()
	waitForNoActiveRuntimeSessions(t, app)
}

// TestBackgroundRemoteSessionPersistsWorkerResultOnTerminalExit covers CORE-99:
// when Claude reaches a terminal provider state in the same poll window as the
// worker result, finish must still parse and persist Result into the session
// registry JSON (not only finalize the task via the nil-result fallback).
func TestBackgroundRemoteSessionPersistsWorkerResultOnTerminalExit(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-03-result-terminal.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-99", "Persist result on terminal exit", "todo"))
	countPath := filepath.Join(root, "agents-count")
	claude := writeExecutable(t, root, "fake-claude-terminal-result", `#!/bin/sh
count_path="`+countPath+`"
if [ "$1" = "--bg" ]; then
  echo "backgrounded · bgtermresult · launched"
  exit 0
fi
if [ "$1" = "agents" ]; then
  count=0
  if [ -f "$count_path" ]; then count=$(cat "$count_path"); fi
  count=$((count + 1))
  echo "$count" > "$count_path"
  if [ "$count" -ge 2 ]; then
    echo '[{"id":"bgtermresult","state":"done"}]'
  else
    echo '[{"id":"bgtermresult","state":"running"}]'
  fi
  exit 0
fi
if [ "$1" = "logs" ]; then
  echo "Continue here, on your phone, or at https://claude.ai/code/session_TERMRESULT"
  echo "Done working."
  echo '{"outcome":"completed","summary":"Persisted on terminal exit.","artifacts":["runtime_finalize.go"]}'
  exit 0
fi
if [ "$1" = "stop" ]; then
  exit 0
fi
exit 1
`)

	restorePoll := execution.SetBackgroundRemotePollIntervalForTest(5 * time.Millisecond)
	defer restorePoll()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Backend:    "background-remote",
		Repository: "core.eggs.gd",
		WorkingDir: root,
		Command:    []string{claude, "--bg", "--remote-control", "--name", "CORE-99", "prompt"},
		Launchable: true,
		Outcome:    "launchable",
	}

	app.Exec.StartLaunchCandidate(ctx, task)
	waitForNoActiveRuntimeSessions(t, app)

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "needs_review" {
		t.Fatalf("status = %q, want needs_review", loaded.Status)
	}
	if len(loaded.Comments) == 0 || !strings.Contains(loaded.Comments[len(loaded.Comments)-1].Text, "Persisted on terminal exit.") {
		t.Fatalf("comments = %#v, want the worker summary (not the nil-result fallback)", loaded.Comments)
	}

	records, err := execution.LoadRuntimeSessionRecords(root)
	if err != nil {
		t.Fatal(err)
	}
	var finished execution.RuntimeSession
	for _, record := range records {
		if record.TaskRef == "CORE-99" {
			finished = record
			break
		}
	}
	if finished.ClaimID == "" {
		t.Fatal("missing persisted CORE-99 session record")
	}
	if finished.Result == nil || finished.Result.Outcome != "completed" || finished.Result.Summary != "Persisted on terminal exit." {
		t.Fatalf("persisted Result = %#v, want completed worker result", finished.Result)
	}
	if finished.Status != "exited" || finished.ExecutionStatus != "succeeded" {
		t.Fatalf("persisted status = %s/%s, want exited/succeeded", finished.Status, finished.ExecutionStatus)
	}
}

// TestBackgroundRemoteSessionFinalizesBlockedOutcomeFromTranscript covers the
// blocked side of the same CORE-97 contract: a worker that reports it cannot
// proceed must move the task to blocked (not needs_review), again without
// waiting for the background session to end.
func TestBackgroundRemoteSessionFinalizesBlockedOutcomeFromTranscript(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-03-blocked-result.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-98", "Report blocked outcome from transcript", "todo"))
	claude := writeExecutable(t, root, "fake-claude-blocked-result", `#!/bin/sh
if [ "$1" = "--bg" ]; then
  echo "backgrounded · bgblocked · launched"
  exit 0
fi
if [ "$1" = "agents" ]; then
  echo '[{"id":"bgblocked","state":"running"}]'
  exit 0
fi
if [ "$1" = "logs" ]; then
  echo "Continue here, on your phone, or at https://claude.ai/code/session_BLOCKED"
  echo '{"outcome":"blocked","summary":"Need staging credentials.","blockers":["missing CORE_STAGING_TOKEN"]}'
  exit 0
fi
if [ "$1" = "stop" ]; then
  exit 0
fi
exit 1
`)

	restorePoll := execution.SetBackgroundRemotePollIntervalForTest(5 * time.Millisecond)
	defer restorePoll()

	ctx, cancel := context.WithCancel(t.Context())
	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Backend:    "background-remote",
		Repository: "core.eggs.gd",
		WorkingDir: root,
		Command:    []string{claude, "--bg", "--remote-control", "--name", "CORE-98", "prompt"},
		Launchable: true,
		Outcome:    "launchable",
	}

	app.Exec.StartLaunchCandidate(ctx, task)

	deadline := time.Now().Add(2 * time.Second)
	var loaded tasklifecycle.Task
	for time.Now().Before(deadline) {
		loaded, err = markdown.LoadTaskFile(root, taskPath)
		if err == nil && loaded.Status == "blocked" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if loaded.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", loaded.Status)
	}
	if len(loaded.Comments) == 0 || !strings.Contains(loaded.Comments[len(loaded.Comments)-1].Text, "missing CORE_STAGING_TOKEN") {
		t.Fatalf("comments = %#v, want the worker's reported blocker", loaded.Comments)
	}

	cancel()
	waitForNoActiveRuntimeSessions(t, app)
}

func TestGenericControlSocketProviderFailureMovesTaskToBlocked(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-socket.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-77", "Provider socket failure", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: time.Second})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "cursor",
		Backend:    "process",
		Repository: "core.eggs.gd",
		WorkingDir: root,
		Command:    []string{"/bin/sh", "-c", "echo 'ECONNREFUSED provider daemon control socket' >&2; exit 1"},
		Launchable: true,
		Outcome:    "launchable",
	}

	app.Exec.StartLaunchCandidate(t.Context(), task)
	waitForNoActiveRuntimeSessions(t, app)

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", loaded.Status)
	}
	if len(loaded.Comments) == 0 || !strings.Contains(loaded.Comments[0].Text, "control_socket_unavailable") {
		t.Fatalf("comments = %#v, want control socket provider blocker", loaded.Comments)
	}
	state := app.State()
	if len(state.RuntimeSessions) != 0 {
		t.Fatalf("runtime sessions = %#v, want active strip empty for terminal provider-error session", state.RuntimeSessions)
	}
	session, ok := findSessionInGroups(state, "CORE-77")
	if !ok {
		t.Fatalf("session_groups = %#v, want terminal provider-error session in history", state.SessionGroups)
	}
	if session.IsActive() {
		t.Fatalf("provider error session is still active: %#v", session)
	}
	if session.ProviderError == nil || session.ProviderError.Kind != "control_socket_unavailable" {
		t.Fatalf("provider error = %#v, want control_socket_unavailable", session.ProviderError)
	}
}

func TestLaunchNotReadyMovesPickupTaskToBlocked(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-01-a.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-53", "Missing repository", "needs_rework"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.FailLaunchGate("repository", "repositories is empty")
	task.ResolveLaunchEvaluation()

	app.Exec.HandleLaunchResult(t.Context(), task)

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", loaded.Status)
	}
	if len(loaded.Comments) == 0 || !strings.Contains(loaded.Comments[0].Text, "Launch is not ready: repository:") {
		t.Fatalf("comments = %#v, want launch-not-ready comment", loaded.Comments)
	}
	if loaded.BlockedReason != loaded.Comments[0].Text {
		t.Fatalf("blocked reason = %q, want comment %q", loaded.BlockedReason, loaded.Comments[0].Text)
	}
}

func TestUnavailableAssignedAgentLeavesPickupTaskUnblocked(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-01-a.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-62", "Codex-only task", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.FailLaunchGate("agent_live_ready", "Codex CLI live daemon launch is not verified yet.")
	task.ResolveLaunchEvaluation()

	app.Exec.HandleLaunchResult(t.Context(), task)

	loaded, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Status != "todo" {
		t.Fatalf("status = %q, want todo", loaded.Status)
	}
	if len(loaded.Comments) != 0 {
		t.Fatalf("comments = %#v, want no persistent blocker comments", loaded.Comments)
	}
	stateTask, ok := app.Store.TaskByPath(task.Path)
	if !ok {
		t.Fatal("task missing from runtime store")
	}
	if stateTask.LaunchEvaluation.Outcome != "not_launchable" {
		t.Fatalf("runtime outcome = %q, want not_launchable", stateTask.LaunchEvaluation.Outcome)
	}
	if len(stateTask.LaunchEvaluation.FailedGates) != 1 || !strings.HasPrefix(stateTask.LaunchEvaluation.FailedGates[0], "agent_live_ready:") {
		t.Fatalf("failed gates = %#v, want agent_live_ready", stateTask.LaunchEvaluation.FailedGates)
	}
}

func TestStartLaunchCandidateTimesOutHungProcess(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-a.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-51", "Timeout session", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root, DryRun: false, SessionTimeout: 25 * time.Millisecond})
	task, err := markdown.LoadTaskFile(root, taskPath)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Repository: "core.eggs.gd",
		WorkingDir: root,
		Command:    []string{"/bin/sleep", "1"},
		Launchable: true,
		Outcome:    "launchable",
	}

	app.Exec.StartLaunchCandidate(t.Context(), task)
	waitForNoActiveRuntimeSessions(t, app)

	data := firstSessionLog(t, root)
	if !strings.Contains(data, "process timed out after 25ms") {
		t.Fatalf("session log does not record timeout:\n%s", string(data))
	}
}

func launchCandidateForTest(t *testing.T, root string, path string) tasklifecycle.Task {
	t.Helper()

	task, err := markdown.LoadTaskFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	task.LaunchEvaluation = tasklifecycle.LaunchEvaluation{
		Agent:      "claude",
		Repository: "core.eggs.gd",
		WorkingDir: root,
		Command:    []string{"/bin/sleep", "1"},
		Launchable: true,
		Outcome:    "launchable",
	}
	return task
}

func testTaskMarkdownWithID(id string, ref string, title string, status string) string {
	return strings.Replace(testTaskMarkdown(ref, title, status), "id: work-test", "id: "+id, 1)
}

func testTaskMarkdownWithPriority(id string, ref string, title string, status string, priority int) string {
	body := testTaskMarkdownWithID(id, ref, title, status)
	return strings.Replace(body, "status: "+status+"\n", "status: "+status+"\npriority: "+strconv.Itoa(priority)+"\n", 1)
}

// waitForLaunchesToEnd waits until every launch has finished, including the task
// update that ends it. Looking at the session list is not enough: a session
// leaves it before its task is updated, and it is not in it yet when the launch
// has only just been started.
func waitForLaunchesToEnd(t *testing.T, app *App) {
	t.Helper()

	if !app.Exec.WaitIdle(15 * time.Second) {
		t.Fatalf("a launch was still running after 15s: sessions=%#v", app.State().RuntimeSessions)
	}
}

func waitForNoRuntimeSessions(t *testing.T, app *App) {
	t.Helper()

	waitForLaunchesToEnd(t, app)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(app.State().RuntimeSessions) == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("sessions still active after cancellation: %#v", app.State().RuntimeSessions)
}

func waitForRuntimeState(t *testing.T, app *App, ok func(State) bool) State {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		state := app.State()
		if ok(state) {
			return state
		}
		time.Sleep(10 * time.Millisecond)
	}
	state := app.State()
	t.Fatalf("runtime state did not satisfy condition: sessions=%#v tasks=%#v", state.RuntimeSessions, state.Tasks)
	return State{}
}

func waitForNoActiveRuntimeSessions(t *testing.T, app *App) {
	t.Helper()

	waitForLaunchesToEnd(t, app)
	waitForSessionsRemoved(t, app)
}

// waitForSessionsRemoved waits until no runtime session is active. It does not
// wait for launches to end: the tests that use it on their own tear down fake
// providers that block on a gate, whose runners are not expected to return.
func waitForSessionsRemoved(t *testing.T, app *App) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		active := 0
		for _, session := range app.State().RuntimeSessions {
			if session.IsActive() {
				active++
			}
		}
		if active == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("sessions still active: %#v", app.State().RuntimeSessions)
}

func findSessionInGroups(state State, taskRef string) (execution.RuntimeSession, bool) {
	for _, group := range state.SessionGroups {
		if group.TaskRef != taskRef && group.TaskID != taskRef {
			continue
		}
		if n := len(group.Sessions); n > 0 {
			return group.Sessions[n-1].RuntimeSession, true
		}
	}
	for _, group := range state.SessionGroups {
		for i := len(group.Sessions) - 1; i >= 0; i-- {
			session := group.Sessions[i].RuntimeSession
			if session.TaskRef == taskRef || session.ClaimID == taskRef || strings.Contains(session.ClaimID, strings.ToLower(taskRef)) {
				return session, true
			}
		}
	}
	return execution.RuntimeSession{}, false
}

func waitForSessionState(t *testing.T, app *App, state string) execution.RuntimeSession {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, session := range app.State().RuntimeSessions {
			if session.ExecutionStatus == state {
				return session
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no runtime session reached state %q: %#v", state, app.State().RuntimeSessions)
	return execution.RuntimeSession{}
}

func writeRuntimeRequeueFixture(t *testing.T) (string, string) {
	t.Helper()

	base := t.TempDir()
	root := filepath.Join(base, "core")
	repo := filepath.Join(base, "core.eggs.gd")
	writeTestFile(t, filepath.Join(root, "Fleet", "codex.md"), "# Codex\n")
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
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	return root, repo
}

func installFakeCodexAppServer(t *testing.T, root string) {
	t.Helper()

	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	codex := writeExecutable(t, binDir, "codex", `#!/bin/sh
if [ "$1" != "app-server" ]; then
  echo "unexpected codex args: $*" >&2
  exit 1
fi
while IFS= read -r line; do
  id=$(printf '%s\n' "$line" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')
  case "$line" in
    *'"method":"initialize"'*)
      printf '{"id":%s,"result":{}}\n' "$id"
      ;;
    *'"method":"thread/start"'*)
      printf '{"id":%s,"result":{"thread":{"id":"thread_%s_%s"}}}\n' "$id" "$$" "$id"
      ;;
    *'"method":"thread/resume"'*)
      tid=$(printf '%s\n' "$line" | sed -n 's/.*"threadId":"\([^"]*\)".*/\1/p')
      printf '{"id":%s,"result":{"thread":{"id":"%s"}}}\n' "$id" "$tid"
      ;;
    *'"method":"thread/name/set"'*)
      printf '{"id":%s,"result":{}}\n' "$id"
      ;;
    *'"method":"turn/start"'*)
      printf '{"id":%s,"result":{"turn":{"id":"turn_%s_%s","status":"completed"}}}\n' "$id" "$$" "$id"
      printf '{"method":"turn/completed","params":{"threadId":"thread_%s_%s","turnId":"turn_%s_%s","status":"completed"}}\n' "$$" "$id" "$$" "$id"
      exit 0
      ;;
  esac
done
`)
	t.Setenv("PATH", filepath.Dir(codex)+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func waitForTaskStatus(t *testing.T, root string, path string, status string) tasklifecycle.Task {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	var last tasklifecycle.Task
	for time.Now().Before(deadline) {
		task, err := markdown.LoadTaskFile(root, path)
		if err == nil {
			last = task
			if task.Status == status {
				return task
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s status = %q, want %q", path, last.Status, status)
	return tasklifecycle.Task{}
}

func assertEventLogContains(t *testing.T, root string, expected string) {
	t.Helper()

	data, err := audit.EventLogText(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(data, expected) {
		t.Fatalf("event log missing %q:\n%s", expected, data)
	}
}

func writeExecutable(t *testing.T, root string, name string, content string) string {
	t.Helper()

	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func firstSessionLog(t *testing.T, root string) string {
	t.Helper()

	records, err := execution.LoadRuntimeSessionRecords(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("session logs = %d records, want exactly one", len(records))
	}
	return execution.ReadSessionLog(root, records[0].LogPath)
}

// waitForLaunchToFinish waits until the launch of the task at taskPath is over.
// The caller's assertions report what is still wrong.
func waitForLaunchToFinish(t *testing.T, app *App, root string, taskPath string, timeout time.Duration) {
	t.Helper()

	if !app.Exec.WaitIdle(timeout) {
		t.Logf("launch of %s was still running after %s", taskPath, timeout)
	}
}
