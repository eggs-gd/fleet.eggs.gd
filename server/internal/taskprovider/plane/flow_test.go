package plane

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider"
)

func TestFlowProviderCreateUpdateCommentTransition(t *testing.T) {
	fake := newFakePlane()
	server := fake.server(t)
	defer server.Close()
	provider := newTestProvider(t, server.URL)
	flow := provider.Flow()
	svc := taskprovider.NewService(provider)
	defer svc.Close()

	created, err := svc.Create(context.Background(), taskflow.CreateTask{
		Title:       "Plane provider contract",
		Description: "Prove create/update/comment/transition through taskflow.TaskProvider.",
		Project:     "core-eggs-gd",
		Repository:  "core.eggs.gd",
		Status:      taskflow.StatusTodo,
		Type:        "feature",
		Assignee:    "cursor",
		Priority:    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(created.Ref, "CORE-") || created.Locator == "" {
		t.Fatalf("created = %+v", created)
	}
	if !strings.HasPrefix(created.Locator, "plane://") {
		t.Fatalf("locator = %q, want plane://…", created.Locator)
	}

	if err := svc.AddComment(context.Background(), created.Locator, "operator note via provider"); err != nil {
		t.Fatal(err)
	}

	if err := svc.Transition(context.Background(), created.Locator, taskflow.StatusDoing, taskflow.TransitionMeta{
		Actor:   "launcher",
		Comment: "claim",
	}); err != nil {
		t.Fatal(err)
	}

	err = svc.Transition(context.Background(), created.Locator, taskflow.StatusArchived, taskflow.TransitionMeta{})
	if !errors.Is(err, taskflow.ErrInvalidTransition) {
		t.Fatalf("invalid transition err = %v", err)
	}

	got, err := flow.Get(context.Background(), created.Locator)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != taskflow.StatusDoing {
		t.Fatalf("status = %s, want doing", got.Status)
	}
	if len(got.Comments) < 2 {
		t.Fatalf("comments = %d, want >= 2", len(got.Comments))
	}

	got.Status = taskflow.StatusNeedsReview
	if err := flow.Update(context.Background(), got); err != nil {
		t.Fatal(err)
	}
	reloaded, err := provider.Load(created.Locator)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Status != "needs_review" {
		t.Fatalf("reload status = %q", reloaded.Status)
	}

	listed, err := flow.List(context.Background(), taskflow.TaskFilter{
		Project: "core-eggs-gd",
		Status:  taskflow.StatusNeedsReview,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("list = %d, want 1", len(listed))
	}
}

func TestFlowProviderReportExecution(t *testing.T) {
	fake := newFakePlane()
	server := fake.server(t)
	defer server.Close()
	provider := newTestProvider(t, server.URL)
	svc := taskprovider.NewService(provider)
	defer svc.Close()

	created, err := svc.Create(context.Background(), taskflow.CreateTask{
		Title:       "Finalize via Plane",
		Description: "Exercise ReportExecution through Plane Flow.",
		Project:     "core-eggs-gd",
		Status:      taskflow.StatusTodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Claim(context.Background(), created.Locator); err != nil {
		t.Fatal(err)
	}
	if err := svc.ReportExecution(context.Background(), taskflow.ExecutionResult{
		TaskID:  created.Locator,
		Outcome: taskflow.ExecutionCompleted,
		Summary: "Plane-backed TaskService path works.",
	}); err != nil {
		t.Fatal(err)
	}
	got, err := provider.Flow().Get(context.Background(), created.Locator)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != taskflow.StatusNeedsReview {
		t.Fatalf("status = %s, want needs_review", got.Status)
	}
}
