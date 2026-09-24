package taskprovider

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/lib/chain"
)

type ActiveSession struct {
	ClaimID         string
	ExecutionStatus string
	Status          string
}

type ObservedTaskChange struct {
	Event          TaskEvent
	Task           Task
	GuardApplied   bool
	ObservedStatus string
	Session        ActiveSession
}

type ListenerHooks struct {
	Before                func(locator string) (Task, bool)
	ActiveSessionForTask  func(Task) (ActiveSession, bool)
	RevertActiveExecution func(locator string, session ActiveSession) (Task, error)
	// OnTaskChange is optional store/projection feedback for non-Markdown
	// providers. Markdown owns projection via Provider hooks instead.
	OnTaskChange func(ObservedTaskChange) error
}

type ListenerProvider interface {
	Listener(interval time.Duration, hooks ListenerHooks, out chan<- TaskEvent) chain.Processor
}

func NewListenerService(provider Provider, interval time.Duration, hooks ListenerHooks, out chan<- TaskEvent) chain.Processor {
	if listener, ok := provider.(ListenerProvider); ok {
		return listener.Listener(interval, hooks, out)
	}
	return chain.NewEntryPoint(out, &pollingListener{
		provider: provider,
		source:   changeSource(provider),
		interval: interval,
		hooks:    hooks,
		seen:     map[string]string{},
		tasks:    map[string]taskflow.Task{},
	})
}

type pollingListener struct {
	provider Provider
	source   ChangeSource
	interval time.Duration
	hooks    ListenerHooks
	seen     map[string]string
	tasks    map[string]taskflow.Task
}

func (listener *pollingListener) Start(chout chan<- TaskEvent, ctx context.Context) {
	if listener.interval <= 0 {
		listener.interval = 45 * time.Second
	}
	listener.poll(chout, ctx)
	ticker := time.NewTicker(listener.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			listener.poll(chout, ctx)
		}
	}
}

func (listener *pollingListener) Decorate(event TaskEvent) (TaskEvent, error) {
	return event, nil
}

func (listener *pollingListener) Stop() {}

func (listener *pollingListener) poll(chout chan<- TaskEvent, ctx context.Context) {
	if listener.source != nil {
		listener.pollSource(chout, ctx)
		return
	}
	listener.pollGeneric(chout, ctx)
}

func (listener *pollingListener) pollSource(chout chan<- TaskEvent, ctx context.Context) {
	changes, err := listener.source.ObserveChanges()
	if err != nil {
		return
	}
	for _, change := range changes {
		event := listener.enrichEvent(change.Event, change.Task, taskflow.TaskEventSourcePlanePoll)
		if listener.hooks.OnTaskChange != nil {
			if err := listener.hooks.OnTaskChange(ObservedTaskChange{Event: event, Task: change.Task}); err != nil {
				continue
			}
		}
		select {
		case <-ctx.Done():
			return
		case chout <- event:
		}
	}
}

func (listener *pollingListener) pollGeneric(chout chan<- TaskEvent, ctx context.Context) {
	tasks, err := listener.provider.List()
	if err != nil {
		return
	}
	for i := range tasks {
		task := tasks[i]
		fingerprint := listenerFingerprint(task)
		locator := locatorForTask(task)
		if listener.seen[locator] == fingerprint {
			continue
		}
		var before *taskflow.Task
		if prev, ok := listener.tasks[locator]; ok {
			prevCopy := prev
			before = &prevCopy
		}
		listener.seen[locator] = fingerprint
		after := flowTaskFromLifecycle(task)
		listener.tasks[locator] = after

		event := taskflow.NewTaskEvent(locator, before, after, taskflow.TaskEventSourcePlanePoll)
		if listener.hooks.OnTaskChange != nil {
			if err := listener.hooks.OnTaskChange(ObservedTaskChange{Event: event, Task: task}); err != nil {
				continue
			}
		}
		select {
		case <-ctx.Done():
			return
		case chout <- event:
		}
	}
}

func (listener *pollingListener) enrichEvent(event TaskEvent, task Task, defaultSource TaskEventSource) TaskEvent {
	if event.Source == "" {
		event.Source = defaultSource
	}
	after := event.After
	if strings.TrimSpace(after.Locator) == "" && strings.TrimSpace(after.ID) == "" {
		after = flowTaskFromLifecycle(task)
	}
	before := event.Before
	if before == nil && listener.hooks.Before != nil {
		locator := strings.TrimSpace(event.TaskID)
		if locator == "" {
			locator = strings.TrimSpace(after.Locator)
		}
		if prior, ok := listener.hooks.Before(locator); ok {
			flow := flowTaskFromLifecycle(prior)
			before = &flow
		}
	}
	locator := strings.TrimSpace(event.TaskID)
	if locator == "" {
		locator = strings.TrimSpace(after.Locator)
	}
	enriched := taskflow.NewTaskEvent(locator, before, after, event.Source)
	if locator != "" {
		listener.tasks[locator] = after
	}
	return enriched
}

func changeSource(provider Provider) ChangeSource {
	source, _ := provider.(ChangeSource)
	return source
}

func locatorForTask(task Task) string {
	locator := strings.TrimSpace(task.RelativePath)
	if locator == "" {
		locator = strings.TrimSpace(task.Path)
	}
	return locator
}

func listenerFingerprint(task Task) string {
	return strings.Join([]string{
		task.Ref,
		task.Status,
		task.Assignee,
		task.UpdatedAt,
		strings.TrimSpace(task.BlockedReason),
		strconv.Itoa(task.Priority),
		strconv.Itoa(len(task.Comments)),
		strings.Join(task.DependsOn, ","),
	}, "|")
}

func flowTaskFromLifecycle(task Task) taskflow.Task {
	locator := locatorForTask(task)
	comments := make([]taskflow.Comment, 0, len(task.Comments))
	for _, c := range task.Comments {
		comments = append(comments, taskflow.Comment{
			Author:    c.Author,
			CreatedAt: c.CreatedAt,
			Text:      c.Text,
		})
	}
	return taskflow.Task{
		SchemaVersion:    task.SchemaVersion,
		ID:               task.ID,
		Ref:              task.Ref,
		Title:            task.Title,
		Type:             task.Type,
		Status:           taskflow.Status(task.Status),
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
		Locator:          locator,
	}
}
