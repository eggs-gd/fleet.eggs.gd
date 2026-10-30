package taskflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Sentinel errors returned by Service validation.
var (
	ErrClosed            = errors.New("task service closed")
	ErrInvalidTransition = errors.New("task status transition rejected")
	// ErrMilestoneNotPersisted means a milestone patch was accepted by the
	// provider call but the task read back without it (the provider has no
	// milestone field), so reporting success would silently lose the change.
	ErrMilestoneNotPersisted = errors.New("task provider does not store milestones")
	ErrAlreadyClaimed        = errors.New("task is already claimed")
	ErrUnknownOutcome        = errors.New("unknown execution outcome")
	ErrEmptyTaskID           = errors.New("task id is required")
	ErrEmptyComment          = errors.New("comment text is required")
	ErrNilExecutionResult    = errors.New("execution result is required")
)

// Service is the single application-level TaskService implementation.
// Writes are serialized through a private command channel processed by one
// goroutine. Reads call the provider directly and do not enter the queue.
//
// Callers never receive the command channel. Provider-specific storage
// details stay behind TaskProvider.
type Service struct {
	provider TaskProvider
	cmds     chan taskCommand
	done     chan struct{}

	mu        sync.RWMutex
	closed    bool
	closeOnce sync.Once
}

// taskCommand is the private write-queue unit. Callers never see this type
// or its result channel — public methods enqueue and await typed results.
type taskCommand struct {
	ctx    context.Context
	run    func(context.Context, TaskProvider) (any, error)
	result chan taskCommandResult
}

type taskCommandResult struct {
	value any
	err   error
}

// Compile-time check that Service implements TaskService.
var _ TaskService = (*Service)(nil)

// NewService starts the serialized mutation worker against provider.
// Close the service when the process is shutting down.
func NewService(provider TaskProvider) *Service {
	if provider == nil {
		panic("taskflow: NewService requires a non-nil TaskProvider")
	}
	s := &Service{
		provider: provider,
		cmds:     make(chan taskCommand),
		done:     make(chan struct{}),
	}
	go s.loop()
	return s
}

// Close stops accepting new mutations and waits for the worker to exit.
// Concurrent Close calls are safe. In-flight commands still complete.
func (s *Service) Close() {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		close(s.cmds)
		s.mu.Unlock()
		<-s.done
	})
}

func (s *Service) loop() {
	defer close(s.done)
	for cmd := range s.cmds {
		ctx := cmd.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		value, err := cmd.run(ctx, s.provider)
		cmd.result <- taskCommandResult{value: value, err: err}
	}
}

// execWrite enqueues a mutation and waits for its typed result.
func (s *Service) execWrite(ctx context.Context, run func(context.Context, TaskProvider) (any, error)) (any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result := make(chan taskCommandResult, 1)
	cmd := taskCommand{ctx: ctx, run: run, result: result}

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, ErrClosed
	}
	select {
	case s.cmds <- cmd:
		s.mu.RUnlock()
	case <-ctx.Done():
		s.mu.RUnlock()
		return nil, ctx.Err()
	}

	select {
	case res := <-result:
		return res.value, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Create enqueues a task create and returns the persisted task.
func (s *Service) Create(ctx context.Context, input CreateTask) (Task, error) {
	value, err := s.execWrite(ctx, func(ctx context.Context, p TaskProvider) (any, error) {
		return p.Create(ctx, input)
	})
	if err != nil {
		return Task{}, err
	}
	task, _ := value.(Task)
	return task, nil
}

// Get reads one task by opaque id. Reads bypass the write queue.
func (s *Service) Get(ctx context.Context, id string) (Task, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Task{}, ErrEmptyTaskID
	}
	return s.provider.Get(ctx, id)
}

// List reads tasks matching filter. Reads bypass the write queue.
func (s *Service) List(ctx context.Context, filter TaskFilter) ([]Task, error) {
	return s.provider.List(ctx, filter)
}

// Claim moves a pickup-status task (todo|needs_rework) to doing inside the
// write queue. The pickup check and status write are one serialized mutation
// so concurrent launchers cannot both observe a free pickup status.
func (s *Service) Claim(ctx context.Context, id string) (Task, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Task{}, ErrEmptyTaskID
	}
	value, err := s.execWrite(ctx, func(ctx context.Context, p TaskProvider) (any, error) {
		return claimLocked(ctx, p, id)
	})
	if err != nil {
		return Task{}, err
	}
	task, _ := value.(Task)
	return task, nil
}

// Transition validates DOMAIN_MODEL status rules then persists the change.
// Optional TransitionMeta.Comment / Reason are recorded as a comment.
func (s *Service) Transition(ctx context.Context, id string, to Status, meta TransitionMeta) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrEmptyTaskID
	}
	_, err := s.execWrite(ctx, func(ctx context.Context, p TaskProvider) (any, error) {
		return nil, transitionLocked(ctx, p, id, to, meta)
	})
	return err
}

// Update persists a full task writeback through the serialized queue.
// Prefer Transition for status-only changes so DOMAIN_MODEL rules stay central.
func (s *Service) Update(ctx context.Context, task Task) error {
	locator := strings.TrimSpace(task.Locator)
	if locator == "" {
		locator = strings.TrimSpace(task.ID)
	}
	if locator == "" {
		return ErrEmptyTaskID
	}
	task.Locator = locator
	_, err := s.execWrite(ctx, func(ctx context.Context, p TaskProvider) (any, error) {
		return nil, p.Update(ctx, task)
	})
	return err
}

// Patch applies a combined manager/dashboard mutation in one queued write.
// Status changes are validated against DOMAIN_MODEL; comments use CommentAuthor
// when provided (via context for provider adapters).
func (s *Service) Patch(ctx context.Context, id string, patch PatchInput) (Task, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Task{}, ErrEmptyTaskID
	}
	value, err := s.execWrite(ctx, func(ctx context.Context, p TaskProvider) (any, error) {
		return patchLocked(ctx, p, id, patch)
	})
	if err != nil {
		return Task{}, err
	}
	task, _ := value.(Task)
	return task, nil
}

// AddComment appends an operator/worker-visible comment.
func (s *Service) AddComment(ctx context.Context, id string, text string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrEmptyTaskID
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return ErrEmptyComment
	}
	_, err := s.execWrite(ctx, func(ctx context.Context, p TaskProvider) (any, error) {
		return nil, p.AddComment(ctx, id, text)
	})
	return err
}

// ReportExecution applies the Finalizer outcome→status table in one queued
// mutation: comment first, then validated transition. Must not re-enter the
// public Transition/AddComment methods (that would deadlock the queue).
func (s *Service) ReportExecution(ctx context.Context, result ExecutionResult) error {
	if strings.TrimSpace(result.TaskID) == "" {
		return ErrNilExecutionResult
	}
	_, err := s.execWrite(ctx, func(ctx context.Context, p TaskProvider) (any, error) {
		return nil, reportExecutionLocked(ctx, p, result)
	})
	return err
}

func claimLocked(ctx context.Context, p TaskProvider, id string) (Task, error) {
	task, err := p.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	switch task.Status {
	case StatusTodo, StatusNeedsRework:
		// launchable pickup statuses
	default:
		return Task{}, fmt.Errorf("%w: status is %q", ErrAlreadyClaimed, task.Status)
	}
	if err := transitionLocked(ctx, p, id, StatusDoing, TransitionMeta{Actor: "launcher"}); err != nil {
		return Task{}, err
	}
	return p.Get(ctx, id)
}

func transitionLocked(ctx context.Context, p TaskProvider, id string, to Status, meta TransitionMeta) error {
	task, err := p.Get(ctx, id)
	if err != nil {
		return err
	}
	if !AllowedStatusTransition(task.Status, to) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, task.Status, to)
	}
	if task.Status != to {
		task.Status = to
		if strings.TrimSpace(meta.Reason) != "" && to == StatusBlocked {
			task.BlockedReason = strings.TrimSpace(meta.Reason)
		}
		// Clear Body so provider Update leaves the markdown body untouched.
		task.Body = ""
		if err := p.Update(ctx, task); err != nil {
			return err
		}
	}
	if comment := transitionComment(meta); comment != "" {
		if err := p.AddComment(ctx, id, comment); err != nil {
			return err
		}
	}
	return nil
}

func patchLocked(ctx context.Context, p TaskProvider, id string, patch PatchInput) (Task, error) {
	task, err := p.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}

	needsUpdate, err := applyPatchFields(&task, patch)
	if err != nil {
		return Task{}, err
	}
	if needsUpdate {
		updateCtx := ctx
		if patch.AllowStatusOverride {
			updateCtx = WithStatusOverride(ctx)
		}
		if projectPatch := strings.TrimSpace(patch.Project); projectPatch != "" {
			updateCtx = WithProjectPatch(updateCtx, projectPatch, strings.TrimSpace(patch.Repository))
		}
		if err := p.Update(updateCtx, task); err != nil {
			return Task{}, err
		}
	}

	locator := currentLocator(ctx, p, id, task.Ref)
	if comment := strings.TrimSpace(patch.Comment); comment != "" {
		commentCtx := WithCommentAuthor(ctx, patch.CommentAuthor)
		if err := p.AddComment(commentCtx, locator, comment); err != nil {
			return Task{}, err
		}
	}

	result, err := p.Get(ctx, locator)
	if err != nil {
		return Task{}, err
	}
	if patch.Milestone != nil && result.Milestone != strings.TrimSpace(*patch.Milestone) {
		return Task{}, ErrMilestoneNotPersisted
	}
	return result, nil
}

// applyPatchFields copies the set fields of patch onto task and reports whether
// anything needs saving. A status change is checked against the allowed moves.
func applyPatchFields(task *Task, patch PatchInput) (bool, error) {
	needsUpdate, err := applyStatusPatch(task, patch)
	if err != nil {
		return false, err
	}
	if patch.Priority != nil {
		task.Priority = *patch.Priority
		needsUpdate = true
	}
	if assignee := strings.TrimSpace(patch.Assignee); assignee != "" {
		task.Assignee = assignee
		needsUpdate = true
	}
	if project := strings.TrimSpace(patch.Project); project != "" {
		task.Project = project
		task.Repositories = nil
		if repo := strings.TrimSpace(patch.Repository); repo != "" {
			task.Repositories = []string{repo}
		}
		needsUpdate = true
	}
	if patch.Milestone != nil {
		task.Milestone = strings.TrimSpace(*patch.Milestone)
		needsUpdate = true
	}
	if patch.DependsOn != nil {
		task.DependsOn = NormalizeDependsOn(*patch.DependsOn)
		needsUpdate = true
	}
	if patch.Body != nil {
		task.Body = *patch.Body
		needsUpdate = true
	} else {
		// Omit body rewrite on field/status patches.
		task.Body = ""
	}
	return needsUpdate, nil
}

func applyStatusPatch(task *Task, patch PatchInput) (bool, error) {
	status := Status(strings.TrimSpace(string(patch.Status)))
	if status == "" {
		return false, nil
	}
	if !patch.AllowStatusOverride && !AllowedStatusTransition(task.Status, status) {
		return false, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, task.Status, status)
	}
	changed := task.Status != status
	task.Status = status
	if reason := strings.TrimSpace(patch.Reason); reason != "" && status == StatusBlocked {
		task.BlockedReason = reason
		changed = true
	}
	return changed, nil
}

// currentLocator is where the task lives now. A project move can change the
// storage locator, so when the old one is gone the durable ref finds it again.
func currentLocator(ctx context.Context, p TaskProvider, id string, ref string) string {
	if refreshed, err := p.Get(ctx, id); err == nil {
		if loc := strings.TrimSpace(refreshed.Locator); loc != "" {
			return loc
		}
		return id
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return id
	}
	listed, err := p.List(ctx, TaskFilter{Ref: ref})
	if err != nil || len(listed) != 1 {
		return id
	}
	if loc := strings.TrimSpace(listed[0].Locator); loc != "" {
		return loc
	}
	return id
}

func reportExecutionLocked(ctx context.Context, p TaskProvider, result ExecutionResult) error {
	to, ok := MapExecutionOutcomeToStatus(result.Outcome)
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownOutcome, result.Outcome)
	}
	comment := formatExecutionComment(result)
	if comment != "" {
		if err := p.AddComment(ctx, result.TaskID, comment); err != nil {
			return err
		}
	}
	return transitionLocked(ctx, p, result.TaskID, to, TransitionMeta{
		Actor: "finalizer",
	})
}

func transitionComment(meta TransitionMeta) string {
	parts := make([]string, 0, 2)
	if text := strings.TrimSpace(meta.Comment); text != "" {
		parts = append(parts, text)
	}
	if reason := strings.TrimSpace(meta.Reason); reason != "" {
		parts = append(parts, "Reason: "+reason)
	}
	if len(parts) == 0 {
		return ""
	}
	comment := strings.Join(parts, "\n\n")
	if actor := strings.TrimSpace(meta.Actor); actor != "" {
		comment = "[" + actor + "] " + comment
	}
	return comment
}

func formatExecutionComment(result ExecutionResult) string {
	var b strings.Builder
	b.WriteString("Execution outcome: `")
	b.WriteString(string(result.Outcome))
	b.WriteString("`")
	if agent := strings.TrimSpace(result.Agent); agent != "" {
		b.WriteString("\nAgent: `")
		b.WriteString(agent)
		b.WriteString("`")
	}
	if execID := strings.TrimSpace(result.ExecutionID); execID != "" {
		b.WriteString("\nExecution ID: `")
		b.WriteString(execID)
		b.WriteString("`")
	}
	if summary := strings.TrimSpace(result.Summary); summary != "" {
		b.WriteString("\n\n")
		b.WriteString(summary)
	}
	if question := strings.TrimSpace(result.Question); question != "" {
		b.WriteString("\n\nQuestion: ")
		b.WriteString(question)
	}
	if errText := strings.TrimSpace(result.Error); errText != "" {
		b.WriteString("\n\nError: ")
		b.WriteString(errText)
	}
	if len(result.Artifacts) > 0 {
		b.WriteString("\n\nArtifacts:")
		for _, artifact := range result.Artifacts {
			artifact = strings.TrimSpace(artifact)
			if artifact == "" {
				continue
			}
			b.WriteString("\n- `")
			b.WriteString(artifact)
			b.WriteString("`")
		}
	}
	if len(result.Tests) > 0 {
		b.WriteString("\n\nTests:")
		for _, test := range result.Tests {
			test = strings.TrimSpace(test)
			if test == "" {
				continue
			}
			b.WriteString("\n- `")
			b.WriteString(test)
			b.WriteString("`")
		}
	}
	return b.String()
}
