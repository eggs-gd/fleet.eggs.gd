// Package taskflow defines the canonical application contracts for Core task
// orchestration and the serialized TaskService implementation.
//
// These types are the compile-checked contracts documented in
// _docs/ARCHITECTURE.md. This package must not import
// Markdown, Plane, fswalker, or manager packages — providers and callers
// depend inward on these contracts.
//
// Commands (requested mutations) and events (observed changes) are distinct.
// Service keeps the write queue private; the exported TaskCommand type is
// documentation of the internal shape only and is not used by callers.
package taskflow

import (
	"context"
	"strings"
)

// Status is a durable task lifecycle status from _docs/DOMAIN_MODEL.md.
type Status string

const (
	StatusBacklog     Status = "backlog"
	StatusNeedsRework Status = "needs_rework"
	StatusTodo        Status = "todo"
	StatusDoing       Status = "doing"
	StatusBlocked     Status = "blocked"
	StatusNeedsReview Status = "needs_review"
	StatusDone        Status = "done"
	StatusArchived    Status = "archived"
)

// Task is the provider-neutral in-memory task. Locator fields (Path /
// RelativePath) are opaque storage identities; generic code must not parse
// them as filesystem paths or Plane URLs.
type Task struct {
	SchemaVersion int
	ID            string
	Ref           string
	Title         string
	Type          string
	Status        Status
	Priority      int
	Project       string
	ProjectID     string
	WorkspaceID   string
	Repositories  []string
	// DependsOn lists prerequisite refs/ids/locators that must be done before
	// daemon launch (CORE-148). Empty means no hard dependency gate.
	DependsOn        []string
	Assignee         string
	AssignmentReason string
	Source           string
	CreatedAt        string
	UpdatedAt        string
	Summary          string
	Body             string
	Comments         []Comment
	BlockedReason    string
	// Locator is the opaque provider-scoped identity used with TaskProvider.
	Locator string
}

// Comment is one operator/worker-visible note attached to a task.
type Comment struct {
	Author    string
	CreatedAt string
	Text      string
}

// CreateTask is the application-level create input. It must not contain
// Markdown paths or Plane endpoints.
type CreateTask struct {
	Title            string
	Description      string
	Project          string
	Repository       string
	Status           Status
	Type             string
	Assignee         string
	Priority         int
	AssignmentReason string
	Source           string
	// DependsOn lists prerequisite refs that must be done before launch
	// (CORE-148). Empty means no hard dependency gate.
	DependsOn []string
}

// TaskFilter narrows List reads. Reads do not pass through the write queue.
type TaskFilter struct {
	Project  string
	Status   Status
	Assignee string
	Ref      string
}

// TransitionMeta carries optional audit fields for a status change.
type TransitionMeta struct {
	Actor   string
	Comment string
	Reason  string
}

// PatchInput is a provider-neutral partial update for manager/dashboard writes.
// Empty Status / Assignee / Project leave those fields unchanged. Priority, Body,
// and DependsOn are pointers so callers can distinguish "omit" from "set".
type PatchInput struct {
	Status     Status
	Priority   *int
	Assignee   string
	Project    string
	Repository string
	// DependsOn replaces the hard launch dependency list when non-nil
	// (CORE-148). An empty slice clears depends_on.
	DependsOn     *[]string
	Body          *string
	Comment       string
	CommentAuthor string
	Actor         string
	Reason        string
	// AllowStatusOverride skips DOMAIN_MODEL transition checks. Reserved for
	// runtime system guards (e.g. active-execution file revert); manager and
	// dashboard callers must leave this false.
	AllowStatusOverride bool
}

// TaskService is the single application-level API for task mutations.
// Manager, dashboard, launcher, and finalizer all use this interface.
// No caller talks directly to Markdown or Plane.
type TaskService interface {
	Create(ctx context.Context, input CreateTask) (Task, error)
	Get(ctx context.Context, id string) (Task, error)
	List(ctx context.Context, filter TaskFilter) ([]Task, error)
	Claim(ctx context.Context, id string) (Task, error)
	Transition(ctx context.Context, id string, to Status, meta TransitionMeta) error
	Update(ctx context.Context, task Task) error
	Patch(ctx context.Context, id string, patch PatchInput) (Task, error)
	AddComment(ctx context.Context, id string, text string) error
	ReportExecution(ctx context.Context, result ExecutionResult) error
}

// TaskProvider is the persistence adapter contract. MarkdownProvider and
// PlaneProvider both implement it. Provider-specific storage details must not
// leak into manager or launcher code.
type TaskProvider interface {
	Create(ctx context.Context, input CreateTask) (Task, error)
	Get(ctx context.Context, id string) (Task, error)
	List(ctx context.Context, filter TaskFilter) ([]Task, error)
	Update(ctx context.Context, task Task) error
	AddComment(ctx context.Context, id string, text string) error
}

// TaskCommand is the internal write-queue unit inside TaskService.
// Callers never receive the channel; public methods enqueue and await.
// This is a command (requested mutation), not an event.
type TaskCommand struct {
	Run    func(context.Context, TaskProvider) (any, error)
	Result chan TaskCommandResult
}

// TaskCommandResult is the reply for one queued mutation.
type TaskCommandResult struct {
	Value any
	Err   error
}

// TaskEventSource identifies which change-detection path produced an event.
// Contour 2 treats all sources the same; Source is for diagnostics only.
type TaskEventSource string

const (
	TaskEventSourceMarkdownFS   TaskEventSource = "markdown_fs"
	TaskEventSourcePlanePoll    TaskEventSource = "plane_poll"
	TaskEventSourcePlaneWebhook TaskEventSource = "plane_webhook"
	TaskEventSourceTaskService  TaskEventSource = "task_service"
)

// TaskEventKind is a migration-era classification. Contour 2 eligibility must
// be derived from Before/After (see IsDeleted / StatusChanged), not Kind alone.
// Kept until Contour 2 fully migrates off Kind (CORE-111 / CORE-112).
type TaskEventKind string

const (
	TaskEventUpserted          TaskEventKind = "upserted"
	TaskEventDeleted           TaskEventKind = "deleted"
	TaskEventStatusChanged     TaskEventKind = "status_changed"
	TaskEventRelaunchRequested TaskEventKind = "relaunch_requested"
)

// TaskEvent is a normalized "change already observed" signal for Contour 2.
// Markdown FS and Plane poll/webhook both publish into one TaskEvent channel.
// Execution consumes TaskEvent; it must not consume filesystem paths or Plane
// webhook payloads.
type TaskEvent struct {
	TaskID string
	Before *Task // nil when the task did not previously exist
	After  Task  // zero/tombstone After means delete (see IsDeleted)
	Source TaskEventSource
	// Kind is filled for migration compatibility. Prefer Before/After.
	Kind TaskEventKind
}

// NewTaskEvent builds a normalized change event with derived Kind.
func NewTaskEvent(taskID string, before *Task, after Task, source TaskEventSource) TaskEvent {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		taskID = strings.TrimSpace(after.Locator)
	}
	if taskID == "" {
		taskID = strings.TrimSpace(after.ID)
	}
	return TaskEvent{
		TaskID: taskID,
		Before: before,
		After:  after,
		Source: source,
		Kind:   DeriveTaskEventKind(before, after),
	}
}

// DeriveTaskEventKind classifies a Before/After delta for migration-era Kind.
func DeriveTaskEventKind(before *Task, after Task) TaskEventKind {
	if isTaskTombstone(after) {
		return TaskEventDeleted
	}
	if before != nil && before.Status != "" && before.Status != after.Status {
		return TaskEventStatusChanged
	}
	return TaskEventUpserted
}

// IsDeleted reports whether Contour 2 should skip this event as a deletion.
func (e TaskEvent) IsDeleted() bool {
	if e.Kind == TaskEventDeleted {
		return true
	}
	return isTaskTombstone(e.After) && e.Before != nil
}

// StatusChanged reports whether After.Status differs from Before.Status.
func (e TaskEvent) StatusChanged() bool {
	if e.Before == nil {
		return false
	}
	return e.Before.Status != "" && e.Before.Status != e.After.Status
}

// MayConsiderLaunch reports whether Contour 2 should evaluate this event for
// agent launch. Eligibility is derived only from Before/After (deleted /
// pickup status on After). Source is diagnostics-only and must not affect
// this decision — Markdown and Plane share the same path (CORE-112).
func (e TaskEvent) MayConsiderLaunch() bool {
	if e.IsDeleted() {
		return false
	}
	switch e.After.Status {
	case StatusTodo, StatusNeedsRework:
		return true
	default:
		return false
	}
}

func isTaskTombstone(task Task) bool {
	return strings.TrimSpace(task.Locator) == "" &&
		strings.TrimSpace(task.ID) == "" &&
		strings.TrimSpace(string(task.Status)) == ""
}

// ExecutionOutcome is a typed agent/runtime result. Workers report outcomes;
// they do not transition task lifecycle themselves.
type ExecutionOutcome string

const (
	ExecutionCompleted   ExecutionOutcome = "completed"
	ExecutionFailed      ExecutionOutcome = "failed"
	ExecutionNeedsInput  ExecutionOutcome = "needs_input"
	ExecutionNeedsRework ExecutionOutcome = "needs_rework"
	ExecutionBlocked     ExecutionOutcome = "blocked"
	ExecutionCancelled   ExecutionOutcome = "cancelled"
	ExecutionTimedOut    ExecutionOutcome = "timed_out"
	ExecutionOrphaned    ExecutionOutcome = "orphaned"
)

// ExecutionResult is the only channel through which agents/runtime report
// work assessment back to Core. Contour 3 (Finalizer) maps it onto
// TaskService writes — Contour 2 must only publish, never finalize.
type ExecutionResult struct {
	TaskID      string
	ExecutionID string // runtime claim / session id
	Agent       string
	Outcome     ExecutionOutcome
	Summary     string
	Tests       []string
	Artifacts   []string
	Question    string
	Error       string
}

// AllowedStatusTransition reports whether DOMAIN_MODEL permits from → to.
// A no-op (from == to) is always allowed. TaskService must use this table
// (or the equivalent tasklifecycle helper during migration) for all writes.
func AllowedStatusTransition(from, to Status) bool {
	if from == to {
		return true
	}
	allowed, ok := allowedStatusTransitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}

var allowedStatusTransitions = map[Status]map[Status]bool{
	StatusBacklog: {
		StatusTodo:     true,
		StatusArchived: true,
	},
	StatusTodo: {
		StatusDoing:    true,
		StatusBlocked:  true,
		StatusBacklog:  true,
		StatusArchived: true,
	},
	StatusNeedsRework: {
		StatusDoing:    true,
		StatusBlocked:  true,
		StatusTodo:     true,
		StatusArchived: true,
	},
	StatusDoing: {
		StatusNeedsReview: true,
		StatusNeedsRework: true,
		StatusBlocked:     true,
		StatusTodo:        true,
		StatusDone:        true,
	},
	StatusBlocked: {
		StatusNeedsReview: true,
		StatusNeedsRework: true,
		StatusTodo:        true,
		StatusArchived:    true,
	},
	StatusNeedsReview: {
		StatusNeedsRework: true,
		StatusTodo:        true,
		StatusDone:        true,
		StatusArchived:    true,
	},
	StatusDone: {
		StatusArchived: true,
	},
	StatusArchived: {
		StatusBacklog: true,
	},
}

// MapExecutionOutcomeToStatus is the Contour 3 Finalizer decision table for
// ReportExecution (_docs/AGENT_LAUNCHER.md). Bot-produced completed
// work stops at needs_review. cancelled maps to blocked (not pickup) —
// Fleet/LAUNCH_POLICY.md does not choose a return-to-pickup rule for MVP.
func MapExecutionOutcomeToStatus(outcome ExecutionOutcome) (Status, bool) {
	switch outcome {
	case ExecutionCompleted:
		return StatusNeedsReview, true
	case ExecutionFailed, ExecutionNeedsInput, ExecutionBlocked,
		ExecutionTimedOut, ExecutionCancelled, ExecutionOrphaned:
		return StatusBlocked, true
	case ExecutionNeedsRework:
		return StatusNeedsRework, true
	default:
		return "", false
	}
}
