package taskprovider

import (
	"context"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

type fakeListProvider struct {
	tasks []tasklifecycle.Task
}

func (f *fakeListProvider) Type() string { return "fake" }
func (f *fakeListProvider) List() ([]tasklifecycle.Task, error) {
	return append([]tasklifecycle.Task{}, f.tasks...), nil
}
func (f *fakeListProvider) Load(string) (tasklifecycle.Task, error) {
	return tasklifecycle.Task{}, nil
}
func (f *fakeListProvider) CreateFromRequest(tasklifecycle.TaskCreateRequest) (tasklifecycle.Task, error) {
	return tasklifecycle.Task{}, nil
}
func (f *fakeListProvider) Mutate(tasklifecycle.TaskPatch) (tasklifecycle.Task, error) {
	return tasklifecycle.Task{}, nil
}
func (f *fakeListProvider) Claim(string) (tasklifecycle.Task, error) {
	return tasklifecycle.Task{}, nil
}

var _ Provider = (*fakeListProvider)(nil)

func TestPollingListenerDedupesUnchangedTasksAcrossPolls(t *testing.T) {
	provider := &fakeListProvider{tasks: []tasklifecycle.Task{{
		RelativePath: "plane://eggs_gd/proj-1/item-1",
		Ref:          "CORE-1",
		Status:       "todo",
		Priority:     3,
		Assignee:     "claude",
		UpdatedAt:    "2026-08-03T00:00:00Z",
	}}}
	listener := &pollingListener{
		provider: provider,
		seen:     map[string]string{},
		tasks:    map[string]taskflow.Task{},
	}
	ctx := context.Background()
	chout := make(chan TaskEvent, 4)

	listener.poll(chout, ctx)
	listener.poll(chout, ctx)
	listener.poll(chout, ctx)
	close(chout)

	count := 0
	for range chout {
		count++
	}
	if count != 1 {
		t.Fatalf("expected exactly one emission for an unchanged task across polls, got %d", count)
	}
}

func TestPollingListenerReemitsOnStatusChange(t *testing.T) {
	provider := &fakeListProvider{tasks: []tasklifecycle.Task{{
		RelativePath: "plane://eggs_gd/proj-1/item-1",
		Ref:          "CORE-1",
		Status:       "todo",
		UpdatedAt:    "2026-08-03T00:00:00Z",
	}}}
	listener := &pollingListener{
		provider: provider,
		seen:     map[string]string{},
		tasks:    map[string]taskflow.Task{},
	}
	ctx := context.Background()
	chout := make(chan TaskEvent, 4)

	listener.poll(chout, ctx)
	provider.tasks[0].Status = "doing"
	provider.tasks[0].UpdatedAt = "2026-08-03T00:05:00Z"
	listener.poll(chout, ctx)
	close(chout)

	var events []TaskEvent
	for event := range chout {
		events = append(events, event)
	}
	if len(events) != 2 {
		t.Fatalf("expected re-emission after a status change, got %d emissions", len(events))
	}
	if events[0].Kind != taskflow.TaskEventUpserted {
		t.Fatalf("first kind = %q, want upserted", events[0].Kind)
	}
	if events[0].Source != taskflow.TaskEventSourcePlanePoll {
		t.Fatalf("first Source = %q, want plane_poll", events[0].Source)
	}
	if events[0].Before != nil {
		t.Fatalf("first Before = %#v, want nil", events[0].Before)
	}
	if events[0].After.Status != taskflow.StatusTodo {
		t.Fatalf("first After.Status = %q", events[0].After.Status)
	}
	if events[1].Kind != taskflow.TaskEventStatusChanged {
		t.Fatalf("second kind = %q, want status_changed", events[1].Kind)
	}
	if events[1].Before == nil || events[1].Before.Status != taskflow.StatusTodo {
		t.Fatalf("second Before = %#v, want todo", events[1].Before)
	}
	if events[1].After.Status != taskflow.StatusDoing {
		t.Fatalf("second After.Status = %q", events[1].After.Status)
	}
	if events[0].TaskID != "plane://eggs_gd/proj-1/item-1" {
		t.Fatalf("TaskID = %q", events[0].TaskID)
	}
}
