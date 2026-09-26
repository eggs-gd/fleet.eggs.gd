package execution

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// SessionControlPatch is an operator request against a runtime session.
type SessionControlPatch struct {
	ClaimID      string `json:"claim_id"`
	Action       string `json:"action"`
	Input        string `json:"input"`
	TargetStatus string `json:"target_status,omitempty"`
}

const (
	sessionActionRelease                      = "release"
	sessionActionMarkDead                     = "mark_dead"
	sessionActionMarkProviderUnavailable      = "mark_provider_unavailable"
	sessionActionCancelWithoutProviderControl = "cancel_without_provider_control"
)

func IsOperatorReleaseAction(action string) bool {
	switch strings.TrimSpace(strings.ToLower(action)) {
	case sessionActionRelease, sessionActionMarkDead, sessionActionMarkProviderUnavailable, sessionActionCancelWithoutProviderControl:
		return true
	default:
		return false
	}
}

func operatorReleaseExecutionStatus(action string) string {
	switch strings.TrimSpace(strings.ToLower(action)) {
	case sessionActionMarkDead:
		return "dead"
	case sessionActionMarkProviderUnavailable:
		return "provider_unavailable"
	case sessionActionCancelWithoutProviderControl:
		return "cancelled"
	default:
		return "released"
	}
}

// ControlSession handles live provider control and operator release actions.
func (s *Service) ControlSession(patch SessionControlPatch) error {
	action := strings.TrimSpace(strings.ToLower(patch.Action))
	if action == "" {
		return errors.New("session control action is required")
	}
	if strings.TrimSpace(patch.ClaimID) == "" {
		return errors.New("claim_id is required")
	}
	if IsOperatorReleaseAction(action) {
		return s.releaseRuntimeSession(patch)
	}
	return s.SendSessionControl(patch.ClaimID, action, patch.Input)
}

func (s *Service) releaseRuntimeSession(patch SessionControlPatch) error {
	claimID := strings.TrimSpace(patch.ClaimID)
	if claimID == "" {
		return errors.New("claim_id is required to release a runtime session")
	}
	action := strings.TrimSpace(strings.ToLower(patch.Action))
	if !IsOperatorReleaseAction(action) {
		return fmt.Errorf("unsupported operator release action %q", patch.Action)
	}

	session, ok := s.lookupSessionForRelease(claimID)
	if !ok {
		return fmt.Errorf("runtime session %q not found", claimID)
	}

	targetStatus := strings.TrimSpace(patch.TargetStatus)
	if targetStatus == "" {
		targetStatus = tasklifecycle.OperatorReleaseDefaultTaskStatus(action)
	}
	if !tasklifecycle.AllowedOperatorReleaseTaskStatus(targetStatus) {
		return fmt.Errorf("target_status must be needs_review, needs_rework, or blocked (got %q)", targetStatus)
	}

	executionStatus := operatorReleaseExecutionStatus(action)
	now := time.Now().Format(time.RFC3339)
	session.Status = executionStatus
	session.ExecutionStatus = executionStatus
	session.LastEvent = "operator_" + action
	session.LastEventAt = now
	session.LastStatusAt = now
	session.ExitedAt = firstNonEmpty(session.ExitedAt, now)
	session.BlockingReason = ""
	session.ErrorMessage = firstNonEmpty(strings.TrimSpace(patch.Input), tasklifecycle.OperatorReleaseReason(action, claimID))
	if action == sessionActionMarkProviderUnavailable && session.ProviderError == nil {
		session.ProviderError = tasklifecycle.NewProviderError(firstNonEmpty(session.Agent, "unknown"), "provider_unavailable", "operator release", session.ErrorMessage, "")
	}

	s.unregisterSessionControl(claimID)
	s.upsertSession(session)
	s.clearOrphanForSession(session)

	task, taskOK := s.loadTaskForSession(session)
	comment := tasklifecycle.OperatorReleaseComment(action, session.ClaimID, firstNonEmpty(session.ExecutionStatus, session.Status), session.LogPath, targetStatus)
	if taskOK && task.Status == "doing" {
		if !tasklifecycle.AllowedTaskStatusTransition(task.Status, targetStatus) {
			return fmt.Errorf("%w: cannot move task from %q to %q", tasklifecycle.ErrTaskTransitionRejected, task.Status, targetStatus)
		}
		locator := firstNonEmpty(task.RelativePath, task.Path)
		flowUpdated, err := s.tasks.Patch(context.Background(), locator, taskflow.PatchInput{
			Status:        taskflow.Status(targetStatus),
			Comment:       comment,
			CommentAuthor: "owner",
			Actor:         "owner",
			Reason:        session.ErrorMessage,
		})
		if err != nil {
			return err
		}
		updated := TaskFromFlow(flowUpdated)
		if s.reload != nil {
			if loaded, loadErr := s.reload(locator); loadErr == nil {
				updated = loaded
			}
		}
		if s.board.UpsertTask != nil {
			s.board.UpsertTask(updated)
		}
		task = updated
	} else if taskOK {
		locator := firstNonEmpty(task.RelativePath, task.Path)
		if err := s.tasks.AddComment(
			taskflow.WithCommentAuthor(context.Background(), "owner"),
			locator,
			comment,
		); err == nil {
			if updated, loadErr := s.ReloadTaskAfterServiceWrite(locator); loadErr == nil {
				if s.board.UpsertTask != nil {
					s.board.UpsertTask(updated)
				}
				task = updated
			}
		}
	}

	details := map[string]any{
		"claim_id":         session.ClaimID,
		"action":           action,
		"execution_status": session.ExecutionStatus,
		"target_status":    targetStatus,
		"backend":          session.Backend,
		"agent":            session.Agent,
		"task_ref":         session.TaskRef,
		"task_path":        session.TaskPath,
		"log_path":         session.LogPath,
	}
	if taskOK {
		s.emitTaskEvent("runtime_session_operator_released", task, session.ErrorMessage, details)
	} else {
		s.emitRuntimeEvent("runtime_session_operator_released", "", session.ErrorMessage, details)
	}

	s.RequeueReleasedSlot(context.Background(), session)
	return nil
}

// ReleaseActiveSessionForTask terminalizes a live/HITL session when the
// operator moves the task off doing. The queue can start the next todo.
func (s *Service) ReleaseActiveSessionForTask(task Task, reason string) {
	if s == nil {
		return
	}
	session, ok := s.ActiveSessionForTask(task)
	if !ok {
		session = RuntimeSessionForReleasedTask(task, "orphan_released", reason)
		s.clearOrphanForSession(session)
		s.RequeueReleasedSlot(context.Background(), session)
		return
	}
	now := time.Now().Format(time.RFC3339)
	session.Status = "released"
	session.ExecutionStatus = "released"
	session.LastEvent = "task_left_doing"
	session.LastEventAt = now
	session.LastStatusAt = now
	session.ExitedAt = firstNonEmpty(session.ExitedAt, now)
	session.ErrorMessage = firstNonEmpty(reason, "doing task left active-session status")
	s.unregisterSessionControl(session.ClaimID)
	s.upsertSession(session)
	s.clearOrphanForSession(session)
	s.RequeueReleasedSlot(context.Background(), session)
}

func (s *Service) lookupSessionForRelease(claimID string) (RuntimeSession, bool) {
	if session, ok := s.session(claimID); ok {
		return session, true
	}
	sessions, err := LoadRuntimeSessionRecords(s.cfg.RuntimeRoot)
	if err == nil {
		for _, session := range sessions {
			if session.ClaimID == claimID {
				return session, true
			}
		}
	}
	if session, ok := s.orphanSession(claimID); ok {
		return session, true
	}
	return RuntimeSession{}, false
}

func (s *Service) loadTaskForSession(session RuntimeSession) (Task, bool) {
	if session.TaskPath != "" {
		if s.reload != nil {
			if task, err := s.reload(session.TaskPath); err == nil {
				return task, true
			}
		}
		if s.board.TaskByPath != nil {
			if task, ok := s.board.TaskByPath(session.TaskPath); ok {
				return task, true
			}
		}
	}
	if s.board.ListTasks == nil {
		return Task{}, false
	}
	for _, task := range s.board.ListTasks() {
		if sameTaskIdentity(session.TaskPath, task.RelativePath, session.TaskID, task.ID, session.TaskRef, task.Ref) {
			return task, true
		}
	}
	return Task{}, false
}

// SessionReleaseEligible reports whether an operator release control should be
// offered for the session when live provider control is unavailable.
func SessionReleaseEligible(session RuntimeSession) bool {
	state := firstNonEmpty(session.ExecutionStatus, session.Status)
	switch state {
	case "queued", "claimed", "starting", "waiting_for_visible_session", "running", "waiting_input", "operator_attention", "stalled", "resumable",
		"dead", "provider_unavailable", "control_socket_unavailable", "orphaned", "unknown":
		return session.ClaimID != ""
	default:
		return false
	}
}
