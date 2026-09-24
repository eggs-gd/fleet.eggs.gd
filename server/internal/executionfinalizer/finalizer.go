// Package executionfinalizer maps completed execution results to durable task
// updates through TaskService.ReportExecution. It is an independent consumer of
// ExecutionResult, not a trailing step of the execution processor chain.
package executionfinalizer

import (
	"context"
	"errors"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/lib/chain"
)

// Task is the lifecycle projection refreshed after finalization.
type Task = tasklifecycle.Task

// Reload loads the current task after ReportExecution.
type Reload func(string) (Task, error)

// Upsert refreshes the dashboard projection after a successful finalize.
type Upsert func(Task)

// OnSkip is called when finalization is skipped because the task already left doing.
type OnSkip func(task Task, result taskflow.ExecutionResult)

// OnFail is called when ReportExecution fails.
type OnFail func(locator string, result taskflow.ExecutionResult, err error)

// Finalizer applies ExecutionResult outcomes through TaskService only.
type Finalizer struct {
	service taskflow.TaskService
	reload  Reload
	upsert  Upsert
	onSkip  OnSkip
	onFail  OnFail
}

// New builds a Finalizer. reload/upsert/onSkip/onFail may be nil.
func New(service taskflow.TaskService, reload Reload, upsert Upsert) *Finalizer {
	return &Finalizer{service: service, reload: reload, upsert: upsert}
}

// WithHooks attaches optional skip/fail observers used by the composition root.
func (f *Finalizer) WithHooks(onSkip OnSkip, onFail OnFail) *Finalizer {
	f.onSkip = onSkip
	f.onFail = onFail
	return f
}

// NewProcessor wraps Finalizer as an independent chain.Processor consumer.
func NewProcessor(finalizer *Finalizer, in <-chan taskflow.ExecutionResult, out chan<- taskflow.ExecutionResult) chain.Processor {
	return chain.NewDecorator(in, out, finalizer)
}

func (f *Finalizer) Decorate(result taskflow.ExecutionResult) (taskflow.ExecutionResult, error) {
	if _, err := f.Apply(result); err != nil {
		return result, err
	}
	return result, chain.ErrSkippedItem
}

func (f *Finalizer) Stop() {}

// Apply writes one ExecutionResult through TaskService.
func (f *Finalizer) Apply(result taskflow.ExecutionResult) (Task, error) {
	ctx := context.Background()
	locator := strings.TrimSpace(result.TaskID)
	if locator == "" {
		return Task{}, taskflow.ErrNilExecutionResult
	}
	result.TaskID = locator

	current, err := f.service.Get(ctx, locator)
	if err == nil && current.Status != taskflow.StatusDoing {
		loaded := taskFromFlow(current)
		if f.reload != nil {
			if providerLoaded, loadErr := f.reload(locator); loadErr == nil {
				loaded = providerLoaded
			}
		}
		if f.onSkip != nil {
			f.onSkip(loaded, result)
		}
		return loaded, nil
	}

	err = f.service.ReportExecution(ctx, result)
	if errors.Is(err, taskflow.ErrUnknownOutcome) {
		fallback := taskflow.ExecutionResult{
			TaskID:      locator,
			ExecutionID: result.ExecutionID,
			Agent:       result.Agent,
			Outcome:     taskflow.ExecutionFailed,
			Summary:     "Agent execution succeeded but returned invalid result outcome `" + strings.TrimSpace(string(result.Outcome)) + "`.",
			Error:       "invalid execution outcome",
		}
		err = f.service.ReportExecution(ctx, fallback)
	}
	if err != nil {
		if f.onFail != nil {
			f.onFail(locator, result, err)
		}
		return Task{}, err
	}

	updated, err := f.load(locator)
	if err != nil {
		return Task{}, err
	}
	if f.upsert != nil {
		f.upsert(updated)
	}
	return updated, nil
}

func (f *Finalizer) load(locator string) (Task, error) {
	if f.reload != nil {
		if task, err := f.reload(locator); err == nil {
			return task, nil
		}
	}
	flowTask, err := f.service.Get(context.Background(), locator)
	if err != nil {
		return Task{}, err
	}
	return taskFromFlow(flowTask), nil
}

func taskFromFlow(task taskflow.Task) Task {
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
	}
}
