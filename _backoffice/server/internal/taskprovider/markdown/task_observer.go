package markdown

import (
	"fmt"
	"strings"

	"github.com/eggs-gd/core.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/core.eggs.gd/internal/tasklifecycle"
)

type ActiveSession struct {
	ClaimID         string
	ExecutionStatus string
	Status          string
}

type ObserveTaskHooks struct {
	Before                func(locator string) (tasklifecycle.Task, bool)
	ActiveSessionForTask  func(task tasklifecycle.Task) (ActiveSession, bool)
	RevertActiveExecution func(locator string, session ActiveSession) (tasklifecycle.Task, error)
}

type ObservedTaskChange struct {
	Before         *tasklifecycle.Task
	After          tasklifecycle.Task
	Event          taskflow.TaskEvent
	GuardApplied   bool
	ObservedStatus string
	Session        ActiveSession
}

// ObserveTaskFileChange owns Markdown task reload, active-execution guard
// fallback, and normalized TaskEvent construction. Callers provide projection
// hooks so this package does not depend on runtime store implementations.
func (p *Provider) ObserveTaskFileChange(path string, hooks ObserveTaskHooks) (ObservedTaskChange, error) {
	task, err := p.Reload(path)
	if err != nil {
		return ObservedTaskChange{}, err
	}

	locator := strings.TrimSpace(task.RelativePath)
	if locator == "" {
		locator = strings.TrimSpace(task.Path)
	}

	var beforePtr *tasklifecycle.Task
	if hooks.Before != nil {
		if before, ok := hooks.Before(locator); ok {
			beforeCopy := before
			beforePtr = &beforeCopy
		}
	}

	if hooks.ActiveSessionForTask != nil {
		if session, ok := hooks.ActiveSessionForTask(task); ok && task.Status != "doing" {
			if hooks.RevertActiveExecution == nil {
				return ObservedTaskChange{}, fmt.Errorf("markdown: RevertActiveExecution hook is required when active session guard is enabled")
			}
			reverted, err := hooks.RevertActiveExecution(locator, session)
			if err != nil {
				return ObservedTaskChange{}, err
			}
			return ObservedTaskChange{
				Before:         beforePtr,
				After:          reverted,
				Event:          TaskEventFor(beforePtr, reverted, taskflow.TaskEventSourceMarkdownFS),
				GuardApplied:   true,
				ObservedStatus: task.Status,
				Session:        session,
			}, nil
		}
	}

	return ObservedTaskChange{
		Before: beforePtr,
		After:  task,
		Event:  TaskEventFor(beforePtr, task, taskflow.TaskEventSourceMarkdownFS),
	}, nil
}
