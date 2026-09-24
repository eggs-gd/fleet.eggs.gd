package server

import (
	"github.com/eggs-gd/fleet.eggs.gd/internal/audit"
	"github.com/eggs-gd/fleet.eggs.gd/internal/board"
	"github.com/eggs-gd/fleet.eggs.gd/internal/execution"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider"
)

// State is the /api/state JSON payload: board projection + execution status.
type State struct {
	GeneratedAt       string                     `json:"generated_at"`
	Root              string                     `json:"root"`
	Statuses          []string                   `json:"statuses"`
	StatusTransitions map[string][]string        `json:"status_transitions"`
	Workspaces        []board.Workspace          `json:"workspaces"`
	Projects          []board.Project            `json:"projects"`
	Tasks             []tasklifecycle.Task       `json:"tasks"`
	RuntimeSessions   []execution.RuntimeSession `json:"runtime_sessions"`
	SessionGroups     []execution.SessionGroup   `json:"session_groups"`
	OrphanedTasks     []execution.OrphanedTask   `json:"orphaned_tasks"`
	LifeItems         []board.LifeItem           `json:"life_items"`
	Registry          board.RegistryInfo         `json:"registry"`
}

// DashboardSurface is the only Core surface HTTP dashboard handlers may use.
type DashboardSurface interface {
	State() State
	ControlSession(patch execution.SessionControlPatch) error
	CreateTask(req tasklifecycle.TaskCreateRequest) (tasklifecycle.Task, error)
	PatchTask(patch tasklifecycle.TaskPatch) (tasklifecycle.Task, error)
}

func boardHooks(runtimeRoot string, store *taskprovider.Board) execution.BoardHooks {
	return execution.BoardHooks{
		UpsertTask: store.UpsertTask,
		ListTasks: func() []tasklifecycle.Task {
			return store.Snapshot().Tasks
		},
		TaskByPath: store.TaskByPath,
		EmitTaskEvent: func(eventType string, task tasklifecycle.Task, message string, details map[string]any) {
			emitTaskEvent(runtimeRoot, eventType, task, message, details)
		},
		EmitRuntimeEvent: func(typ, path, message string, details map[string]any) {
			_ = audit.AppendEvent(runtimeRoot, audit.Event{Type: typ, Path: path, Message: message, Details: details})
		},
	}
}

func emitTaskEvent(runtimeRoot string, eventType string, task tasklifecycle.Task, message string, extra map[string]any) {
	details := map[string]any{
		"status":            task.Status,
		"priority":          task.Priority,
		"failed_gates":      task.LaunchEvaluation.FailedGates,
		"passed_gates":      task.LaunchEvaluation.PassedGates,
		"launchable":        task.LaunchEvaluation.Launchable,
		"launch_outcome":    task.LaunchEvaluation.Outcome,
		"launch_waiting":    task.LaunchEvaluation.Waiting,
		"working_dir":       task.LaunchEvaluation.WorkingDir,
		"command":           task.LaunchEvaluation.Command,
		"launch_config":     task.Launch,
		"launch_evaluation": task.LaunchEvaluation,
	}
	for key, value := range extra {
		details[key] = value
	}
	_ = audit.AppendEvent(runtimeRoot, audit.Event{
		Type:       eventType,
		TaskRef:    task.Ref,
		TaskID:     task.ID,
		Path:       task.RelativePath,
		Agent:      firstNonEmpty(task.LaunchEvaluation.Agent, task.Launch.Agent, task.Assignee),
		Repository: task.LaunchEvaluation.Repository,
		Outcome:    task.LaunchEvaluation.Outcome,
		Message:    message,
		Details:    details,
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func taskLabel(task *tasklifecycle.Task) string {
	if task == nil {
		return "unknown task"
	}
	if task.Ref != "" {
		return task.Ref
	}
	return firstNonEmpty(task.ID, task.RelativePath, task.Path, "unknown task")
}
