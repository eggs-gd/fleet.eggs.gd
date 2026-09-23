package execution

import (
	"os"
	"syscall"

	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
)

type sessionRecoveryDecision struct {
	AttachAsSession bool
	Status          string
	ExecutionStatus string
	Reason          string
	ProviderError   *tasklifecycle.ProviderError
	LastEvent       string
	MarkTaskBlocked bool
	// WorkerResult is set when a not-resumable persisted session's transcript
	// contained a recognized worker outcome payload.
	WorkerResult *tasklifecycle.WorkerResult
}

func (s *Service) classifyPersistedSessionRecovery(session RuntimeSession) sessionRecoveryDecision {
	caps := ProviderCapabilityFlags(session.Agent, session.Backend)
	if !session.IsActive() {
		return sessionRecoveryDecision{
			ExecutionStatus: "terminal",
			Status:          "terminal",
			Reason:          "persisted runtime session is already terminal",
			LastEvent:       "persisted_session_terminal",
		}
	}

	switch session.Backend {
	case BackendBackgroundRemote:
		if session.Agent == "claude" || session.Agent == "" {
			validation := s.validatePersistedClaudeBackgroundRemoteSession(session)
			if validation.Resumable {
				return sessionRecoveryDecision{
					AttachAsSession: true,
					ExecutionStatus: "resumable",
					Status:          "resumable",
					LastEvent:       "provider_session_verified",
				}
			}
			return sessionRecoveryDecision{
				Status:          firstNonEmpty(validation.Status, "dead"),
				ExecutionStatus: firstNonEmpty(validation.ExecutionStatus, "dead"),
				Reason:          validation.Reason,
				ProviderError:   validation.ProviderError,
				LastEvent:       firstNonEmpty(validation.LastEvent, "provider_reconciliation_failed"),
				MarkTaskBlocked: true,
				WorkerResult:    validation.WorkerResult,
			}
		}
	case BackendCodexAppServer:
		return classifyProcessBackedRecovery(session, caps, session.CodexThreadID != "", "Codex thread")
	case BackendCursorVisible:
		return classifyProcessBackedRecovery(session, caps, session.CursorChatID != "", "Cursor chat")
	}

	if caps.CanDetectRunningSession.IsUnknown() && caps.CanResumeSession.IsUnknown() {
		return sessionRecoveryDecision{
			ExecutionStatus: "unknown",
			Status:          "unknown",
			Reason:          "provider/backend has unknown restart recovery capabilities; refusing to pretend the session is active",
			LastEvent:       "provider_capability_unknown",
		}
	}
	if session.ProcessID > 0 && processIsRunning(session.ProcessID) {
		return sessionRecoveryDecision{
			AttachAsSession: true,
			ExecutionStatus: "resumable",
			Status:          "resumable",
			LastEvent:       "process_still_running",
		}
	}
	if session.ProcessID > 0 {
		return sessionRecoveryDecision{
			ExecutionStatus: "dead",
			Status:          "dead",
			Reason:          "persisted runtime process is not running after startup",
			LastEvent:       "process_not_running",
		}
	}
	return sessionRecoveryDecision{
		ExecutionStatus: "unknown",
		Status:          "unknown",
		Reason:          "persisted runtime session has no trustworthy live process or provider recovery path",
		LastEvent:       "recovery_identity_missing",
	}
}

func classifyProcessBackedRecovery(session RuntimeSession, caps ProviderCapabilities, hasProviderIdentity bool, identityLabel string) sessionRecoveryDecision {
	if caps.CanDetectRunningSession.IsYes() && processIsRunning(session.ProcessID) {
		return sessionRecoveryDecision{
			AttachAsSession: true,
			ExecutionStatus: "resumable",
			Status:          "resumable",
			LastEvent:       "process_still_running",
		}
	}
	if hasProviderIdentity && caps.ResumeSupported() {
		return sessionRecoveryDecision{
			ExecutionStatus: "resumable",
			Status:          "orphaned",
			Reason:          "persisted " + identityLabel + " identity is still known after restart, but the local process is not running; treat as orphaned but resumable",
			LastEvent:       "provider_identity_resumable",
		}
	}
	if session.ProcessID > 0 && !processIsRunning(session.ProcessID) {
		return sessionRecoveryDecision{
			ExecutionStatus: "dead",
			Status:          "dead",
			Reason:          "persisted runtime session is not resumable after startup: process is dead and no provider resume identity is available",
			LastEvent:       "process_not_running",
		}
	}
	if caps.CanDetectRunningSession.IsUnknown() || caps.CanResumeSession.IsUnknown() {
		return sessionRecoveryDecision{
			ExecutionStatus: "unknown",
			Status:          "unknown",
			Reason:          "provider recovery capabilities are unknown for this backend",
			LastEvent:       "provider_capability_unknown",
		}
	}
	return sessionRecoveryDecision{
		ExecutionStatus: "dead",
		Status:          "dead",
		Reason:          "persisted runtime session is not resumable after startup",
		LastEvent:       "not_resumable",
	}
}

func indexTasksForSessionMatch(tasks []Task) map[string]Task {
	tasksByKey := map[string]Task{}
	for _, task := range tasks {
		for _, key := range compactStrings(task.RelativePath, task.ID, task.Ref) {
			tasksByKey[key] = task
		}
	}
	return tasksByKey
}

func taskForPersistedSession(tasksByKey map[string]Task, session RuntimeSession) (Task, bool) {
	for _, key := range compactStrings(session.TaskPath, session.TaskID, session.TaskRef, session.ClaimID) {
		if task, ok := tasksByKey[key]; ok {
			return task, true
		}
	}
	return Task{}, false
}

func enrichRecoveredSession(session RuntimeSession, task Task) RuntimeSession {
	session.TaskRef = firstNonEmpty(session.TaskRef, task.Ref)
	session.TaskID = firstNonEmpty(session.TaskID, task.ID)
	session.TaskPath = firstNonEmpty(session.TaskPath, task.RelativePath)
	session.TaskTitle = firstNonEmpty(session.TaskTitle, task.Title)
	session.ProjectID = firstNonEmpty(session.ProjectID, task.ProjectID)
	session.Repository = firstNonEmpty(session.Repository, FirstRepository(task))
	session.Agent = firstNonEmpty(session.Agent, task.Assignee)
	return session
}

func compactStrings(values ...string) []string {
	seen := map[string]bool{}
	compacted := []string{}
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		compacted = append(compacted, value)
	}
	return compacted
}

func processIsRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

// ProcessIsRunning reports whether a local process id still accepts signal 0.
func ProcessIsRunning(pid int) bool {
	return processIsRunning(pid)
}
