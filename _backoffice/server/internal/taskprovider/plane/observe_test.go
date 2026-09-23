package plane

import (
	"strings"
	"testing"

	"github.com/eggs-gd/core.eggs.gd/internal/taskflow"
)

func TestObservePollDedupesAndClassifiesStatusChange(t *testing.T) {
	fake := newFakePlane()
	server := fake.server(t)
	defer server.Close()
	provider := newTestProvider(t, server.URL)

	created, err := provider.CreateFromRequest(TaskCreateRequest{
		Title:   "Poll observe",
		Request: "n/a",
		Project: "core-eggs-gd",
		Status:  "todo",
	})
	if err != nil {
		t.Fatal(err)
	}

	tracker := NewPollTracker()
	first, err := provider.ObservePoll(tracker)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 {
		t.Fatalf("first poll = %d changes, want 1", len(first))
	}
	if first[0].Event.Kind != taskflow.TaskEventUpserted {
		t.Fatalf("kind = %q, want upserted", first[0].Event.Kind)
	}
	if first[0].Event.Source != taskflow.TaskEventSourcePlanePoll {
		t.Fatalf("Source = %q, want plane_poll", first[0].Event.Source)
	}
	if first[0].Event.Before != nil {
		t.Fatalf("Before = %#v, want nil", first[0].Event.Before)
	}
	if first[0].Event.After.Status != taskflow.StatusTodo {
		t.Fatalf("After.Status = %q", first[0].Event.After.Status)
	}
	if first[0].Event.TaskID != created.RelativePath {
		t.Fatalf("TaskID = %q, want %q", first[0].Event.TaskID, created.RelativePath)
	}

	second, err := provider.ObservePoll(tracker)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 0 {
		t.Fatalf("unchanged poll = %d changes, want 0", len(second))
	}

	if _, err := provider.Mutate(TaskPatch{Path: created.RelativePath, Status: "doing"}); err != nil {
		t.Fatal(err)
	}
	third, err := provider.ObservePoll(tracker)
	if err != nil {
		t.Fatal(err)
	}
	if len(third) != 1 {
		t.Fatalf("status-change poll = %d changes, want 1", len(third))
	}
	if third[0].Event.Kind != taskflow.TaskEventStatusChanged {
		t.Fatalf("kind = %q, want status_changed", third[0].Event.Kind)
	}
	if third[0].Event.Before == nil || third[0].Event.Before.Status != taskflow.StatusTodo {
		t.Fatalf("Before = %#v, want todo", third[0].Event.Before)
	}
	if third[0].Event.After.Status != taskflow.StatusDoing {
		t.Fatalf("After.Status = %q", third[0].Event.After.Status)
	}
	if third[0].Task.Status != "doing" {
		t.Fatalf("task status = %q", third[0].Task.Status)
	}
}

func TestObserveWebhookReloadsWorkItem(t *testing.T) {
	fake := newFakePlane()
	server := fake.server(t)
	defer server.Close()
	provider := newTestProvider(t, server.URL)

	created, err := provider.CreateFromRequest(TaskCreateRequest{
		Title:   "Webhook observe",
		Request: "n/a",
		Project: "core-eggs-gd",
	})
	if err != nil {
		t.Fatal(err)
	}
	id, err := workItemIDFromLocator(created.RelativePath)
	if err != nil {
		t.Fatal(err)
	}

	payloads := []string{
		`{"event":"issue.updated","data":{"id":"` + id + `"}}`,
		`{"issue":{"id":"` + id + `"}}`,
		`{"id":"` + id + `","name":"Webhook observe"}`,
	}
	for _, payload := range payloads {
		observed, err := provider.ObserveWebhook([]byte(payload))
		if err != nil {
			t.Fatalf("ObserveWebhook(%s): %v", payload, err)
		}
		if observed.Event.TaskID != created.RelativePath {
			t.Fatalf("TaskID = %q, want %q", observed.Event.TaskID, created.RelativePath)
		}
		if observed.Event.Kind != taskflow.TaskEventUpserted {
			t.Fatalf("kind = %q", observed.Event.Kind)
		}
		if observed.Event.Source != taskflow.TaskEventSourcePlaneWebhook {
			t.Fatalf("Source = %q, want plane_webhook", observed.Event.Source)
		}
		if observed.Event.After.Ref != created.Ref {
			t.Fatalf("After.Ref = %q", observed.Event.After.Ref)
		}
		if observed.Task.Ref != created.Ref {
			t.Fatalf("ref = %q, want %q", observed.Task.Ref, created.Ref)
		}
	}
}

func TestObserveWebhookRejectsUnknownPayload(t *testing.T) {
	fake := newFakePlane()
	server := fake.server(t)
	defer server.Close()
	provider := newTestProvider(t, server.URL)

	_, err := provider.ObserveWebhook([]byte(`{"event":"ping"}`))
	if err == nil {
		t.Fatal("expected error for payload without work-item id")
	}
	if !strings.Contains(err.Error(), "work-item id") {
		t.Fatalf("err = %v", err)
	}
}

func TestWorkItemIDFromWebhookShapes(t *testing.T) {
	id, err := workItemIDFromWebhook([]byte(`{"data":{"work_item":{"id":"abc-123"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if id != "abc-123" {
		t.Fatalf("id = %q", id)
	}
}
