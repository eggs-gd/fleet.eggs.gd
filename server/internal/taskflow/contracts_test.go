package taskflow

import "testing"

func TestAllowedStatusTransitionMatchesDomainModel(t *testing.T) {
	cases := []struct {
		from, to Status
		ok       bool
	}{
		{StatusBacklog, StatusTodo, true},
		{StatusBacklog, StatusDoing, false},
		{StatusTodo, StatusDoing, true},
		{StatusDoing, StatusNeedsReview, true},
		{StatusDoing, StatusArchived, false},
		{StatusNeedsReview, StatusDone, true},
		{StatusDone, StatusArchived, true},
		{StatusArchived, StatusBacklog, true},
		{StatusBlocked, StatusTodo, true},
		{StatusBlocked, StatusNeedsReview, true},
		{StatusBlocked, StatusDoing, false},
		{StatusTodo, StatusTodo, true},
	}
	for _, tc := range cases {
		if got := AllowedStatusTransition(tc.from, tc.to); got != tc.ok {
			t.Fatalf("%s -> %s: got %v want %v", tc.from, tc.to, got, tc.ok)
		}
	}
}

func TestMapExecutionOutcomeToStatus(t *testing.T) {
	cases := []struct {
		outcome ExecutionOutcome
		status  Status
		ok      bool
	}{
		{ExecutionCompleted, StatusNeedsReview, true},
		{ExecutionFailed, StatusBlocked, true},
		{ExecutionNeedsInput, StatusBlocked, true},
		{ExecutionNeedsRework, StatusNeedsRework, true},
		{ExecutionBlocked, StatusBlocked, true},
		{ExecutionCancelled, StatusBlocked, true},
		{ExecutionTimedOut, StatusBlocked, true},
		{ExecutionOrphaned, StatusBlocked, true},
		{ExecutionOutcome("nope"), "", false},
	}
	for _, tc := range cases {
		got, ok := MapExecutionOutcomeToStatus(tc.outcome)
		if ok != tc.ok || got != tc.status {
			t.Fatalf("%s: got (%s,%v) want (%s,%v)", tc.outcome, got, ok, tc.status, tc.ok)
		}
	}
}

func TestNewTaskEventDerivesBeforeAfterSourceAndKind(t *testing.T) {
	after := Task{Locator: "Work/x/tasks/a.md", ID: "work-a", Ref: "CORE-111", Status: StatusTodo}
	event := NewTaskEvent("", nil, after, TaskEventSourceMarkdownFS)
	if event.TaskID != "Work/x/tasks/a.md" {
		t.Fatalf("TaskID = %q", event.TaskID)
	}
	if event.Source != TaskEventSourceMarkdownFS {
		t.Fatalf("Source = %q", event.Source)
	}
	if event.Before != nil {
		t.Fatalf("Before = %#v", event.Before)
	}
	if event.After.Ref != "CORE-111" || event.Kind != TaskEventUpserted {
		t.Fatalf("event = %#v", event)
	}

	before := &Task{Locator: after.Locator, Status: StatusTodo}
	after.Status = StatusDoing
	changed := NewTaskEvent(after.Locator, before, after, TaskEventSourcePlanePoll)
	if !changed.StatusChanged() || changed.Kind != TaskEventStatusChanged {
		t.Fatalf("changed = %#v", changed)
	}
	if changed.Source != TaskEventSourcePlanePoll {
		t.Fatalf("Source = %q", changed.Source)
	}

	deleted := NewTaskEvent(after.Locator, before, Task{}, TaskEventSourceMarkdownFS)
	if !deleted.IsDeleted() || deleted.Kind != TaskEventDeleted {
		t.Fatalf("deleted = %#v", deleted)
	}
}

func TestMayConsiderLaunchFromBeforeAfter(t *testing.T) {
	todo := Task{Locator: "loc", Status: StatusTodo, Assignee: "cursor"}
	rework := Task{Locator: "loc", Status: StatusNeedsRework, Assignee: "cursor"}
	doing := Task{Locator: "loc", Status: StatusDoing, Assignee: "cursor"}
	backlog := Task{Locator: "loc", Status: StatusBacklog, Assignee: "cursor"}

	cases := []struct {
		name   string
		event  TaskEvent
		want   bool
		source TaskEventSource // Source must never change the decision
	}{
		{"new todo", NewTaskEvent("loc", nil, todo, TaskEventSourceMarkdownFS), true, TaskEventSourceMarkdownFS},
		{"plane new todo", NewTaskEvent("loc", nil, todo, TaskEventSourcePlanePoll), true, TaskEventSourcePlanePoll},
		{"webhook needs_rework", NewTaskEvent("loc", &doing, rework, TaskEventSourcePlaneWebhook), true, TaskEventSourcePlaneWebhook},
		{"backlog to todo", NewTaskEvent("loc", &backlog, todo, TaskEventSourceTaskService), true, TaskEventSourceTaskService},
		{"todo to doing", NewTaskEvent("loc", &todo, doing, TaskEventSourceMarkdownFS), false, TaskEventSourceMarkdownFS},
		{"still doing", NewTaskEvent("loc", &doing, doing, TaskEventSourcePlanePoll), false, TaskEventSourcePlanePoll},
		{"deleted", NewTaskEvent("loc", &todo, Task{}, TaskEventSourceMarkdownFS), false, TaskEventSourceMarkdownFS},
		{"backlog stays", NewTaskEvent("loc", nil, backlog, TaskEventSourceMarkdownFS), false, TaskEventSourceMarkdownFS},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.event.MayConsiderLaunch(); got != tc.want {
				t.Fatalf("MayConsiderLaunch() = %v, want %v (event=%#v)", got, tc.want, tc.event)
			}
			// Same Before/After with a different Source must agree.
			alt := tc.event
			if tc.source == TaskEventSourceMarkdownFS {
				alt.Source = TaskEventSourcePlanePoll
			} else {
				alt.Source = TaskEventSourceMarkdownFS
			}
			if alt.MayConsiderLaunch() != tc.want {
				t.Fatalf("Source must not affect MayConsiderLaunch; got %v with Source=%q", alt.MayConsiderLaunch(), alt.Source)
			}
		})
	}
}
