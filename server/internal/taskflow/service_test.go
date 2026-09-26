package taskflow

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// memoryProvider is a concurrency-hostile in-memory TaskProvider used to
// prove Service serializes writes and leaves reads concurrent.
type memoryProvider struct {
	mu        sync.Mutex
	tasks     map[string]Task
	writeLog  []string
	writeHold time.Duration // artificial delay inside each write
	active    atomic.Int32  // concurrent write depth; must never exceed 1 via Service
	maxActive atomic.Int32
	readCount atomic.Int32
}

func newMemoryProvider() *memoryProvider {
	return &memoryProvider{tasks: map[string]Task{}}
}

func (m *memoryProvider) beginWrite(op string) {
	n := m.active.Add(1)
	for {
		cur := m.maxActive.Load()
		if n <= cur || m.maxActive.CompareAndSwap(cur, n) {
			break
		}
	}
	// Delay before taking the data lock so concurrent Get/List can still
	// proceed — proving Service reads do not wait on the write queue.
	if m.writeHold > 0 {
		time.Sleep(m.writeHold)
	}
	m.mu.Lock()
	m.writeLog = append(m.writeLog, op)
}

func (m *memoryProvider) endWrite() {
	m.mu.Unlock()
	m.active.Add(-1)
}

func (m *memoryProvider) Create(_ context.Context, input CreateTask) (Task, error) {
	m.beginWrite("create:" + input.Title)
	defer m.endWrite()
	id := fmt.Sprintf("task-%d", len(m.tasks)+1)
	task := Task{
		ID:        id,
		Ref:       id,
		Title:     input.Title,
		Status:    input.Status,
		Project:   input.Project,
		Assignee:  input.Assignee,
		Priority:  input.Priority,
		DependsOn: append([]string{}, input.DependsOn...),
		Locator:   id,
		Body:      input.Description,
	}
	if task.Status == "" {
		task.Status = StatusTodo
	}
	m.tasks[id] = task
	return task, nil
}

func (m *memoryProvider) Get(_ context.Context, id string) (Task, error) {
	m.readCount.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	task, ok := m.tasks[id]
	if !ok {
		return Task{}, fmt.Errorf("task %q not found", id)
	}
	return task, nil
}

func (m *memoryProvider) List(_ context.Context, filter TaskFilter) ([]Task, error) {
	m.readCount.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		if filter.Project != "" && task.Project != filter.Project {
			continue
		}
		if filter.Status != "" && task.Status != filter.Status {
			continue
		}
		if filter.Assignee != "" && task.Assignee != filter.Assignee {
			continue
		}
		if filter.Ref != "" && task.Ref != filter.Ref {
			continue
		}
		out = append(out, task)
	}
	return out, nil
}

func (m *memoryProvider) Update(_ context.Context, task Task) error {
	m.beginWrite("update:" + task.Locator + ":" + string(task.Status))
	defer m.endWrite()
	if _, ok := m.tasks[task.Locator]; !ok {
		return fmt.Errorf("task %q not found", task.Locator)
	}
	m.tasks[task.Locator] = task
	return nil
}

func (m *memoryProvider) AddComment(ctx context.Context, id string, text string) error {
	m.beginWrite("comment:" + id)
	defer m.endWrite()
	task, ok := m.tasks[id]
	if !ok {
		return fmt.Errorf("task %q not found", id)
	}
	author := CommentAuthorFrom(ctx)
	if author == "" {
		author = "system"
	}
	task.Comments = append(task.Comments, Comment{Author: author, Text: text})
	m.tasks[id] = task
	return nil
}

func TestServiceSerializesConcurrentWrites(t *testing.T) {
	provider := newMemoryProvider()
	provider.writeHold = 5 * time.Millisecond
	svc := NewService(provider)
	defer svc.Close()

	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			_, err := svc.Create(context.Background(), CreateTask{
				Title:   fmt.Sprintf("task-%02d", i),
				Project: "core-eggs-gd",
				Status:  StatusTodo,
			})
			if err != nil {
				t.Errorf("Create: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := provider.maxActive.Load(); got != 1 {
		t.Fatalf("max concurrent provider writes = %d, want 1 (queue must serialize)", got)
	}
	if len(provider.writeLog) != n {
		t.Fatalf("writeLog len = %d, want %d", len(provider.writeLog), n)
	}
	listed, err := svc.List(context.Background(), TaskFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != n {
		t.Fatalf("List len = %d, want %d", len(listed), n)
	}
}

func TestServiceReadsBypassWriteQueue(t *testing.T) {
	provider := newMemoryProvider()
	svc := NewService(provider)
	defer svc.Close()

	task, err := svc.Create(context.Background(), CreateTask{Title: "seed", Status: StatusTodo})
	if err != nil {
		t.Fatal(err)
	}

	// Hold the write queue with a slow mutation while reads proceed.
	provider.writeHold = 50 * time.Millisecond
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(started)
		done <- svc.AddComment(context.Background(), task.Locator, "slow write")
	}()
	<-started
	time.Sleep(5 * time.Millisecond) // let the write enter the worker

	before := provider.readCount.Load()
	deadline := time.Now().Add(40 * time.Millisecond)
	reads := 0
	for time.Now().Before(deadline) {
		if _, err := svc.Get(context.Background(), task.Locator); err != nil {
			t.Fatalf("Get during write: %v", err)
		}
		reads++
	}
	if reads == 0 {
		t.Fatal("expected concurrent reads while a write was queued/running")
	}
	if provider.readCount.Load() <= before {
		t.Fatal("reads did not hit the provider while write was in flight")
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestServiceTransitionValidationCentralized(t *testing.T) {
	provider := newMemoryProvider()
	svc := NewService(provider)
	defer svc.Close()

	task, err := svc.Create(context.Background(), CreateTask{
		Title:  "lifecycle",
		Status: StatusBacklog,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = svc.Transition(context.Background(), task.Locator, StatusDoing, TransitionMeta{})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("backlog->doing err = %v, want ErrInvalidTransition", err)
	}

	if err := svc.Transition(context.Background(), task.Locator, StatusTodo, TransitionMeta{Actor: "owner"}); err != nil {
		t.Fatalf("backlog->todo: %v", err)
	}
	got, err := svc.Get(context.Background(), task.Locator)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusTodo {
		t.Fatalf("status = %s, want todo", got.Status)
	}

	if err := svc.Transition(context.Background(), task.Locator, StatusDoing, TransitionMeta{
		Actor:   "launcher",
		Comment: "claim",
	}); err != nil {
		t.Fatalf("todo->doing: %v", err)
	}
	got, err = svc.Get(context.Background(), task.Locator)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusDoing {
		t.Fatalf("status = %s, want doing", got.Status)
	}
	if len(got.Comments) != 1 {
		t.Fatalf("comments = %d, want 1", len(got.Comments))
	}
}

func TestServicePatchAllowsBlockedToNeedsReview(t *testing.T) {
	provider := newMemoryProvider()
	svc := NewService(provider)
	defer svc.Close()

	task, err := svc.Create(context.Background(), CreateTask{Title: "blocked recovery", Status: StatusBlocked})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := svc.Patch(context.Background(), task.Locator, PatchInput{
		Status:        StatusNeedsReview,
		Comment:       "Artifacts verified; move to review without re-run.",
		CommentAuthor: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != StatusNeedsReview {
		t.Fatalf("status = %s, want needs_review", updated.Status)
	}
	if len(updated.Comments) == 0 {
		t.Fatal("expected operator recovery comment")
	}
}

func TestServiceReportExecutionMapsOutcome(t *testing.T) {
	provider := newMemoryProvider()
	svc := NewService(provider)
	defer svc.Close()

	task, err := svc.Create(context.Background(), CreateTask{Title: "exec", Status: StatusDoing})
	if err != nil {
		t.Fatal(err)
	}

	err = svc.ReportExecution(context.Background(), ExecutionResult{
		TaskID:    task.Locator,
		Outcome:   ExecutionCompleted,
		Summary:   "shipped artifacts",
		Artifacts: []string{"service.go"},
		Tests:     []string{"go test ./internal/taskflow"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(context.Background(), task.Locator)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusNeedsReview {
		t.Fatalf("status = %s, want needs_review", got.Status)
	}
	if len(got.Comments) == 0 {
		t.Fatal("expected execution comment")
	}

	err = svc.ReportExecution(context.Background(), ExecutionResult{
		TaskID:  task.Locator,
		Outcome: ExecutionOutcome("nope"),
	})
	if !errors.Is(err, ErrUnknownOutcome) {
		t.Fatalf("err = %v, want ErrUnknownOutcome", err)
	}
}

func TestServiceCloseRejectsWrites(t *testing.T) {
	provider := newMemoryProvider()
	svc := NewService(provider)
	svc.Close()
	_, err := svc.Create(context.Background(), CreateTask{Title: "late"})
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("err = %v, want ErrClosed", err)
	}
}

func TestServicePatchCombinesFieldWrites(t *testing.T) {
	provider := newMemoryProvider()
	svc := NewService(provider)
	defer svc.Close()

	task, err := svc.Create(context.Background(), CreateTask{
		Title:    "patch me",
		Status:   StatusBacklog,
		Assignee: "unassigned",
		Priority: 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	priority := 1
	dependsOn := []string{"CORE-144", " CORE-144 ", ""}
	updated, err := svc.Patch(context.Background(), task.Locator, PatchInput{
		Status:        StatusTodo,
		Priority:      &priority,
		Assignee:      "cursor",
		DependsOn:     &dependsOn,
		Comment:       "routed through service",
		CommentAuthor: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != StatusTodo {
		t.Fatalf("status = %s, want todo", updated.Status)
	}
	if updated.Assignee != "cursor" {
		t.Fatalf("assignee = %q, want cursor", updated.Assignee)
	}
	if updated.Priority != 1 {
		t.Fatalf("priority = %d, want 1", updated.Priority)
	}
	if len(updated.DependsOn) != 1 || updated.DependsOn[0] != "CORE-144" {
		t.Fatalf("depends_on = %#v, want [CORE-144]", updated.DependsOn)
	}
	if len(updated.Comments) != 1 || updated.Comments[0].Author != "owner" {
		t.Fatalf("comments = %#v", updated.Comments)
	}

	cleared := []string{}
	clearedTask, err := svc.Patch(context.Background(), task.Locator, PatchInput{DependsOn: &cleared})
	if err != nil {
		t.Fatal(err)
	}
	if len(clearedTask.DependsOn) != 0 {
		t.Fatalf("depends_on after clear = %#v, want empty", clearedTask.DependsOn)
	}
}
