package execution

import (
	"context"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	l "github.com/eggs-gd/fleet.eggs.gd/lib/logger"
)

// FinalizeTaskForExecution normalizes process/session state into an
// ExecutionResult and publishes it for Contour 3.
func (s *Service) FinalizeTaskForExecution(session *RuntimeSession, task Task) {
	if session == nil {
		return
	}
	if sessionNeedsOperatorPause(session) {
		s.pauseSessionForOperatorInput(session, task)
		return
	}

	status := firstNonEmpty(session.ExecutionStatus, tasklifecycle.LegacyExecutionStatus(session.Status, session.ErrorMessage))

	if session.Result != nil && status != "cancelled" {
		if session.ProviderError != nil {
			s.blockProviderErrorTask(*session, task)
			return
		}
		s.publishWorkerExecution(session, task, session.Result, session.LogPath)
		return
	}

	switch status {
	case "provider_error":
		s.blockProviderErrorTask(*session, task)
	case "succeeded":
		s.publishWorkerExecution(session, task, session.Result, session.LogPath)
	case "failed":
		s.PublishExecutionOutcome(*session, task, taskflow.ExecutionFailed,
			tasklifecycle.LaunchProcessFailedComment(session.ErrorMessage, session.LogPath),
			session.ErrorMessage, "")
	case "timed_out":
		s.PublishExecutionOutcome(*session, task, taskflow.ExecutionTimedOut,
			"Agent execution timed out.",
			session.ErrorMessage, session.LogPath)
	case "cancelled":
		s.PublishExecutionOutcome(*session, task, taskflow.ExecutionCancelled,
			"Agent execution was cancelled.",
			session.ErrorMessage, session.LogPath)
	case "orphaned":
		s.PublishExecutionOutcome(*session, task, taskflow.ExecutionOrphaned,
			firstNonEmpty(session.BlockingReason, "Agent execution was orphaned."),
			session.ErrorMessage, session.LogPath)
	}
}

// ApplyWorkerReportedResult publishes a worker JSON outcome as soon as it is
// observed, without waiting for the session to become terminal. HITL
// (needs_input / waiting_input) is a pause: comment, keep doing, hold 1-1-1.
func (s *Service) ApplyWorkerReportedResult(session *RuntimeSession, task Task, result *tasklifecycle.WorkerResult) {
	if session == nil {
		return
	}
	session.Result = result
	session.LastEvent = "worker_result_reported"
	if result != nil && strings.TrimSpace(result.Summary) != "" {
		session.LastMessage = strings.TrimSpace(result.Summary)
	}
	if tasklifecycle.WorkerResultNeedsInput(result) {
		s.pauseSessionForOperatorInput(session, task)
		return
	}
	s.publishWorkerExecution(session, task, result, session.LogPath)
	s.upsertSession(*session)
}

func (s *Service) publishWorkerExecution(session *RuntimeSession, task Task, result *tasklifecycle.WorkerResult, logPath string) {
	if session == nil {
		return
	}
	if logText := ReadSessionLog(s.cfg.RuntimeRoot, firstNonEmpty(logPath, session.LogPath)); logText != "" {
		RefreshSessionToolUsageFromLog(session, logText)
	} else if session.ToolUsage != nil {
		refreshed := executionapi.RefreshToolUsageMissing(*session.ToolUsage, time.Now())
		session.ToolUsage = &refreshed
	}
	s.upsertSession(*session)
	s.publishNormalizedExecutionResult(*session, task, ExecutionResultFromWorkerWithTools(task, result, logPath, session.ToolUsage))
}

// PublishExecutionOutcome publishes a process/runtime outcome for Contour 3.
func (s *Service) PublishExecutionOutcome(session RuntimeSession, task Task, outcome taskflow.ExecutionOutcome, summary, errText, logPath string) {
	s.publishNormalizedExecutionResult(session, task, ExecutionResultFromRuntimeStatus(task, outcome, summary, errText, logPath))
}

func (s *Service) blockProviderErrorTask(session RuntimeSession, task Task) {
	comment := ProviderErrorComment(session)
	if comment == "" {
		comment = tasklifecycle.GenericProviderBlockedComment(session.ErrorMessage, session.LogPath)
	}
	s.PublishExecutionOutcome(session, task, taskflow.ExecutionFailed, comment, session.ErrorMessage, "")
}

// MarkSessionOperatorAttention records an operator-visible comment while a
// live session remains active. It does not publish an ExecutionResult.
func (s *Service) MarkSessionOperatorAttention(session RuntimeSession, task Task, reason string) {
	comment := tasklifecycle.OperatorAttentionComment(reason, session.LogPath)
	_, _ = s.addTaskServiceComment(task, comment)
}

func (s *Service) publishNormalizedExecutionResult(session RuntimeSession, task Task, result taskflow.ExecutionResult) {
	result.TaskID = firstNonEmpty(result.TaskID, task.RelativePath, task.Path, task.ID)
	if strings.TrimSpace(result.ExecutionID) == "" {
		result.ExecutionID = strings.TrimSpace(session.ClaimID)
	}
	if strings.TrimSpace(result.Agent) == "" {
		result.Agent = firstNonEmpty(session.Agent, task.LaunchEvaluation.Agent, task.Launch.Agent, task.Assignee)
	}
	s.PublishExecutionResult(result)
	if s.applyResult == nil {
		return
	}
	if err := s.applyResult(result); err != nil && s.logger != nil {
		s.logger.Warn("contour 3 finalization failed",
			l.String("task", taskLabel(&task)),
			l.String("outcome", string(result.Outcome)),
			l.Error(err))
	}
}

// ReportExecutionResult applies Contour 3 synchronously for tests that
// exercise outcome mapping without running the channel consumer.
func (s *Service) ReportExecutionResult(task Task, result taskflow.ExecutionResult) (Task, error) {
	result.TaskID = firstNonEmpty(result.TaskID, task.RelativePath, task.Path, task.ID)
	if strings.TrimSpace(result.Agent) == "" {
		result.Agent = firstNonEmpty(task.LaunchEvaluation.Agent, task.Launch.Agent, task.Assignee)
	}
	if s.applyResult == nil {
		return Task{}, nil
	}
	if err := s.applyResult(result); err != nil {
		return Task{}, err
	}
	return s.ReloadTaskAfterServiceWrite(firstNonEmpty(result.TaskID, task.RelativePath, task.Path))
}

func (s *Service) ReportWorkerExecution(task Task, result *tasklifecycle.WorkerResult, logPath string) (Task, error) {
	return s.ReportExecutionResult(task, ExecutionResultFromWorker(task, result, logPath))
}

func (s *Service) addTaskServiceComment(task Task, comment string) (Task, error) {
	ctx := context.Background()
	locator := firstNonEmpty(task.RelativePath, task.Path)
	current, err := s.tasks.Get(ctx, locator)
	if err == nil && current.Status != taskflow.StatusDoing {
		loaded := TaskFromFlow(current)
		if s.reload != nil {
			if providerLoaded, loadErr := s.reload(locator); loadErr == nil {
				loaded = providerLoaded
			}
		}
		s.emitTaskEvent("agent_process_status_update_skipped", loaded, "Runtime comment skipped because task already left doing status", map[string]any{
			"target_status": "doing",
		})
		return loaded, nil
	}
	if err := s.tasks.AddComment(ctx, locator, comment); err != nil {
		s.emitTaskEvent("agent_process_status_update_failed", task, err.Error(), map[string]any{
			"target_status": "doing",
		})
		if s.logger != nil {
			s.logger.Warn("agent process status update failed", l.String("task", taskLabel(&task)), l.Error(err))
		}
		return Task{}, err
	}
	updated, err := s.ReloadTaskAfterServiceWrite(locator)
	if err != nil {
		return Task{}, err
	}
	if s.board.UpsertTask != nil {
		s.board.UpsertTask(updated)
	}
	return updated, nil
}

// ReloadTaskAfterServiceWrite reloads the canonical task after a TaskService write.
func (s *Service) ReloadTaskAfterServiceWrite(locator string) (Task, error) {
	if s.reload != nil {
		if loaded, err := s.reload(locator); err == nil {
			return loaded, nil
		}
	}
	flowTask, err := s.tasks.Get(context.Background(), locator)
	if err != nil {
		return Task{}, err
	}
	return TaskFromFlow(flowTask), nil
}

func (s *Service) BlockFailedLaunchTask(session RuntimeSession, task Task) {
	if s.reload != nil {
		if current, err := s.reload(task.RelativePath); err == nil && current.Status != "doing" {
			if s.board.UpsertTask != nil {
				s.board.UpsertTask(current)
			}
			s.emitTaskEvent("agent_process_status_update_skipped", current, "Launch failed after task already left doing status", map[string]any{
				"claim_id":       session.ClaimID,
				"session_status": session.Status,
				"log_path":       session.LogPath,
			})
			return
		}
	}
	outcome := taskflow.ExecutionFailed
	switch firstNonEmpty(session.ExecutionStatus, tasklifecycle.LegacyExecutionStatus(session.Status, session.ErrorMessage)) {
	case "timed_out":
		outcome = taskflow.ExecutionTimedOut
	case "cancelled":
		outcome = taskflow.ExecutionCancelled
	case "orphaned":
		outcome = taskflow.ExecutionOrphaned
	}
	s.PublishExecutionOutcome(session, task, outcome,
		tasklifecycle.LaunchProcessFailedComment(session.ErrorMessage, session.LogPath),
		session.ErrorMessage, "")
}

func (s *Service) BlockNotReadyLaunchTask(task Task, decision LaunchLogClassification) {
	if !tasklifecycle.LaunchablePickupStatus(task.Status) {
		return
	}
	comment := tasklifecycle.LaunchNotReadyComment(decision.Reason)
	if pe := tasklifecycle.ProviderErrorFromLaunchBlocker(
		firstNonEmpty(task.LaunchEvaluation.Agent, task.Launch.Agent, task.Assignee),
		decision.Reason,
	); pe != nil {
		comment = tasklifecycle.ProviderErrorComment(pe, tasklifecycle.ProviderErrorContext{
			Agent: firstNonEmpty(pe.Provider, task.LaunchEvaluation.Agent, task.Assignee),
		})
	}
	locator := firstNonEmpty(task.RelativePath, task.Path)
	blocked, err := s.tasks.Patch(context.Background(), locator, taskflow.PatchInput{
		Status:        taskflow.StatusBlocked,
		Comment:       comment,
		CommentAuthor: tasklifecycle.SystemCommentAuthor,
		Reason:        decision.Reason,
		Actor:         "launcher",
	})
	if err != nil {
		s.emitTaskEvent("launch_not_ready_status_update_failed", task, err.Error(), map[string]any{
			"target_status": "blocked",
			"reason":        decision.Reason,
		})
		if s.logger != nil {
			s.logger.Warn("launch not ready status update failed", l.String("task", taskLabel(&task)), l.Error(err))
		}
		return
	}
	updated := TaskFromFlow(blocked)
	if s.reload != nil {
		if loaded, loadErr := s.reload(locator); loadErr == nil {
			updated = loaded
		}
	}
	if s.board.UpsertTask != nil {
		s.board.UpsertTask(updated)
	}
}
