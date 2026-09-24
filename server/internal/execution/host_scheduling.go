package execution

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/execution/providers"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	l "github.com/eggs-gd/fleet.eggs.gd/lib/logger"
)

// HandleLaunchResult is the post-admission consumer for Contour 2 launches.
func (s *Service) HandleLaunchResult(ctx context.Context, task Task) {
	if s.board.UpsertTask != nil {
		s.board.UpsertTask(task)
	}
	decision := ClassifyLaunchLog(task)
	switch decision.Category {
	case LaunchLogCandidate:
		if s.logger != nil {
			s.logger.Info("launch-plan candidate",
				l.String("task", taskLabel(&task)),
				l.String("agent", task.LaunchEvaluation.Agent),
				l.String("cwd", task.LaunchEvaluation.WorkingDir),
				l.String("dry_run", strconv.FormatBool(s.cfg.DryRun)))
		}
		if s.cfg.AllowsProcessStart() {
			go s.launchNextForAssignee(ctx, task.LaunchEvaluation.Agent)
		}
	case LaunchLogSkipped:
		if !decision.Routine && s.logger != nil {
			s.logger.Info("launch skipped",
				l.String("task", taskLabel(&task)),
				l.String("stage", decision.Stage),
				l.String("reason", decision.Reason))
		}
	case LaunchLogNotReady:
		if s.logger != nil {
			s.logger.Warn("launch not ready",
				l.String("task", taskLabel(&task)),
				l.String("stage", decision.Stage),
				l.String("reason", decision.Reason))
		}
		s.BlockNotReadyLaunchTask(task, decision)
	}
	s.emitTaskEvent(decision.Event, task, decision.Reason, map[string]any{
		"stage":    decision.Stage,
		"category": string(decision.Category),
		"routine":  decision.Routine,
	})
}

// LaunchNextForAssignee picks the next launchable task for an assignee and starts it.
func (s *Service) LaunchNextForAssignee(ctx context.Context, assignee string) {
	s.launchNextForAssignee(ctx, assignee)
}

// LaunchPendingTasks scans the board for launchable todo/needs_rework tasks and
// starts one candidate per assignee when process start is allowed. Used after
// bootstrap so existing pickup tasks do not wait for an unrelated task event.
func (s *Service) LaunchPendingTasks(ctx context.Context) {
	if !s.cfg.AllowsProcessStart() {
		return
	}
	tasks := []Task{}
	if s.board.ListTasks != nil {
		tasks = s.board.ListTasks()
	}
	assignees := map[string]struct{}{}
	queued := 0
	for _, task := range tasks {
		if !tasklifecycle.LaunchablePickupStatus(task.Status) {
			continue
		}
		agent := firstNonEmpty(task.Launch.Agent, task.Assignee)
		if agent == "" {
			s.logLaunchSkipped(task, "bootstrap", "missing assignee")
			continue
		}
		assignees[agent] = struct{}{}
		queued++
		s.emitTaskEvent("launch_queued", task, "Launchable task queued for pickup evaluation", map[string]any{
			"stage":  "bootstrap",
			"agent":  agent,
			"status": task.Status,
		})
		if s.logger != nil {
			s.logger.Info("launchable and queued",
				l.String("task", taskLabel(&task)),
				l.String("agent", agent),
				l.String("status", task.Status),
				l.String("stage", "bootstrap"))
		}
	}
	if s.logger != nil {
		s.logger.Info("bootstrap launch scan",
			l.Int("launchable", queued),
			l.Int("assignees", len(assignees)))
	}
	for agent := range assignees {
		go s.launchNextForAssignee(ctx, agent)
	}
}

func (s *Service) launchNextForAssignee(ctx context.Context, assignee string) {
	tasks := []Task{}
	if s.board.ListTasks != nil {
		tasks = s.board.ListTasks()
	}
	candidates := tasklifecycle.LaunchableCandidatesForAssignee(tasks, assignee)
	if s.logger != nil {
		s.logger.Info("pickup queue",
			l.String("assignee", assignee),
			l.Int("candidates", len(candidates)),
			l.String("order", fmt.Sprintf("%v", TaskLabels(candidates))))
	}
	for _, candidate := range candidates {
		if s.EvaluateAndStartCandidate(ctx, candidate, nil) {
			return
		}
	}
}

// EvaluateAndStartCandidate validates/plans a task and starts it when launchable.
func (s *Service) EvaluateAndStartCandidate(ctx context.Context, candidate Task, released *RuntimeSession) bool {
	task := candidate
	agent := firstNonEmpty(task.Launch.Agent, task.Assignee)
	if !providers.AgentEnabled(agent) {
		s.logLaunchSkipped(task, "disabled", "agent is disabled in Settings; live sessions are not stopped")
		return false
	}
	validated, err := (&Validator{Root: s.cfg.Root, KnownTasks: s.board.ListTasks}).Decorate(&task)
	if err != nil {
		s.logLaunchSkipped(task, "validate", err.Error())
		return false
	}
	planner := s.planner
	if planner == nil {
		planner = DefaultPlanner()
	}
	planned, err := (&AgentLauncher{Planner: planner}).Decorate(validated)
	if err != nil {
		s.logLaunchSkipped(*validated, "plan", err.Error())
		return false
	}
	if !planned.LaunchEvaluation.Launchable {
		if s.board.UpsertTask != nil {
			s.board.UpsertTask(*planned)
		}
		reason := firstNonEmpty(strings.Join(planned.LaunchEvaluation.FailedGates, "; "), "not launchable")
		s.logLaunchSkipped(*planned, "gates", reason)
		return false
	}
	if released != nil {
		s.emitTaskEvent("launch_requeue_selected", *planned, "Selected next launch candidate after slot release", map[string]any{
			"stage":               "slot-release",
			"released_claim_id":   released.ClaimID,
			"released_agent":      released.Agent,
			"released_project":    released.ProjectID,
			"released_repository": released.Repository,
		})
		if s.logger != nil {
			s.logger.Info("requeued after release",
				l.String("task", taskLabel(planned)),
				l.String("released_claim_id", released.ClaimID),
				l.String("agent", planned.LaunchEvaluation.Agent))
		}
	}
	s.startLaunchCandidate(ctx, *planned)
	return true
}

// RequeueDependentsOf starts launch evaluation for pickup tasks that list
// completed as a depends_on prerequisite (CORE-148).
func (s *Service) RequeueDependentsOf(ctx context.Context, completed Task) {
	if s == nil || !s.cfg.AllowsProcessStart() {
		return
	}
	if !tasklifecycle.DependencySatisfied(completed.Status) {
		return
	}
	tasks := []Task{}
	if s.board.ListTasks != nil {
		tasks = s.board.ListTasks()
	}
	dependents := tasklifecycle.TasksDependingOn(tasks, completed)
	if len(dependents) == 0 {
		return
	}
	assignees := map[string]struct{}{}
	for _, dependent := range dependents {
		agent := firstNonEmpty(dependent.Launch.Agent, dependent.Assignee)
		if agent == "" {
			continue
		}
		assignees[agent] = struct{}{}
		s.emitTaskEvent("launch_dependency_unblocked", dependent, "Prerequisite reached done; dependent returned to launch evaluation", map[string]any{
			"stage":            "depends_on",
			"prerequisite_ref": firstNonEmpty(completed.Ref, completed.ID),
			"prerequisite_id":  completed.ID,
			"dependent_ref":    dependent.Ref,
			"dependent_status": dependent.Status,
		})
		if s.logger != nil {
			s.logger.Info("dependency unblocked",
				l.String("task", taskLabel(&dependent)),
				l.String("prerequisite", firstNonEmpty(completed.Ref, completed.ID)),
				l.String("agent", agent))
		}
	}
	for agent := range assignees {
		go s.launchNextForAssignee(ctx, agent)
	}
}

func (s *Service) logLaunchSkipped(task Task, stage string, reason string) {
	s.emitTaskEvent("launch_skipped", task, reason, map[string]any{
		"stage": stage,
	})
	if s.logger != nil {
		s.logger.Info("skipped",
			l.String("task", taskLabel(&task)),
			l.String("stage", stage),
			l.String("reason", reason))
	}
}

// RequeueReleasedSlot evaluates pending launchable tasks after a session ends.
func (s *Service) RequeueReleasedSlot(ctx context.Context, ended RuntimeSession) {
	if !s.cfg.AllowsProcessStart() {
		return
	}
	scopes := ReleasedSlotScopes(ended)
	s.emitRuntimeEvent("runtime_slot_released", "", "Runtime launch slot released; evaluating pending tasks", map[string]any{
		"claim_id":         ended.ClaimID,
		"task_ref":         ended.TaskRef,
		"task_id":          ended.TaskID,
		"task_path":        ended.TaskPath,
		"project_id":       ended.ProjectID,
		"repository":       ended.Repository,
		"agent":            ended.Agent,
		"status":           ended.Status,
		"execution_status": ended.ExecutionStatus,
		"released_scopes":  scopes,
	})

	candidates := s.requeueCandidatesForReleasedSlot(ended)
	refs := TaskLabels(candidates)
	if s.logger != nil {
		s.logger.Info("runtime slot released",
			l.String("claim_id", ended.ClaimID),
			l.String("agent", ended.Agent),
			l.String("project", ended.ProjectID),
			l.String("repository", ended.Repository),
			l.Int("tasks", len(candidates)))
	}
	s.emitRuntimeEvent("launch_requeue_pending", "", "Pending launch tasks matched released slot", map[string]any{
		"released_claim_id": ended.ClaimID,
		"released_scopes":   scopes,
		"task_refs":         refs,
		"task_count":        len(candidates),
	})

	for _, candidate := range candidates {
		if candidate.LaunchEvaluation.Waiting != nil {
			candidate.ClearLaunchWaiting()
			if s.board.UpsertTask != nil {
				s.board.UpsertTask(candidate)
			}
		}
		s.emitTaskEvent("launch_wait_requeued", candidate, "Task matched released launch slot and returned to launch evaluation", map[string]any{
			"stage":               "slot-release",
			"released_claim_id":   ended.ClaimID,
			"released_agent":      ended.Agent,
			"released_repository": ended.Repository,
			"released_project_id": ended.ProjectID,
		})
		if s.logger != nil {
			s.logger.Info("requeued after release",
				l.String("task", taskLabel(&candidate)),
				l.String("released_claim_id", ended.ClaimID),
				l.String("status", candidate.Status))
		}
		if s.EvaluateAndStartCandidate(ctx, candidate, &ended) {
			return
		}
	}

	s.emitRuntimeEvent("launch_requeue_no_eligible", "", "No eligible launch candidate found after slot release", map[string]any{
		"released_claim_id": ended.ClaimID,
		"released_scopes":   scopes,
		"task_refs":         refs,
		"task_count":        len(candidates),
	})
}

func (s *Service) requeueCandidatesForReleasedSlot(ended RuntimeSession) []Task {
	candidates := []Task{}
	tasks := []Task{}
	if s.board.ListTasks != nil {
		tasks = s.board.ListTasks()
	}
	for _, task := range tasks {
		if !tasklifecycle.LaunchablePickupStatus(task.Status) {
			continue
		}
		if TaskMatchesReleasedSlot(task, ended) {
			candidates = append(candidates, task)
		}
	}
	return tasklifecycle.SortTasksForPickup(candidates)
}

// RequeueCandidatesForReleasedSlot lists launchable tasks that match a freed slot.
func (s *Service) RequeueCandidatesForReleasedSlot(ended RuntimeSession) []Task {
	return s.requeueCandidatesForReleasedSlot(ended)
}

func TaskMatchesReleasedSlot(task Task, ended RuntimeSession) bool {
	agent := firstNonEmpty(task.Launch.Agent, task.Assignee)
	if ended.Agent != "" && agent == ended.Agent {
		return true
	}
	if ended.ProjectID != "" && task.ProjectID == ended.ProjectID {
		return true
	}
	if ended.Repository != "" {
		if firstRepository(task) == ended.Repository {
			return true
		}
		for _, repository := range task.Repositories {
			if repository == ended.Repository {
				return true
			}
		}
	}
	wait := task.LaunchEvaluation.Waiting
	if wait == nil {
		return false
	}
	if wait.ConflictClaimID == ended.ClaimID {
		return true
	}
	switch wait.Scope {
	case "assignee", "assignee_orphan":
		return ended.Agent != "" && wait.Resource == ended.Agent
	case "project":
		return ended.ProjectID != "" && wait.Resource == ended.ProjectID
	case "repository":
		return ended.Repository != "" && wait.Resource == ended.Repository
	default:
		return false
	}
}

func ReleasedSlotScopes(session RuntimeSession) []string {
	scopes := []string{}
	if session.Agent != "" {
		scopes = append(scopes, "assignee:"+session.Agent)
	}
	if session.ProjectID != "" {
		scopes = append(scopes, "project:"+session.ProjectID)
	}
	if session.Repository != "" {
		scopes = append(scopes, "repository:"+session.Repository)
	}
	return scopes
}

func TaskLabels(tasks []Task) []string {
	labels := make([]string, 0, len(tasks))
	for _, task := range tasks {
		labels = append(labels, taskLabel(&task))
	}
	return labels
}

func SessionConflictResource(scope string, task Task) string {
	if scope == "assignee" || scope == "assignee_orphan" {
		return firstNonEmpty(task.LaunchEvaluation.Agent, task.Launch.Agent, task.Assignee)
	}
	if scope == "repository" {
		return firstRepository(task)
	}
	return firstNonEmpty(task.ProjectID, firstRepository(task))
}

func LaunchConflictReason(conflict RuntimeSession, scope string, resource string) string {
	if scope == "assignee_orphan" {
		return fmt.Sprintf("assignee %q has unresolved orphaned doing task %s; move the orphan to blocked, needs_rework, todo, done, or another valid state to release this launch slot", resource, taskLabelFromParts(conflict.TaskRef, conflict.TaskID, conflict.TaskPath))
	}
	state := firstNonEmpty(conflict.ExecutionStatus, conflict.Status)
	if conflict.IsProviderStartup() {
		return fmt.Sprintf("active runtime session %s already owns %s %q during provider startup (state %s); wait for visible-session registration or startup failure before launching another task", conflict.ClaimID, scope, resource, state)
	}
	return fmt.Sprintf("active runtime session %s already owns %s %q", conflict.ClaimID, scope, resource)
}

func FirstRepository(task Task) string {
	if task.LaunchEvaluation.Repository != "" {
		return task.LaunchEvaluation.Repository
	}
	if len(task.Repositories) > 0 {
		return task.Repositories[0]
	}
	return ""
}

func firstRepository(task Task) string {
	return FirstRepository(task)
}

func taskLabelFromParts(ref string, id string, path string) string {
	return firstNonEmpty(ref, id, path, "unknown task")
}
