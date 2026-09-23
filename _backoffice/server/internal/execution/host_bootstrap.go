package execution

import (
	"strings"
	"time"

	"github.com/eggs-gd/core.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
	l "github.com/eggs-gd/core.eggs.gd/lib/logger"
)

// ReconcilePersistedSessions reattaches or classifies registry sessions after restart.
func (s *Service) ReconcilePersistedSessions(tasks []Task) error {
	sessions, err := LoadRuntimeSessionRecords(s.cfg.Root)
	if err != nil {
		return err
	}
	tasksByKey := indexTasksForSessionMatch(tasks)
	rehydrated := 0
	ignored := 0
	for _, session := range sessions {
		task, ok := taskForPersistedSession(tasksByKey, session)
		if !ok || !session.IsActive() {
			ignored++
			continue
		}
		decision := s.classifyPersistedSessionRecovery(session)
		session = enrichRecoveredSession(session, task)
		session.Capabilities = ProviderCapabilityFlags(firstNonEmpty(session.Agent, task.Assignee), session.Backend)
		session.VisibilityMode = VisibilityModeForSession(session)
		if task.Status != "doing" {
			s.recordUnresolvedPersistedSession(task, session, decision)
			ignored++
			continue
		}

		switch {
		case decision.AttachAsSession:
			session.Status = firstNonEmpty(decision.Status, session.Status, "resumable")
			session.ExecutionStatus = firstNonEmpty(decision.ExecutionStatus, "resumable")
			session.LastEvent = firstNonEmpty(decision.LastEvent, session.LastEvent, "recovered_after_restart")
			session.ErrorMessage = ""
			session.ProviderError = nil
			s.upsertSession(session)
			s.emitTaskEvent("runtime_session_recovered", task, "Runtime session recovered from persisted session registry", map[string]any{
				"claim_id":         session.ClaimID,
				"backend":          session.Backend,
				"execution_status": session.ExecutionStatus,
				"log_path":         session.LogPath,
				"recovery_state":   session.ExecutionStatus,
			})
			rehydrated++
		default:
			s.applyFailedSessionRecovery(task, session, decision)
			ignored++
		}
	}
	if (rehydrated > 0 || ignored > 0) && s.logger != nil {
		s.logger.Info("persisted sessions reconciled", l.Int("recovered", rehydrated), l.Int("classified", ignored))
	}
	s.backfillPersistedSessionResults(sessions)
	return nil
}

// backfillPersistedSessionResults recovers a missing session.Result from the
// Core session log when a terminal registry record still has result: null.
func (s *Service) backfillPersistedSessionResults(sessions []RuntimeSession) {
	for _, session := range sessions {
		if session.Result != nil || session.IsActive() || strings.TrimSpace(session.ClaimID) == "" {
			continue
		}
		result := tasklifecycle.ParseWorkerResult(ReadSessionLog(s.cfg.Root, session.LogPath))
		if result == nil {
			continue
		}
		session.Result = result
		if session.LastEvent == "" || session.LastEvent == "transcript_updated" {
			session.LastEvent = "worker_result_backfilled"
		}
		_ = persistRuntimeSession(s.cfg.Root, session)
	}
}

func (s *Service) applyFailedSessionRecovery(task Task, session RuntimeSession, decision sessionRecoveryDecision) {
	session.Status = firstNonEmpty(decision.Status, "orphaned")
	session.ExecutionStatus = firstNonEmpty(decision.ExecutionStatus, "orphaned")
	session.ErrorMessage = decision.Reason
	session.LastEvent = firstNonEmpty(decision.LastEvent, "provider_reconciliation_failed")
	session.ProviderError = decision.ProviderError
	if session.ExitedAt == "" && (decision.ExecutionStatus == "dead" || decision.ExecutionStatus == "terminal" || decision.MarkTaskBlocked) {
		session.ExitedAt = time.Now().Format(time.RFC3339)
	}

	if decision.WorkerResult != nil {
		session.Result = decision.WorkerResult
		if !tasklifecycle.WorkerResultNeedsInput(decision.WorkerResult) {
			session.Status = "exited"
			session.ExecutionStatus = "succeeded"
			if session.ExitedAt == "" {
				session.ExitedAt = time.Now().Format(time.RFC3339)
			}
		}
		s.ApplyWorkerReportedResult(&session, task, decision.WorkerResult)
		s.emitTaskEvent("runtime_worker_result_recovered_after_restart", task, decision.Reason, map[string]any{
			"claim_id":      session.ClaimID,
			"backend":       session.Backend,
			"background_id": session.BackgroundID,
			"outcome":       decision.WorkerResult.Outcome,
		})
		return
	}

	if decision.MarkTaskBlocked {
		s.upsertSession(session)
		comment := staleClaudeBackgroundSessionComment(session)
		s.PublishExecutionOutcome(session, task, taskflow.ExecutionOrphaned, comment, session.ErrorMessage, session.LogPath)
		s.emitTaskEvent("runtime_provider_session_stale", task, decision.Reason, map[string]any{
			"claim_id":         session.ClaimID,
			"backend":          session.Backend,
			"background_id":    session.BackgroundID,
			"execution_status": session.ExecutionStatus,
			"provider_error":   session.ProviderError,
			"recovery_state":   session.ExecutionStatus,
		})
	} else {
		s.emitTaskEvent("runtime_orphan_detected", task, decision.Reason, map[string]any{
			"claim_id":         session.ClaimID,
			"backend":          session.Backend,
			"execution_status": session.ExecutionStatus,
			"process_id":       session.ProcessID,
			"log_path":         session.LogPath,
			"recovery_state":   session.ExecutionStatus,
			"thread_id":        session.ProviderThreadID(),
			"session_id":       session.ProviderSessionID(),
		})
	}

	s.recordOrphan(task, session, session.ExecutionStatus, decision.Reason)
}

// recordUnresolvedPersistedSession classifies a leftover registry session after
// the task already left doing. A still-running process is ignored (the task
// already moved on). A dead/unresumable leftover becomes an orphan for the
// operator list. The task itself is not mutated.
func (s *Service) recordUnresolvedPersistedSession(task Task, session RuntimeSession, decision sessionRecoveryDecision) {
	if decision.WorkerResult != nil {
		session.Result = decision.WorkerResult
		session.Status = "exited"
		session.ExecutionStatus = "succeeded"
		if session.ExitedAt == "" {
			session.ExitedAt = time.Now().Format(time.RFC3339)
		}
		session.LastEvent = firstNonEmpty(decision.LastEvent, "worker_result_recovered_after_restart")
		_ = persistRuntimeSession(s.cfg.Root, session)
		return
	}
	if decision.AttachAsSession {
		return
	}
	session.Status = firstNonEmpty(decision.Status, "orphaned")
	session.ExecutionStatus = firstNonEmpty(decision.ExecutionStatus, "orphaned")
	session.ErrorMessage = decision.Reason
	session.LastEvent = firstNonEmpty(decision.LastEvent, "provider_reconciliation_failed")
	session.ProviderError = decision.ProviderError
	_ = persistRuntimeSession(s.cfg.Root, session)
	s.recordOrphan(task, session, session.ExecutionStatus, firstNonEmpty(decision.Reason, "runtime session is not active"))
	s.emitTaskEvent("runtime_orphan_detected", task, firstNonEmpty(decision.Reason, "runtime session is not active"), map[string]any{
		"claim_id":         session.ClaimID,
		"backend":          session.Backend,
		"execution_status": session.ExecutionStatus,
		"process_id":       session.ProcessID,
		"log_path":         session.LogPath,
		"recovery_state":   session.ExecutionStatus,
		"task_status":      task.Status,
	})
}

// DetectStartupOrphans records doing tasks with no recoverable session.
func (s *Service) DetectStartupOrphans(tasks []Task) {
	detectedAt := time.Now().Format(time.RFC3339)
	count := 0
	for _, task := range tasks {
		if s.board.TaskByPath != nil {
			if current, ok := s.board.TaskByPath(task.RelativePath); ok {
				task = current
			}
		}
		if task.Status != "doing" {
			continue
		}
		if _, ok := s.activeSessionForTask(task); ok {
			continue
		}
		if s.hasOrphanForTask(task) {
			continue
		}
		if session, ok := previousSessionForTask(s.cfg.Root, task); ok && session.IsActive() {
			decision := s.classifyPersistedSessionRecovery(session)
			session = enrichRecoveredSession(session, task)
			session.Capabilities = ProviderCapabilityFlags(firstNonEmpty(session.Agent, task.Assignee), session.Backend)
			session.VisibilityMode = VisibilityModeForSession(session)
			if decision.AttachAsSession {
				session.Status = firstNonEmpty(decision.Status, "resumable")
				session.ExecutionStatus = firstNonEmpty(decision.ExecutionStatus, "resumable")
				session.LastEvent = firstNonEmpty(decision.LastEvent, "recovered_after_restart")
				s.upsertSession(session)
				continue
			}
			s.applyFailedSessionRecovery(task, session, decision)
			count++
			continue
		}
		session := RuntimeSession{
			ClaimedAt: detectedAt,
			Agent:     task.Assignee,
			Backend:   "",
		}
		caps := ProviderCapabilityFlags(task.Assignee, "")
		session.Capabilities = caps
		state := "orphaned"
		reason := "task is doing but runtime has no active session after startup"
		if caps.CanDetectRunningSession.IsUnknown() && caps.CanResumeSession.IsUnknown() {
			state = "unknown"
			reason = "task is doing with no persisted session metadata and unknown provider recovery capabilities"
		}
		s.recordOrphan(task, session, state, reason)
		s.emitTaskEvent("runtime_orphan_detected", task, reason, map[string]any{
			"detected_at":    detectedAt,
			"repository":     FirstRepository(task),
			"recovery_state": state,
			"provider":       firstNonEmpty(session.Agent, task.Assignee),
		})
		count++
	}
	if count > 0 && s.logger != nil {
		s.logger.Warn("orphaned doing tasks detected", l.Int("tasks", count))
	}
}
