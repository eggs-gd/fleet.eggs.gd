package execution

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	l "github.com/eggs-gd/fleet.eggs.gd/lib/logger"
)

func (s *Service) StartLaunchCandidate(ctx context.Context, candidate Task) {
	s.startLaunchCandidate(ctx, candidate)
}

func (s *Service) startLaunchCandidate(ctx context.Context, candidate Task) {
	session := runtimeSessionForTask(candidate, s.cfg)
	if conflict, scope, ok := s.tryReserveSession(session); !ok {
		resource := SessionConflictResource(scope, candidate)
		reason := LaunchConflictReason(conflict, scope, resource)
		candidate.MarkLaunchWaiting(tasklifecycle.LaunchWait{
			Scope:           scope,
			Resource:        resource,
			ConflictClaimID: conflict.ClaimID,
			Reason:          reason,
			WaitingSince:    time.Now().Format(time.RFC3339),
		})
		if s.board.UpsertTask != nil {
			s.board.UpsertTask(candidate)
		}
		s.emitTaskEvent("launch_waiting", candidate, reason, map[string]any{
			"stage":             "session-gate",
			"conflict_claim_id": conflict.ClaimID,
			"scope":             scope,
			"conflict_status":   firstNonEmpty(conflict.ExecutionStatus, conflict.Status),
			"provider_startup":  conflict.IsProviderStartup(),
		})
		if s.logger != nil {
			if conflict.IsProviderStartup() {
				s.logger.Warn("waiting on in-progress provider startup",
					l.String("task", taskLabel(&candidate)),
					l.String("stage", "session-gate"),
					l.String("scope", scope),
					l.String("conflict_claim_id", conflict.ClaimID),
					l.String("conflict_status", firstNonEmpty(conflict.ExecutionStatus, conflict.Status)),
					l.String("reason", reason))
			} else {
				s.logger.Warn("waiting due to active session", l.String("task", taskLabel(&candidate)), l.String("stage", "session-gate"), l.String("reason", reason))
			}
		}
		return
	}
	candidate.ClearLaunchWaiting()
	if s.board.UpsertTask != nil {
		s.board.UpsertTask(candidate)
	}
	s.emitTaskEvent("runtime_slot_reserved", candidate, "Launch slot reserved before provider start", map[string]any{
		"stage":      "session-gate",
		"claim_id":   session.ClaimID,
		"agent":      session.Agent,
		"project_id": session.ProjectID,
		"repository": session.Repository,
		"status":     firstNonEmpty(session.ExecutionStatus, session.Status),
	})
	if s.logger != nil {
		s.logger.Info("launch slot reserved before provider start",
			l.String("task", taskLabel(&candidate)),
			l.String("claim_id", session.ClaimID),
			l.String("agent", session.Agent),
			l.String("project", session.ProjectID),
			l.String("repository", session.Repository))
	}

	locator := firstNonEmpty(candidate.RelativePath, candidate.Path)
	flowClaimed, err := s.tasks.Claim(ctx, locator)
	if err != nil {
		s.removeSession(session.ClaimID)
		s.emitTaskEvent("launch_claim_failed", candidate, err.Error(), map[string]any{
			"stage":    "claim",
			"claim_id": session.ClaimID,
		})
		if s.logger != nil {
			s.logger.Warn("launch claim failed", l.String("task", taskLabel(&candidate)), l.Error(err))
		}
		return
	}
	claimed := TaskFromFlow(flowClaimed)
	if s.reload != nil {
		if loaded, loadErr := s.reload(locator); loadErr == nil {
			claimed = loaded
		}
	}
	claimed.LaunchEvaluation = candidate.LaunchEvaluation
	s.emitTaskEvent("launch_claimed", claimed, "Task claimed for agent start", map[string]any{
		"stage":    "claim",
		"claim_id": session.ClaimID,
		"agent":    claimed.LaunchEvaluation.Agent,
	})
	if s.logger != nil {
		s.logger.Info("claimed",
			l.String("task", taskLabel(&claimed)),
			l.String("claim_id", session.ClaimID),
			l.String("agent", claimed.LaunchEvaluation.Agent))
	}

	session.TaskRef = claimed.Ref
	session.TaskID = claimed.ID
	session.TaskPath = claimed.RelativePath
	session.TaskTitle = claimed.Title
	session.ProjectID = claimed.ProjectID
	session.Repository = claimed.LaunchEvaluation.Repository
	session.Agent = claimed.LaunchEvaluation.Agent
	session.Backend = claimed.LaunchEvaluation.Backend
	session.VisibilityMode = firstNonEmpty(claimed.LaunchEvaluation.VisibilityMode, VisibilityModeForSession(session))
	session.Capabilities = ProviderCapabilityFlags(session.Agent, session.Backend)
	if session.VisibilityMode == "" || session.VisibilityMode == string(VisibilityUnknown) {
		session.VisibilityMode = firstNonEmpty(session.Capabilities.OperatorVisibility, VisibilityModeForSession(session))
	}
	session.Command = append([]string{}, claimed.LaunchEvaluation.Command...)
	session.WorkingDir = claimed.LaunchEvaluation.WorkingDir
	if session.Backend == BackendCodexAppServer {
		session.CodexThreadTitle = CodexRemoteThreadTitle(claimed.Ref, claimed.Title, claimed.ID)
	}
	s.applySessionReuse(&session, claimed)
	InitSessionToolUsage(&session, claimed)
	s.upsertSession(session)
	s.emitTaskEvent("agent_process_starting", claimed, "Agent process launch starting", map[string]any{
		"claim_id":            session.ClaimID,
		"launcher":            session.Launcher,
		"command":             session.Command,
		"log_path":            session.LogPath,
		"session_timeout":     s.cfg.EffectiveSessionTimeout().String(),
		"supersedes_claim_id": session.SupersedesClaimID,
		"reuse_capability":    session.ReuseCapability,
		"resume_attempted":    session.ResumeAttempted,
	})

	if claimed.LaunchEvaluation.Backend == BackendBackgroundRemote {
		s.spawnRunner(RunBackgroundRemoteSession, ctx, session, claimed)
		return
	}
	if claimed.LaunchEvaluation.Backend == BackendCodexAppServer {
		s.spawnRunner(RunCodexAppServerSession, ctx, session, claimed)
		return
	}
	if claimed.LaunchEvaluation.Backend == BackendCursorVisible {
		s.spawnRunner(RunCursorVisibleSession, ctx, session, claimed)
		return
	}
	if claimed.LaunchEvaluation.Backend == BackendGeminiHeadless {
		s.spawnRunner(RunGeminiHeadlessSession, ctx, session, claimed)
		return
	}

	process, err := StartProcess(ctx, Plan{
		Agent:        session.Agent,
		Backend:      claimed.LaunchEvaluation.Backend,
		Command:      session.Command,
		WorkingDir:   session.WorkingDir,
		InitialInput: claimed.LaunchEvaluation.InitialInput,
	}, filepath.Join(s.cfg.RuntimeRoot, session.LogPath), s.cfg.EffectiveSessionTimeout())
	if err != nil {
		s.FailLaunchSession(session, claimed, err)
		return
	}

	session.ProcessID = process.Cmd.Process.Pid
	session.StartedAt = time.Now().Format(time.RFC3339)
	session.Status = "running"
	session.ExecutionStatus = "running"
	s.upsertSession(session)
	s.emitTaskEvent("agent_process_started", claimed, "Agent process started", map[string]any{
		"claim_id":   session.ClaimID,
		"process_id": session.ProcessID,
		"launcher":   session.Launcher,
		"log_path":   session.LogPath,
	})
	if s.logger != nil {
		s.logger.Info("agent process started", l.String("task", taskLabel(&claimed)), l.String("claim_id", session.ClaimID), l.Int("process_id", session.ProcessID))
	}

	s.spawn(func(session RuntimeSession, claimed Task) func() {
		return func() { s.waitLaunchSession(ctx, process, session, claimed) }
	}(session, claimed))
}

func (s *Service) runnerOptions() RunnerOptions {
	return RunnerOptions{
		RuntimeRoot:    s.cfg.RuntimeRoot,
		SessionTimeout: s.cfg.EffectiveSessionTimeout(),
		Callbacks: RunnerCallbacks{
			ClassifyProviderError:      ClassifyProviderError,
			FailLaunchSession:          s.FailLaunchSession,
			RecordProviderSessionStart: s.RecordProviderSessionStart,
			CompleteProviderSession:    s.CompleteProviderSession,
			UpsertSession:              s.upsertSession,
			AddTaskComment: func(locator string, comment string) error {
				return s.tasks.AddComment(context.Background(), locator, comment)
			},
			MarkSessionOperatorAttention: s.MarkSessionOperatorAttention,
			ApplyWorkerReportedResult:    s.ApplyWorkerReportedResult,
			RegisterSessionControl:       s.registerSessionControl,
			UnregisterSessionControl:     s.unregisterSessionControl,
		},
	}
}

// ApplySessionReuse links a new session to the most recent prior session for the task.
func (s *Service) ApplySessionReuse(session *RuntimeSession, task Task) {
	if s == nil {
		return
	}
	ApplySessionReuseAt(s.cfg.RuntimeRoot, session, task, s.emitTaskEvent)
}

func (s *Service) applySessionReuse(session *RuntimeSession, task Task) {
	s.ApplySessionReuse(session, task)
	s.applyAgentProjectSessionReuse(session, task)
}

// applyAgentProjectSessionReuse is the 1-1-1 fallback (see
// ApplyAgentProjectSessionReuse) used when this task has no same-task prior
// session to resume.
func (s *Service) applyAgentProjectSessionReuse(session *RuntimeSession, task Task) {
	if s == nil {
		return
	}
	ApplyAgentProjectSessionReuse(s.cfg.RuntimeRoot, session, task, s.emitTaskEvent)
}

// ApplySessionReuseAt implements CORE-77 relaunch reuse without requiring a Service.
func ApplySessionReuseAt(root string, session *RuntimeSession, task Task, emit func(eventType string, task Task, message string, details map[string]any)) {
	if session == nil {
		return
	}
	previous, ok := previousSessionForTask(root, task)
	if !ok {
		return
	}

	session.SupersedesClaimID = previous.ClaimID
	capability := ProviderSessionReuseCapability(session.Backend)
	session.ReuseCapability = string(capability.Level)
	session.ResumeOutcome = "not_attempted"

	if capability.CanAttempt {
		switch session.Backend {
		case BackendCodexAppServer:
			if strings.TrimSpace(previous.CodexThreadID) != "" {
				session.CodexThreadID = previous.CodexThreadID
				session.ResumeAttempted = true
				session.ResumeOutcome = "attempting"
			}
		case BackendCursorVisible:
			if strings.TrimSpace(previous.CursorChatID) != "" {
				session.CursorChatID = previous.CursorChatID
				session.ResumeAttempted = true
				session.ResumeOutcome = "attempting"
			}
		case BackendGeminiHeadless:
			if strings.TrimSpace(previous.GeminiConversationID) != "" {
				session.GeminiProjectID = previous.GeminiProjectID
				session.GeminiConversationID = previous.GeminiConversationID
				session.ResumeAttempted = true
				session.ResumeOutcome = "attempting"
			}
		}
	}

	previous.SupersededByClaimID = session.ClaimID
	_ = persistRuntimeSession(root, previous)

	if emit != nil {
		emit("session_superseded", task, "New runtime session supersedes prior session for this task", map[string]any{
			"claim_id":            session.ClaimID,
			"supersedes_claim_id": previous.ClaimID,
			"backend":             session.Backend,
			"reuse_capability":    string(capability.Level),
			"can_attempt":         capability.CanAttempt,
			"resume_attempted":    session.ResumeAttempted,
		})
	}
}

// ApplyAgentProjectSessionReuse implements the 1-1-1 session model
// (_docs/AGENT_SESSION_REUSE.md): Core keeps at most one live-or-stopped
// session per agent per project. When this task is not a same-task relaunch
// (ApplySessionReuseAt above found nothing to resume), look for the most
// recent session this agent ran anywhere else in the same project and carry
// its resumable identity forward, so a new task continues that session
// instead of always starting fresh.
//
// This intentionally does not touch SupersedesClaimID/SupersededByClaimID:
// those describe one task's own session history for dashboard grouping: a
// cross-task reuse here is a different thing and must not be mixed into it.
//
// Scope: exactly one persistent session per (project, agent). Do not extend
// this to pick among multiple candidate sessions, and do not add a "start a
// second parallel session for this agent" path here — running more than one
// agent session per project needs git-flow/worktree coordination rules Core
// does not define yet (branch or worktree isolation between concurrent
// sessions in the same repository). That is future work, not a parameter to
// flip on this function.
func ApplyAgentProjectSessionReuse(root string, session *RuntimeSession, task Task, emit func(eventType string, task Task, message string, details map[string]any)) {
	if session == nil {
		return
	}
	if strings.TrimSpace(session.SessionID) != "" || strings.TrimSpace(session.CodexThreadID) != "" || strings.TrimSpace(session.CursorChatID) != "" || strings.TrimSpace(session.GeminiConversationID) != "" {
		return // ApplySessionReuseAt already resolved a same-task session.
	}
	previous, ok := previousSessionForAgentProject(root, session.ProjectID, session.Agent)
	if !ok {
		return
	}
	capability := ProviderSessionReuseCapability(session.Backend)
	if !capability.CanAttempt {
		return
	}
	switch session.Backend {
	case BackendBackgroundRemote:
		if strings.TrimSpace(previous.SessionID) == "" {
			return
		}
		session.SessionID = previous.SessionID
		session.BackgroundID = previous.BackgroundID
	case BackendCodexAppServer:
		if strings.TrimSpace(previous.CodexThreadID) == "" {
			return
		}
		session.CodexThreadID = previous.CodexThreadID
	case BackendCursorVisible:
		if strings.TrimSpace(previous.CursorChatID) == "" {
			return
		}
		session.CursorChatID = previous.CursorChatID
	case BackendGeminiHeadless:
		if strings.TrimSpace(previous.GeminiConversationID) == "" {
			return
		}
		session.GeminiProjectID = previous.GeminiProjectID
		session.GeminiConversationID = previous.GeminiConversationID
	default:
		return
	}
	session.ReuseCapability = string(capability.Level)
	session.ResumeAttempted = true
	session.ResumeOutcome = "attempting"
	if emit != nil {
		emit("agent_project_session_reused", task, "Reusing this project's existing agent session for a new task", map[string]any{
			"claim_id":          session.ClaimID,
			"reused_from_claim": previous.ClaimID,
			"backend":           session.Backend,
			"agent":             session.Agent,
			"project_id":        session.ProjectID,
		})
	}
}

// FailLaunchSession records a launch-start failure and publishes Contour 3 outcome.
func (s *Service) FailLaunchSession(session RuntimeSession, task Task, err error) {
	session.Status = "failed"
	if session.ProviderError == nil {
		session.ProviderError = ClassifyProviderError(session.Agent, session.Backend, "launch", err, "")
	}
	if session.ProviderError == nil {
		session.ProviderError = tasklifecycle.NewProviderError(
			firstNonEmpty(session.Agent, task.LaunchEvaluation.Agent, task.Assignee, "agent"),
			"provider_startup_failed",
			"launch",
			"Provider startup failed.",
			err.Error(),
		)
	}
	if session.ProviderError != nil {
		session.ExecutionStatus = "provider_error"
	} else {
		session.ExecutionStatus = "failed"
	}
	session.ErrorMessage = err.Error()
	session.ExitedAt = time.Now().Format(time.RFC3339)
	s.upsertSession(session)
	s.removeSession(session.ClaimID)

	comment := tasklifecycle.LaunchStartFailedComment(err.Error())
	if session.ProviderError != nil {
		comment = ProviderErrorComment(session)
		s.emitRuntimeEvent("runtime_slot_released_after_provider_startup_failure", "", "Provider startup failed; launch slot released", map[string]any{
			"claim_id":         session.ClaimID,
			"task_ref":         session.TaskRef,
			"task_id":          session.TaskID,
			"agent":            session.Agent,
			"project_id":       session.ProjectID,
			"repository":       session.Repository,
			"execution_status": session.ExecutionStatus,
			"provider_error":   providerErrorKind(session),
		})
		if s.logger != nil {
			s.logger.Warn("provider startup failed; launch slot released",
				l.String("task", taskLabel(&task)),
				l.String("claim_id", session.ClaimID),
				l.String("agent", session.Agent),
				l.String("execution_status", session.ExecutionStatus),
				l.String("provider_error", providerErrorKind(session)))
		}
	}
	s.PublishExecutionOutcome(session, task, taskflow.ExecutionFailed, comment, err.Error(), session.LogPath)
	s.emitTaskEvent("agent_process_start_failed", task, err.Error(), map[string]any{
		"claim_id": session.ClaimID,
		"log_path": session.LogPath,
	})
	if s.logger != nil {
		s.logger.Warn("agent process start failed", l.String("task", taskLabel(&task)), l.String("claim_id", session.ClaimID), l.Error(err))
	}
	s.RequeueReleasedSlot(context.Background(), session)
}

func (s *Service) waitLaunchSession(ctx context.Context, process *RunningProcess, session RuntimeSession, task Task) {
	defer process.Close()

	err := process.Cmd.Wait()
	session.ExitedAt = time.Now().Format(time.RFC3339)
	session.Status = "exited"
	session.ExecutionStatus = "succeeded"
	if err != nil {
		session.Status = "failed"
		session.ExecutionStatus = "failed"
		session.ErrorMessage = err.Error()
		if errors.Is(process.Context.Err(), context.DeadlineExceeded) {
			session.ExecutionStatus = "timed_out"
			session.ErrorMessage = "process timed out after " + process.Timeout.String()
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			session.ExitCode = exitErr.ExitCode()
		}
	}
	if session.Status == "failed" {
		session.ProviderError = ClassifyProviderError(session.Agent, session.Backend, "process output", err, ReadSessionLog(s.cfg.RuntimeRoot, session.LogPath))
		if session.ProviderError != nil {
			session.ExecutionStatus = "provider_error"
		}
	}
	if session.Status == "failed" && session.ErrorMessage != "" {
		_, _ = fmt.Fprintf(process.LogFile, "\n[core] session failed: %s\n", session.ErrorMessage)
	}
	if session.Result == nil {
		if result := tasklifecycle.ParseWorkerResult(ReadSessionLog(s.cfg.RuntimeRoot, session.LogPath)); result != nil {
			session.Result = result
			session.LastEvent = "worker_result_reported"
			if summary := strings.TrimSpace(result.Summary); summary != "" {
				session.LastMessage = summary
			}
		}
	}
	RefreshSessionToolUsageFromLog(&session, ReadSessionLog(s.cfg.RuntimeRoot, session.LogPath))
	markSessionStatusFromWorkerResult(&session)
	s.CompleteProviderSession(session, task, firstNonEmpty(session.ErrorMessage, "Agent process exited"), map[string]any{
		"process_id": session.ProcessID,
		"exit_code":  session.ExitCode,
	})
	if s.logger != nil {
		s.logger.Info("agent process exited", l.String("task", taskLabel(&task)), l.String("claim_id", session.ClaimID), l.String("status", session.Status), l.Int("exit_code", session.ExitCode))
	}
	// Requeue is owned by CompleteProviderSession so provider backends share the same wake path.
	_ = ctx
}

// RecordProviderSessionStart persists a running session and emits started breadcrumbs.
func (s *Service) RecordProviderSessionStart(session RuntimeSession, task Task, message string, details map[string]any) {
	s.upsertSession(session)
	s.emitTaskEvent("agent_process_started", task, message, details)
}

// CompleteProviderSession finalizes Contour 3, persists/removes the session, and emits exited.
// After the slot is freed it wakes waiting launchable tasks — Cursor/Claude/Codex
// backends finish only through this path, so requeue must live here (not only on
// the generic process Wait path). Provider startup failures (including
// remote_control_unavailable) must release the assignee/project/repository slot
// deterministically so later tasks can retry.
// HITL (needs_input / waiting_input) is a pause, not a close: keep the session
// in the hub so 1-1-1 still blocks the rest of the project queue.
func (s *Service) CompleteProviderSession(session RuntimeSession, task Task, exitMessage string, details map[string]any) {
	if sessionNeedsOperatorPause(&session) {
		s.pauseSessionForOperatorInput(&session, task)
		return
	}
	s.FinalizeTaskForExecution(&session, task)
	s.upsertSession(session)
	startupFailed := session.ProviderError != nil
	s.removeSession(session.ClaimID)
	if details == nil {
		details = map[string]any{}
	}
	if _, ok := details["claim_id"]; !ok {
		details["claim_id"] = session.ClaimID
	}
	if _, ok := details["status"]; !ok {
		details["status"] = session.Status
	}
	if _, ok := details["exited_at"]; !ok {
		details["exited_at"] = session.ExitedAt
	}
	if _, ok := details["log_path"]; !ok {
		details["log_path"] = session.LogPath
	}
	s.emitTaskEvent("agent_process_exited", task, firstNonEmpty(exitMessage, session.ErrorMessage, "Agent session ended"), details)
	if startupFailed {
		s.emitRuntimeEvent("runtime_slot_released_after_provider_startup_failure", "", "Provider startup failed; launch slot released", map[string]any{
			"claim_id":         session.ClaimID,
			"task_ref":         session.TaskRef,
			"task_id":          session.TaskID,
			"agent":            session.Agent,
			"project_id":       session.ProjectID,
			"repository":       session.Repository,
			"execution_status": session.ExecutionStatus,
			"provider_error":   providerErrorKind(session),
		})
		if s.logger != nil {
			s.logger.Warn("provider startup failed; launch slot released",
				l.String("task", taskLabel(&task)),
				l.String("claim_id", session.ClaimID),
				l.String("agent", session.Agent),
				l.String("execution_status", session.ExecutionStatus),
				l.String("provider_error", providerErrorKind(session)))
		}
	}
	s.RequeueReleasedSlot(context.Background(), session)
}

func providerErrorKind(session RuntimeSession) string {
	if session.ProviderError == nil {
		return ""
	}
	return session.ProviderError.Kind
}

// RuntimeSessionForTask builds the initial session reservation for a launch candidate.
func RuntimeSessionForTask(task Task, cfg Config) RuntimeSession {
	now := time.Now()
	claimID := strings.ToLower(firstNonEmpty(task.Ref, task.ID, "task")) + "-" + strconv.FormatInt(now.UnixNano(), 10)
	evaluation := task.LaunchEvaluation
	hostName := currentHostName()
	return RuntimeSession{
		ClaimID:         claimID,
		TaskRef:         task.Ref,
		TaskID:          task.ID,
		TaskPath:        task.RelativePath,
		TaskTitle:       task.Title,
		ProjectID:       task.ProjectID,
		Repository:      evaluation.Repository,
		Agent:           evaluation.Agent,
		Launcher:        firstNonEmpty(cfg.LauncherIdentity, "core"),
		Backend:         evaluation.Backend,
		HostID:          hostName,
		HostName:        hostName,
		Command:         append([]string{}, evaluation.Command...),
		LogPath:         filepath.ToSlash(filepath.Join("_registry", "sessions", claimID+".log")),
		WorkingDir:      evaluation.WorkingDir,
		ClaimedAt:       now.Format(time.RFC3339),
		Status:          "starting",
		ExecutionStatus: "starting",
	}
}

func runtimeSessionForTask(task Task, cfg Config) RuntimeSession {
	return RuntimeSessionForTask(task, cfg)
}

func RuntimeSessionForReleasedTask(task Task, claimID string, reason string) RuntimeSession {
	return RuntimeSession{
		ClaimID:         claimID,
		TaskRef:         task.Ref,
		TaskID:          task.ID,
		TaskPath:        task.RelativePath,
		TaskTitle:       task.Title,
		ProjectID:       task.ProjectID,
		Repository:      firstRepository(task),
		Agent:           firstNonEmpty(task.Launch.Agent, task.Assignee),
		Status:          "released",
		ExecutionStatus: "released",
		ErrorMessage:    reason,
		ExitedAt:        time.Now().Format(time.RFC3339),
	}
}

func currentHostName() string {
	name, err := os.Hostname()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(name)
}
