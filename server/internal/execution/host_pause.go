package execution

import (
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// sessionNeedsOperatorPause is HITL: the worker asked for a human decision,
// or the live session is already waiting. The 1-1-1 slot stays occupied.
func sessionNeedsOperatorPause(session *RuntimeSession) bool {
	if session == nil {
		return false
	}
	if tasklifecycle.WorkerResultNeedsInput(session.Result) {
		return true
	}
	switch firstNonEmpty(session.ExecutionStatus, session.Status) {
	case "waiting_input", "operator_attention", "stalled":
		return true
	default:
		return false
	}
}

func markSessionStatusFromWorkerResult(session *RuntimeSession) {
	if session == nil || session.Result == nil {
		return
	}
	if session.ExecutionStatus == "cancelled" || session.ProviderError != nil {
		return
	}
	if tasklifecycle.WorkerResultNeedsInput(session.Result) {
		session.ExecutionStatus = "waiting_input"
		session.Status = "waiting_input"
		session.ExitedAt = ""
		return
	}
	session.ExecutionStatus = "succeeded"
}

// pauseSessionForOperatorInput keeps the task in doing and the session in
// the hub as waiting_input so later todos in the same project cannot start.
func (s *Service) pauseSessionForOperatorInput(session *RuntimeSession, task Task) {
	if session == nil {
		return
	}
	already := session.LastEvent == "waiting_operator_input" && session.IsActive()
	if tasklifecycle.WorkerResultNeedsInput(session.Result) {
		session.ExecutionStatus = "waiting_input"
		session.Status = "waiting_input"
	} else {
		status := firstNonEmpty(session.ExecutionStatus, session.Status)
		switch status {
		case "waiting_input", "operator_attention", "stalled":
		default:
			session.ExecutionStatus = "waiting_input"
			session.Status = "waiting_input"
		}
	}
	session.ExitedAt = ""
	if session.Result != nil {
		session.BlockingReason = firstNonEmpty(session.Result.Question, session.Result.Summary, session.BlockingReason)
	}
	if strings.TrimSpace(session.BlockingReason) == "" {
		session.BlockingReason = "Waiting on operator input."
	}
	session.LastEvent = "waiting_operator_input"
	s.upsertSession(*session)
	if already {
		return
	}
	comment := tasklifecycle.OperatorAttentionComment(session.BlockingReason, session.LogPath)
	if session.Result != nil {
		_, comment = tasklifecycle.FinalizeSucceededExecution(session.Result, session.LogPath)
	}
	_, _ = s.addTaskServiceComment(task, comment)
}
