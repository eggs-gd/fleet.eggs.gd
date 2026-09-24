package execution

import (
	"context"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	l "github.com/eggs-gd/fleet.eggs.gd/lib/logger"
)

// SessionSummary is a RuntimeSession annotated with its role within the
// group of sessions launched for the same task ref, for dashboard grouping.
type SessionSummary struct {
	RuntimeSession
	// Role is "current" (exactly one: the newest non-superseded generation
	// for this task), "historical" (older, not linked into the supersede
	// chain — including active legacy duplicates), or "superseded"
	// (explicitly replaced by a later session).
	Role string `json:"role"`
	// Resumable reflects whether this session's backend has a verified
	// automatic-reuse capability, i.e. whether reopening this task again
	// could continue this exact session rather than creating another one.
	Resumable bool   `json:"resumable"`
	ReuseNote string `json:"reuse_note,omitempty"`
}

// SessionGroup collects every known runtime session (active and historical)
// for a single task ref/id/path, so the dashboard can show current vs.
// historical vs. superseded sessions per task instead of a flat list.
type SessionGroup struct {
	TaskRef  string           `json:"task_ref"`
	TaskID   string           `json:"task_id"`
	TaskPath string           `json:"task_path"`
	Sessions []SessionSummary `json:"sessions"`
}

// OrphanedTask is a lost runtime session the operator still needs to resolve:
// a doing task with no live session, or a leftover persisted session that
// still looks unresolved after the task left doing.
type OrphanedTask struct {
	TaskRef        string                       `json:"task_ref"`
	TaskID         string                       `json:"task_id"`
	TaskPath       string                       `json:"task_path"`
	ProjectID      string                       `json:"project_id"`
	Repository     string                       `json:"repository"`
	Assignee       string                       `json:"assignee"`
	ClaimID        string                       `json:"claim_id,omitempty"`
	ExecutionState string                       `json:"execution_state"`
	Provider       string                       `json:"provider,omitempty"`
	Backend        string                       `json:"backend,omitempty"`
	VisibilityMode string                       `json:"visibility_mode"`
	ProcessID      int                          `json:"process_id,omitempty"`
	ThreadID       string                       `json:"thread_id,omitempty"`
	SessionID      string                       `json:"session_id,omitempty"`
	TurnID         string                       `json:"turn_id,omitempty"`
	LogPath        string                       `json:"log_path,omitempty"`
	LastEvent      string                       `json:"last_event,omitempty"`
	LastActivityAt string                       `json:"last_activity_at,omitempty"`
	DetectedAt     string                       `json:"detected_at"`
	Reason         string                       `json:"reason"`
	BlockingReason string                       `json:"blocking_reason"`
	ProviderError  *tasklifecycle.ProviderError `json:"provider_error,omitempty"`
	Capabilities   ProviderCapabilities         `json:"capabilities,omitempty"`
}

// Status is execution's black-box projection of live runtime sessions,
// session history groups, and orphaned "doing" tasks for operator poll APIs.
type Status struct {
	RuntimeSessions []RuntimeSession `json:"runtime_sessions"`
	SessionGroups   []SessionGroup   `json:"session_groups"`
	OrphanedTasks   []OrphanedTask   `json:"orphaned_tasks"`
}

// Status snapshots the execution-owned session/orphan hub.
func (s *Service) Status() Status {
	if s == nil || s.hub == nil {
		return Status{}
	}
	return s.hub.status()
}

// AnnotateTask revalidates a task's launch-wait reason against the current
// session/orphan hub and sets task.Execution from the active session or
// orphan backing it, mirroring corechain.Store's per-task projection
// (revalidateLaunchWaitingLocked + executionForTaskLocked).
// When a stale wait clears with no current blocker, the task is persisted and
// requeued so clearing metadata alone cannot leave a launchable task idle.
// Pickup tasks with depends_on also refresh that gate against the live board
// so /api/state explains dependency waiting without mutating status.
func (s *Service) AnnotateTask(task tasklifecycle.Task) tasklifecycle.Task {
	if s == nil {
		return task
	}
	updated := task
	if s.hub != nil {
		before := task.LaunchEvaluation.Waiting
		var changed bool
		updated, changed = s.hub.revalidateLaunchWaiting(task)
		if changed {
			message := "replaced stale launch waiting reason with current blocker"
			details := map[string]any{"stale_wait": before}
			if updated.LaunchEvaluation.Waiting == nil {
				message = "cleared stale launch waiting reason with no current blocker"
			} else {
				details["current_wait"] = updated.LaunchEvaluation.Waiting
			}
			if s.board.UpsertTask != nil {
				s.board.UpsertTask(updated)
			}
			s.emitTaskEvent("launch_waiting_stale_cleared", updated, message, details)
			if s.logger != nil {
				s.logger.Info("stale wait cleared",
					l.String("task", taskLabel(&updated)),
					l.String("message", message))
			}
			if updated.LaunchEvaluation.Waiting == nil && s.cfg.AllowsProcessStart() {
				go s.requeueAfterStaleWaitCleared(updated, before)
			}
		}
		updated.Execution = s.hub.executionForTask(updated)
	}
	return s.refreshDependsOnEvaluation(updated)
}

func (s *Service) refreshDependsOnEvaluation(task tasklifecycle.Task) tasklifecycle.Task {
	if len(tasklifecycle.NormalizeDependsOn(task.DependsOn)) == 0 {
		return task
	}
	if !tasklifecycle.LaunchablePickupStatus(task.Status) {
		return task
	}
	known := []Task{}
	if s.board.ListTasks != nil {
		known = s.board.ListTasks()
	}
	unresolved := tasklifecycle.UnresolvedDependencies(task, known)

	hadEval := len(task.LaunchEvaluation.FailedGates) > 0 || len(task.LaunchEvaluation.PassedGates) > 0
	gates := make([]string, 0, len(task.LaunchEvaluation.FailedGates))
	for _, gate := range task.LaunchEvaluation.FailedGates {
		if strings.HasPrefix(gate, "depends_on:") {
			continue
		}
		gates = append(gates, gate)
	}
	passed := make([]string, 0, len(task.LaunchEvaluation.PassedGates))
	for _, step := range task.LaunchEvaluation.PassedGates {
		if step == "depends_on" {
			continue
		}
		passed = append(passed, step)
	}
	task.LaunchEvaluation.FailedGates = gates
	task.LaunchEvaluation.PassedGates = passed

	if len(unresolved) > 0 {
		task.FailLaunchGate("depends_on", strings.Join(unresolved, "; "))
		task.ResolveLaunchEvaluation()
		return task
	}
	if !hadEval {
		// Do not invent a full launchable outcome from depends_on alone.
		return task
	}
	task.Pass("depends_on")
	task.ResolveLaunchEvaluation()
	return task
}

func (s *Service) requeueAfterStaleWaitCleared(task Task, before *tasklifecycle.LaunchWait) {
	ended := RuntimeSession{
		ClaimID:         "stale_wait_cleared",
		TaskRef:         task.Ref,
		TaskID:          task.ID,
		TaskPath:        task.RelativePath,
		ProjectID:       task.ProjectID,
		Repository:      FirstRepository(task),
		Agent:           firstNonEmpty(task.LaunchEvaluation.Agent, task.Launch.Agent, task.Assignee),
		Status:          "released",
		ExecutionStatus: "released",
		ErrorMessage:    "stale launch wait cleared with no current blocker",
	}
	if before != nil {
		if before.ConflictClaimID != "" {
			ended.ClaimID = before.ConflictClaimID
		}
		switch before.Scope {
		case "assignee", "assignee_orphan":
			ended.Agent = firstNonEmpty(before.Resource, ended.Agent)
		case "project":
			ended.ProjectID = firstNonEmpty(before.Resource, ended.ProjectID)
		case "repository":
			ended.Repository = firstNonEmpty(before.Resource, ended.Repository)
		}
	}
	s.emitTaskEvent("launch_wait_requeued", task, "Stale launch wait cleared; task returned to launch evaluation", map[string]any{
		"stage":             "stale-wait-clear",
		"released_claim_id": ended.ClaimID,
	})
	s.RequeueReleasedSlot(context.Background(), ended)
}
