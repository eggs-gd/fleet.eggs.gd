package execution

import (
	"testing"

	"github.com/eggs-gd/core.eggs.gd/internal/taskflow"
)

func TestPreviousSessionForTaskFindsMostRecentByClaimID(t *testing.T) {
	root := t.TempDir()

	older := RuntimeSession{
		ClaimID:             "core-13-100",
		TaskRef:             "CORE-13",
		TaskPath:            "Work/core-eggs-gd/tasks/2026-08-01-example.md",
		Backend:             BackendCodexAppServer,
		CodexSessionDetails: CodexSessionDetails{CodexThreadID: "thread-old"},
		ClaimedAt:           "2026-08-01T10:00:00+03:00",
		Status:              "exited",
	}
	newer := RuntimeSession{
		ClaimID:             "core-13-200",
		TaskRef:             "CORE-13",
		TaskPath:            "Work/core-eggs-gd/tasks/2026-08-01-example.md",
		Backend:             BackendCodexAppServer,
		CodexSessionDetails: CodexSessionDetails{CodexThreadID: "thread-new"},
		ClaimedAt:           "2026-08-02T10:00:00+03:00",
		Status:              "exited",
	}
	unrelated := RuntimeSession{
		ClaimID:   "core-14-100",
		TaskRef:   "CORE-14",
		TaskPath:  "Work/core-eggs-gd/tasks/2026-08-01-other.md",
		ClaimedAt: "2026-08-03T10:00:00+03:00",
		Status:    "exited",
	}
	for _, session := range []RuntimeSession{older, newer, unrelated} {
		if err := PersistRuntimeSession(root, session); err != nil {
			t.Fatalf("PersistRuntimeSession(%s): %v", session.ClaimID, err)
		}
	}

	task := Task{Ref: "CORE-13", RelativePath: "Work/core-eggs-gd/tasks/2026-08-01-example.md"}
	found, ok := PreviousSessionForTask(root, task)
	if !ok {
		t.Fatal("expected a previous session for CORE-13")
	}
	if found.ClaimID != "core-13-200" {
		t.Fatalf("previousSessionForTask returned %q, want the newer claim core-13-200", found.ClaimID)
	}

	unmatchedTask := Task{Ref: "CORE-99", RelativePath: "Work/core-eggs-gd/tasks/does-not-exist.md"}
	if _, ok := PreviousSessionForTask(root, unmatchedTask); ok {
		t.Fatal("expected no previous session for an unrelated task ref")
	}
}

// TestApplySessionReuseCodexAttemptsResume covers the "reusable session"
// case: a task with a prior Codex session (which has a resumable thread id)
// gets relaunched, and the launch backend supports an automatic best-effort
// resume attempt.
func TestApplySessionReuseCodexAttemptsResume(t *testing.T) {
	root := t.TempDir()
	
	task := Task{Ref: "CORE-13", ID: "work-2026-08-01-example", RelativePath: "Work/core-eggs-gd/tasks/2026-08-01-example.md"}
	previous := RuntimeSession{
		ClaimID:             "core-13-100",
		TaskRef:             task.Ref,
		TaskID:              task.ID,
		TaskPath:            task.RelativePath,
		Backend:             BackendCodexAppServer,
		CodexSessionDetails: CodexSessionDetails{CodexThreadID: "thread-abc"},
		ClaimedAt:           "2026-08-01T10:00:00+03:00",
		Status:              "exited",
	}
	if err := PersistRuntimeSession(root, previous); err != nil {
		t.Fatalf("persistRuntimeSession: %v", err)
	}

	next := RuntimeSession{ClaimID: "core-13-200", Backend: BackendCodexAppServer, ClaimedAt: "2026-08-02T10:00:00+03:00"}
	ApplySessionReuseAt(root, &next, task, nil)

	if next.SupersedesClaimID != "core-13-100" {
		t.Fatalf("SupersedesClaimID = %q, want core-13-100", next.SupersedesClaimID)
	}
	if next.ReuseCapability != string(SessionReuseUnverified) {
		t.Fatalf("ReuseCapability = %q, want %q", next.ReuseCapability, SessionReuseUnverified)
	}
	if !next.ResumeAttempted {
		t.Fatal("expected ResumeAttempted=true for codex backend with a known prior thread id")
	}
	if next.ResumeOutcome != "attempting" {
		t.Fatalf("ResumeOutcome = %q, want attempting", next.ResumeOutcome)
	}
	if next.CodexThreadID != "thread-abc" {
		t.Fatalf("CodexThreadID = %q, want the prior thread id to be seeded for resume", next.CodexThreadID)
	}

	records, err := LoadRuntimeSessionRecords(root)
	if err != nil {
		t.Fatalf("loadRuntimeSessionRecords: %v", err)
	}
	var reread RuntimeSession
	for _, record := range records {
		if record.ClaimID == "core-13-100" {
			reread = record
		}
	}
	if reread.SupersededByClaimID != "core-13-200" {
		t.Fatalf("previous session SupersededByClaimID = %q, want core-13-200", reread.SupersededByClaimID)
	}
}

// TestApplySessionReuseClaudeDoesNotAttempt covers the "non-reusable
// provider" case: Claude's background-remote backend records the supersede
// link for the dashboard, but must not seed a resume attempt since Core has
// not verified `--resume` reuses the same visible session (CORE-77).
func TestApplySessionReuseClaudeDoesNotAttempt(t *testing.T) {
	root := t.TempDir()
	
	task := Task{Ref: "CORE-13", RelativePath: "Work/core-eggs-gd/tasks/2026-08-01-example.md"}
	previous := RuntimeSession{
		ClaimID:              "core-13-claude-100",
		TaskRef:              task.Ref,
		TaskPath:             task.RelativePath,
		Backend:              "background-remote",
		ClaudeSessionDetails: ClaudeSessionDetails{RemoteControlURL: "https://claude.ai/code/session_abc"},
		ClaimedAt:            "2026-08-01T10:00:00+03:00",
		Status:               "exited",
	}
	if err := PersistRuntimeSession(root, previous); err != nil {
		t.Fatalf("persistRuntimeSession: %v", err)
	}

	next := RuntimeSession{ClaimID: "core-13-claude-200", Backend: "background-remote", ClaimedAt: "2026-08-02T10:00:00+03:00"}
	ApplySessionReuseAt(root, &next, task, nil)

	if next.SupersedesClaimID != "core-13-claude-100" {
		t.Fatalf("SupersedesClaimID = %q, want core-13-claude-100", next.SupersedesClaimID)
	}
	if next.ResumeAttempted {
		t.Fatal("expected ResumeAttempted=false for an unverified/no-attempt backend")
	}
	if next.ResumeOutcome != "not_attempted" {
		t.Fatalf("ResumeOutcome = %q, want not_attempted", next.ResumeOutcome)
	}
	if next.CodexThreadID != "" {
		t.Fatalf("CodexThreadID should stay empty for a non-codex backend, got %q", next.CodexThreadID)
	}
}

func TestApplySessionReuseCursorAttemptsResumeByChatID(t *testing.T) {
	root := t.TempDir()
	
	task := Task{Ref: "CORE-79", ID: "work-core-79", RelativePath: "Work/core-eggs-gd/tasks/2026-08-02-cursor.md"}
	previous := RuntimeSession{
		ClaimID:              "core-79-cursor-100",
		TaskRef:              task.Ref,
		TaskID:               task.ID,
		TaskPath:             task.RelativePath,
		Backend:              cursorVisibleBackend,
		CursorSessionDetails: CursorSessionDetails{CursorChatID: "chat_cursor_123"},
		ClaimedAt:            "2026-08-02T10:00:00+03:00",
		Status:               "exited",
	}
	if err := PersistRuntimeSession(root, previous); err != nil {
		t.Fatalf("persistRuntimeSession: %v", err)
	}

	next := RuntimeSession{ClaimID: "core-79-cursor-200", Backend: cursorVisibleBackend, ClaimedAt: "2026-08-02T11:00:00+03:00"}
	ApplySessionReuseAt(root, &next, task, nil)

	if next.SupersedesClaimID != "core-79-cursor-100" {
		t.Fatalf("SupersedesClaimID = %q, want core-79-cursor-100", next.SupersedesClaimID)
	}
	if next.ReuseCapability != string(SessionReuseVerified) {
		t.Fatalf("ReuseCapability = %q, want %q", next.ReuseCapability, SessionReuseVerified)
	}
	if !next.ResumeAttempted || next.ResumeOutcome != "attempting" {
		t.Fatalf("expected cursor resume attempt, got attempted=%v outcome=%q", next.ResumeAttempted, next.ResumeOutcome)
	}
	if next.CursorChatID != "chat_cursor_123" {
		t.Fatalf("CursorChatID = %q, want prior chat id", next.CursorChatID)
	}
}

// TestApplySessionReuseNoPreviousSession covers a first launch for a task:
// there is nothing to supersede, so the reuse fields must stay zero-valued
// rather than fabricating a link.
func TestApplySessionReuseNoPreviousSession(t *testing.T) {
	root := t.TempDir()
	
	task := Task{Ref: "CORE-999", RelativePath: "Work/core-eggs-gd/tasks/2026-08-01-brand-new.md"}
	next := RuntimeSession{ClaimID: "core-999-1", Backend: BackendCodexAppServer, ClaimedAt: "2026-08-02T10:00:00+03:00"}
	ApplySessionReuseAt(root, &next, task, nil)

	if next.SupersedesClaimID != "" {
		t.Fatalf("SupersedesClaimID = %q, want empty for a task with no previous session", next.SupersedesClaimID)
	}
	if next.ResumeAttempted || next.ResumeOutcome != "" {
		t.Fatalf("expected no resume attempt for a first launch, got attempted=%v outcome=%q", next.ResumeAttempted, next.ResumeOutcome)
	}
}

// TestAssignSessionRolesHandlesSupersedeChainAndLegacyDuplicates covers the
// dashboard grouping requirement: a clean supersede chain marks exactly one
// "current" session. Pre-CORE-77 duplicate data (two unrelated sessions for
// the same task, neither linked) still keeps the older session visible as
// "historical" so the operator notices the anomaly, but only the newest
// non-superseded generation is "current" (CORE-86).
func TestAssignSessionRolesHandlesSupersedeChainAndLegacyDuplicates(t *testing.T) {
	chain := []SessionSummary{
		{RuntimeSession: RuntimeSession{ClaimID: "a", ClaimedAt: "1", Status: "exited", SupersededByClaimID: "b"}},
		{RuntimeSession: RuntimeSession{ClaimID: "b", ClaimedAt: "2", Status: "exited", SupersededByClaimID: "c"}},
		{RuntimeSession: RuntimeSession{ClaimID: "c", ClaimedAt: "3", Status: "running"}},
	}
	assignSessionRoles(chain)
	wantRoles := map[string]string{"a": "superseded", "b": "superseded", "c": "current"}
	for _, session := range chain {
		if got := session.Role; got != wantRoles[session.ClaimID] {
			t.Errorf("session %s role = %q, want %q", session.ClaimID, got, wantRoles[session.ClaimID])
		}
	}

	legacyDuplicates := []SessionSummary{
		{RuntimeSession: RuntimeSession{ClaimID: "3f990304", ClaimedAt: "1", Status: "blocked", ExecutionStatus: "operator_attention"}},
		{RuntimeSession: RuntimeSession{ClaimID: "8920ca20", ClaimedAt: "2", Status: "running", ExecutionStatus: "running"}},
	}
	assignSessionRoles(legacyDuplicates)
	if legacyDuplicates[1].Role != "current" {
		t.Fatalf("newest legacy duplicate role = %q, want current", legacyDuplicates[1].Role)
	}
	if legacyDuplicates[0].Role != "historical" {
		t.Fatalf("older-but-still-active legacy duplicate role = %q, want historical (visible, not a second current)", legacyDuplicates[0].Role)
	}

	allSuperseded := []SessionSummary{
		{RuntimeSession: RuntimeSession{ClaimID: "old", ClaimedAt: "1", Status: "exited", SupersededByClaimID: "gone"}},
	}
	assignSessionRoles(allSuperseded)
	if allSuperseded[0].Role != "superseded" {
		t.Fatalf("sole superseded session role = %q, want superseded (no fabricated current)", allSuperseded[0].Role)
	}
}

func TestBuildSessionGroupsLockedMergesLiveAndPersisted(t *testing.T) {
	root := t.TempDir()
	persisted := RuntimeSession{
		ClaimID:   "core-13-100",
		TaskRef:   "CORE-13",
		TaskPath:  "Work/core-eggs-gd/tasks/2026-08-01-example.md",
		Backend:   BackendCodexAppServer,
		ClaimedAt: "2026-08-01T10:00:00+03:00",
		Status:    "exited",
	}
	if err := PersistRuntimeSession(root, persisted); err != nil {
		t.Fatalf("PersistRuntimeSession: %v", err)
	}

	svc := NewService(ServiceOptions{
		Config: Config{Root: root, DryRun: true},
		Board: BoardHooks{
			ListTasks:  func() []Task { return nil },
			TaskByPath: func(string) (Task, bool) { return Task{}, false },
		},
		In:               make(chan taskflow.TaskEvent),
		Tasks:            make(chan *Task),
		Validated:        make(chan *Task),
		Launched:         make(chan *Task),
		Errch:            make(chan error, 1),
		ExecutionResults: make(chan taskflow.ExecutionResult, 1),
	})
	svc.UpsertSession(RuntimeSession{
		ClaimID:   "core-13-200",
		TaskRef:   "CORE-13",
		TaskPath:  "Work/core-eggs-gd/tasks/2026-08-01-example.md",
		Backend:   BackendCodexAppServer,
		ClaimedAt: "2026-08-02T10:00:00+03:00",
		Status:    "running",
	})

	groups := svc.Status().SessionGroups
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	group := groups[0]
	if group.TaskRef != "CORE-13" {
		t.Fatalf("group.TaskRef = %q, want CORE-13", group.TaskRef)
	}
	if len(group.Sessions) != 2 {
		t.Fatalf("expected 2 sessions in group, got %d", len(group.Sessions))
	}
	if group.Sessions[0].ClaimID != "core-13-100" || group.Sessions[1].ClaimID != "core-13-200" {
		t.Fatalf("sessions not ordered oldest-first: %#v", group.Sessions)
	}
}
