package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/audit"
	"github.com/eggs-gd/fleet.eggs.gd/internal/board"
	"github.com/eggs-gd/fleet.eggs.gd/internal/execution"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider"
	"github.com/eggs-gd/fleet.eggs.gd/internal/workfiles"
	l "github.com/eggs-gd/fleet.eggs.gd/lib/logger"
)

// Dashboard is the operator-facing HTTP surface over Board + TaskService + execution.
type Dashboard struct {
	root           string
	runtimeRoot    string
	dryRun         bool
	sessionTimeout time.Duration
	store          *taskprovider.Board
	taskStore      taskprovider.Provider
	taskService    *taskprovider.Service
	exec           *execution.Service
	logger         *l.Logger
}

// DashboardOptions constructs the operator-facing dashboard surface.
type DashboardOptions struct {
	Root           string
	RuntimeRoot    string
	DryRun         bool
	SessionTimeout time.Duration
	Store          *taskprovider.Board
	TaskStore      taskprovider.Provider
	TaskService    *taskprovider.Service
	Execution      *execution.Service
	Logger         *l.Logger
}

func NewDashboard(opts DashboardOptions) *Dashboard {
	return &Dashboard{
		root:           opts.Root,
		runtimeRoot:    opts.RuntimeRoot,
		dryRun:         opts.DryRun,
		sessionTimeout: opts.SessionTimeout,
		store:          opts.Store,
		taskStore:      opts.TaskStore,
		taskService:    opts.TaskService,
		exec:           opts.Execution,
		logger:         opts.Logger,
	}
}

func (d *Dashboard) effectiveSessionTimeout() time.Duration {
	if d == nil || d.sessionTimeout <= 0 {
		return 10 * time.Minute
	}
	return d.sessionTimeout
}

func (d *Dashboard) allowsProcessStart() bool {
	return d != nil && !d.dryRun
}

func (d *Dashboard) State() State {
	if d == nil || d.store == nil {
		return State{}
	}
	snap := d.store.Snapshot()
	state := State{
		GeneratedAt:       snap.GeneratedAt,
		Root:              snap.Root,
		Statuses:          snap.Statuses,
		StatusTransitions: tasklifecycle.StatusTransitionMap(),
		Workspaces:        snap.Workspaces,
		Projects:          snap.Projects,
		Tasks:             snap.Tasks,
		People:            tasklifecycle.RosterPeople(d.root),
		Registry:          snap.Registry,
	}
	if d.exec != nil {
		status := d.exec.Status()
		state.RuntimeSessions = status.RuntimeSessions
		state.SessionGroups = status.SessionGroups
		state.OrphanedTasks = status.OrphanedTasks
		for i := range state.Tasks {
			state.Tasks[i] = d.exec.AnnotateTask(state.Tasks[i])
		}
	}
	for i := range state.RuntimeSessions {
		state.RuntimeSessions[i].ProviderControllable = d.exec != nil && d.exec.HasSessionControl(state.RuntimeSessions[i].ClaimID)
	}
	for i := range state.SessionGroups {
		for j := range state.SessionGroups[i].Sessions {
			state.SessionGroups[i].Sessions[j].ProviderControllable = d.exec != nil && d.exec.HasSessionControl(state.SessionGroups[i].Sessions[j].ClaimID)
		}
	}
	return state
}

func (d *Dashboard) Bootstrap() error {
	if d == nil {
		return errors.New("dashboard is not available")
	}
	root := d.root
	runtimeRoot := d.runtimeRoot
	launchMode := "dry-run"
	if d.allowsProcessStart() {
		launchMode = "live"
	}
	timeout := d.effectiveSessionTimeout().String()
	if d.logger != nil {
		d.logger.Info("bootstrap started", l.String("stage", "scan"), l.String("root", root), l.String("launch_mode", launchMode), l.String("session_timeout", timeout))
	}
	_ = audit.AppendEvent(runtimeRoot, audit.Event{
		Type:    "bootstrap_started",
		Message: "Core runtime bootstrap started",
		Details: map[string]any{
			"stage":           "scan",
			"mode":            "initial",
			"launch_mode":     launchMode,
			"session_timeout": timeout,
		},
	})
	tasks, workspaces, err := taskprovider.Hydrate(d.store, root, d.taskStore)
	if err != nil {
		_ = audit.AppendEvent(runtimeRoot, audit.Event{Type: "bootstrap_failed", Message: err.Error()})
		return err
	}
	if d.exec != nil {
		if err := d.exec.ReconcilePersistedSessions(tasks); err != nil {
			_ = audit.AppendEvent(runtimeRoot, audit.Event{Type: "bootstrap_failed", Message: err.Error()})
			return err
		}
		d.exec.DetectStartupOrphans(tasks)
		d.exec.AnnounceOpenPauses(tasks)
	}
	if rebuilder, ok := d.taskStore.(interface{ RebuildDerivedViews() error }); ok {
		if err := rebuilder.RebuildDerivedViews(); err != nil {
			_ = audit.AppendEvent(runtimeRoot, audit.Event{Type: "bootstrap_failed", Message: err.Error()})
			return err
		}
	}
	_ = audit.AppendEvent(runtimeRoot, audit.Event{
		Type:    "bootstrap_completed",
		Message: "Core runtime bootstrap completed",
		Details: map[string]any{
			"stage":           "scan",
			"mode":            "initial",
			"launch_mode":     launchMode,
			"session_timeout": timeout,
			"tasks":           len(tasks),
			"workspaces":      len(workspaces),
		},
	})
	if d.logger != nil {
		d.logger.Info("bootstrap completed", l.String("stage", "scan"), l.String("mode", "initial"), l.String("launch_mode", launchMode), l.String("session_timeout", timeout), l.Int("tasks", len(tasks)), l.Int("workspaces", len(workspaces)))
	}
	return nil
}

func (d *Dashboard) PatchTask(patch tasklifecycle.TaskPatch) (tasklifecycle.Task, error) {
	before, _ := d.taskStore.Load(patch.Path)
	flowTask, err := d.taskService.Patch(context.Background(), patch.Path, taskflow.PatchInput{
		Status:        taskflow.Status(strings.TrimSpace(patch.Status)),
		Priority:      patch.Priority,
		Assignee:      patch.Assignee,
		Project:       patch.Project,
		Repository:    patch.Repository,
		Milestone:     patch.Milestone,
		DependsOn:     patch.DependsOn,
		Body:          patch.Body,
		Comment:       patch.Comment,
		CommentAuthor: patch.CommentAuthor,
	})
	if err != nil {
		return tasklifecycle.Task{}, err
	}
	task := execution.TaskFromFlow(flowTask)
	if loaded, loadErr := d.taskStore.Load(firstNonEmpty(task.RelativePath, task.Path, patch.Path)); loadErr == nil {
		task = loaded
	}
	changes := tasklifecycle.TaskPatchChangeSummary(before, task, patch)
	if len(changes) > 0 && d.logger != nil {
		d.logger.Info("status update", l.String("task", taskLabel(&task)), l.String("source", "api"), l.String("index_rebuilt", "true"), l.String("changes", strings.Join(changes, " ")))
	}
	if before.Status == "doing" && task.Status != "doing" && d.exec != nil {
		d.exec.ReleaseActiveSessionForTask(before, "doing task left active-session status")
	}
	return task, nil
}

func (d *Dashboard) Milestones(project string) ([]board.Milestone, error) {
	return workfiles.Milestones(d.root, project)
}

func (d *Dashboard) CreateMilestone(project, title, version string) (board.Milestone, error) {
	milestone, err := workfiles.CreateMilestone(d.root, project, title, version)
	if err != nil {
		return board.Milestone{}, err
	}
	d.ReloadProjectCard(project)
	return milestone, nil
}

// PatchMilestone applies whichever of state/title/version is non-nil to one
// milestone on a project card.
func (d *Dashboard) PatchMilestone(project, milestoneID string, state, title, version *string) (board.Milestone, error) {
	milestone, err := workfiles.PatchMilestone(d.root, project, milestoneID, state, title, version)
	if err != nil {
		return board.Milestone{}, err
	}
	d.ReloadProjectCard(project)
	return milestone, nil
}

// DeleteMilestone clears the milestone from every task that points at it, then
// removes it from the card. Tasks go first: if one fails the milestone is still
// there and the delete can be retried, instead of leaving tasks that name a
// milestone that no longer exists.
func (d *Dashboard) DeleteMilestone(project, milestoneID string) (int, error) {
	if !workfiles.ValidProjectID(project) {
		return 0, errors.New("project id is required")
	}
	existing, err := workfiles.Milestones(d.root, project)
	if err != nil {
		return 0, err
	}
	found := false
	for _, milestone := range existing {
		found = found || milestone.ID == milestoneID
	}
	if !found {
		return 0, fmt.Errorf("project %s has no milestone %q", project, milestoneID)
	}
	cleared := 0
	none := ""
	for _, task := range d.store.Snapshot().Tasks {
		if task.WorkspaceID != project || task.Milestone != milestoneID {
			continue
		}
		if _, err := d.taskService.Patch(context.Background(), task.Path, taskflow.PatchInput{Milestone: &none}); err != nil {
			return cleared, fmt.Errorf("clear milestone on %s: %w", firstNonEmpty(task.Ref, task.Path), err)
		}
		cleared++
	}
	if err := workfiles.DeleteMilestone(d.root, project, milestoneID); err != nil {
		return cleared, err
	}
	d.ReloadProjectCard(project)
	return cleared, nil
}

// ReloadProjectCard refreshes the live board from one project's PROJECT.md
// after a write made outside the scanner (milestones, aliases, summary). The
// board is otherwise only hydrated at Bootstrap.
func (d *Dashboard) ReloadProjectCard(project string) {
	if d == nil || d.store == nil || !workfiles.ValidProjectID(project) {
		return
	}
	if err := taskprovider.ReloadWorkspace(d.store, d.root, project); err != nil && d.logger != nil {
		d.logger.Warn("reload project card", l.String("project", project), l.String("error", err.Error()))
	}
}

func (d *Dashboard) CreateTask(req tasklifecycle.TaskCreateRequest) (tasklifecycle.Task, error) {
	input := taskflow.CreateTask{
		Title:            req.Title,
		Description:      req.Request,
		Project:          req.Project,
		Repository:       req.Repository,
		Status:           taskflow.Status(req.Status),
		Type:             req.Type,
		Assignee:         req.Assignee,
		AssignmentReason: req.AssignmentReason,
		DependsOn:        append([]string{}, req.DependsOn...),
	}
	if req.Priority != nil {
		input.Priority = *req.Priority
	}
	flowTask, err := d.taskService.Create(context.Background(), input)
	if err != nil {
		return tasklifecycle.Task{}, err
	}
	task := execution.TaskFromFlow(flowTask)
	if loaded, loadErr := d.taskStore.Load(firstNonEmpty(task.RelativePath, task.Path, flowTask.Locator)); loadErr == nil {
		task = loaded
	}
	if d.logger != nil {
		d.logger.Info("task created", l.String("task", taskLabel(&task)), l.String("source", "api"), l.String("path", task.RelativePath), l.String("status", task.Status))
	}
	return task, nil
}

func (d *Dashboard) ControlSession(patch execution.SessionControlPatch) error {
	if d == nil || d.exec == nil {
		return errors.New("execution service is not available")
	}
	return d.exec.ControlSession(patch)
}

func (d *Dashboard) TaskService() taskprovider.TaskService {
	if d == nil {
		return nil
	}
	return d.taskService
}

var _ DashboardSurface = (*Dashboard)(nil)
