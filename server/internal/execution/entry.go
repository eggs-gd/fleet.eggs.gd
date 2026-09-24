// Package execution owns provider-neutral launch admission: TaskEvent intake,
// launch validation, and agent command planning. It must not import
// concrete taskprovider adapters.
package execution

import (
	"context"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/lib/chain"
)

// Task is the lifecycle projection used by launch admission.
type Task = tasklifecycle.Task

// Lookup resolves a projected task by locator/id for launch fields that are
// not yet fully represented on taskflow.Task.
type Lookup func(string) (Task, bool)

// NewEntry is the execution runtime's TaskEvent entry point.
// It never begins from a filesystem path, Markdown document, Plane payload,
// or Manager request. Provider Source values are diagnostics only.
// onDependencySatisfied is optional and fires when a task reaches the
// depends_on terminal status so dependents can re-enter launch evaluation.
func NewEntry(service taskflow.TaskService, lookup Lookup, in <-chan taskflow.TaskEvent, out chan<- *Task) chain.Processor {
	return chain.NewDecorator(in, out, &Entry{Service: service, Lookup: lookup})
}

// NewEntryWithHooks is NewEntry plus optional post-change hooks used by the
// execution host (CORE-148 dependent wake).
func NewEntryWithHooks(service taskflow.TaskService, lookup Lookup, onDependencySatisfied func(Task), in <-chan taskflow.TaskEvent, out chan<- *Task) chain.Processor {
	return chain.NewDecorator(in, out, &Entry{
		Service:               service,
		Lookup:                lookup,
		OnDependencySatisfied: onDependencySatisfied,
	})
}

// Entry inspects TaskEvent Before/After deltas and loads a task for Validator.
type Entry struct {
	Service               taskflow.TaskService
	Lookup                Lookup
	OnDependencySatisfied func(Task)
}

func (entry *Entry) Decorate(event taskflow.TaskEvent) (*Task, error) {
	if strings.TrimSpace(event.TaskID) == "" {
		return nil, chain.ErrSkippedItem
	}

	if event.StatusChanged() && tasklifecycle.DependencySatisfied(string(event.After.Status)) {
		if entry.OnDependencySatisfied != nil {
			entry.OnDependencySatisfied(TaskFromFlow(event.After))
		}
	}

	if !event.MayConsiderLaunch() {
		return nil, chain.ErrSkippedItem
	}

	flowTask, err := entry.Service.Get(context.Background(), event.TaskID)
	if err != nil {
		return nil, err
	}

	if entry.Lookup != nil {
		if stored, ok := entry.Lookup(event.TaskID); ok {
			return &stored, nil
		}
		if locator := strings.TrimSpace(flowTask.Locator); locator != "" {
			if stored, ok := entry.Lookup(locator); ok {
				return &stored, nil
			}
		}
	}

	task := TaskFromFlow(flowTask)
	return &task, nil
}

func (entry *Entry) Stop() {}

// TaskFromFlow maps a taskflow.Task into the lifecycle projection used by
// Validator / AgentLauncher.
func TaskFromFlow(task taskflow.Task) Task {
	locator := strings.TrimSpace(task.Locator)
	if locator == "" {
		locator = strings.TrimSpace(task.ID)
	}
	comments := make([]tasklifecycle.Comment, 0, len(task.Comments))
	for _, c := range task.Comments {
		comments = append(comments, tasklifecycle.Comment{
			Author:    c.Author,
			CreatedAt: c.CreatedAt,
			Text:      c.Text,
		})
	}
	return Task{
		SchemaVersion:    task.SchemaVersion,
		ID:               task.ID,
		Ref:              task.Ref,
		Title:            task.Title,
		Type:             task.Type,
		Status:           string(task.Status),
		Priority:         task.Priority,
		Project:          task.Project,
		ProjectID:        task.ProjectID,
		WorkspaceID:      task.WorkspaceID,
		Repositories:     append([]string{}, task.Repositories...),
		DependsOn:        append([]string{}, task.DependsOn...),
		Assignee:         task.Assignee,
		AssignmentReason: task.AssignmentReason,
		Source:           task.Source,
		CreatedAt:        task.CreatedAt,
		UpdatedAt:        task.UpdatedAt,
		Summary:          task.Summary,
		Body:             task.Body,
		Comments:         comments,
		BlockedReason:    task.BlockedReason,
		Path:             locator,
		RelativePath:     locator,
		LaunchEvaluation: tasklifecycle.LaunchEvaluation{
			Agent: task.Assignee,
		},
	}
}
