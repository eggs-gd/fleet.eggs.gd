// Package executionfinalizer maps completed execution results to durable task
// updates through TaskService.ReportExecution. It is an independent consumer of
// ExecutionResult, not a trailing step of the execution processor chain.
package executionfinalizer

import (
	"context"
	"errors"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/eventbus"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/lib/chain"
)

const (
	eventTaskNeedsReview    = "task.needs_review"
	eventTaskNeedsAttention = "task.needs_attention"
)

// Publisher accepts events after a task status is already durable.
// Publish errors must not roll that status back.
type Publisher interface {
	Publish(eventbus.Event) error
}

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
	service   taskflow.TaskService
	reload    Reload
	upsert    Upsert
	onSkip    OnSkip
	onFail    OnFail
	publisher Publisher
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

// WithPublisher attaches the event bus. Nil leaves finalization silent.
func (f *Finalizer) WithPublisher(publisher Publisher) *Finalizer {
	f.publisher = publisher
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

	applied := result
	err = f.service.ReportExecution(ctx, applied)
	if errors.Is(err, taskflow.ErrUnknownOutcome) {
		applied = taskflow.ExecutionResult{
			TaskID:      locator,
			ExecutionID: result.ExecutionID,
			Agent:       result.Agent,
			Outcome:     taskflow.ExecutionFailed,
			Summary:     "Agent execution succeeded but returned invalid result outcome `" + strings.TrimSpace(string(result.Outcome)) + "`.",
			Error:       "invalid execution outcome",
		}
		err = f.service.ReportExecution(ctx, applied)
	}
	if err != nil {
		if f.onFail != nil {
			f.onFail(locator, result, err)
		}
		return Task{}, err
	}
	updated, err := f.load(locator)
	f.publish(applied, updated.Ref)
	if err != nil {
		return Task{}, err
	}
	if f.upsert != nil {
		f.upsert(updated)
	}
	return updated, nil
}

// publish announces the outcome. ref is the task's human ref (FLET-12); the
// event's task_id stays the provider locator, and task_ref carries the ref for
// readers such as manager_events.
func (f *Finalizer) publish(result taskflow.ExecutionResult, ref string) {
	if f.publisher == nil {
		return
	}
	event, ok := taskEvent(result)
	if !ok {
		return
	}
	if ref = strings.TrimSpace(ref); ref != "" {
		event.Fields["task_ref"] = ref
	}
	_ = f.publisher.Publish(event)
}

func taskEvent(result taskflow.ExecutionResult) (eventbus.Event, bool) {
	taskID := strings.TrimSpace(result.TaskID)
	fields := map[string]string{}
	if taskID != "" {
		fields["task_id"] = taskID
	}
	switch result.Outcome {
	case taskflow.ExecutionCompleted:
		return eventbus.Event{
			Channel: eventbus.ChannelTask,
			Type:    eventTaskNeedsReview,
			Text:    taskNotice(taskID, "is ready for review", result.Summary),
			Fields:  fields,
		}, true
	case taskflow.ExecutionNeedsInput, taskflow.ExecutionFailed, taskflow.ExecutionBlocked,
		taskflow.ExecutionTimedOut, taskflow.ExecutionCancelled, taskflow.ExecutionOrphaned:
		what := "is blocked"
		if result.Outcome == taskflow.ExecutionNeedsInput {
			what = "needs a decision before continuing"
		}
		return eventbus.Event{
			Channel: eventbus.ChannelTask,
			Type:    eventTaskNeedsAttention,
			Text:    taskNotice(taskID, what, attentionDetail(result)),
			Fields:  fields,
		}, true
	default:
		return eventbus.Event{}, false
	}
}

func attentionDetail(result taskflow.ExecutionResult) string {
	var parts []string
	for _, part := range []string{result.Question, result.Summary, result.Error} {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if len(parts) > 0 && parts[len(parts)-1] == part {
			continue
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "\n")
}

func taskNotice(taskID, what, detail string) string {
	subject := taskID
	if subject == "" {
		subject = "A task"
	} else {
		subject = "Task " + subject
	}
	text := subject + " " + what + "."
	if detail = strings.TrimSpace(detail); detail != "" {
		text += "\n" + detail
	}
	return text
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
