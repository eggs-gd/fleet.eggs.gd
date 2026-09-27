package executionfinalizer_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/eventbus"
	"github.com/eggs-gd/fleet.eggs.gd/internal/executionfinalizer"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider/markdown"
)

func TestFinalizerMapsOutcomesThroughTaskService(t *testing.T) {
	cases := []struct {
		name       string
		outcome    taskflow.ExecutionOutcome
		wantStatus string
		sub        string
	}{
		{"completed", taskflow.ExecutionCompleted, "needs_review", "completed"},
		{"failed", taskflow.ExecutionFailed, "blocked", "failed"},
		{"timed_out", taskflow.ExecutionTimedOut, "blocked", "timed_out"},
		{"cancelled", taskflow.ExecutionCancelled, "blocked", "cancelled"},
		{"needs_input", taskflow.ExecutionNeedsInput, "blocked", "needs_input"},
		{"orphaned", taskflow.ExecutionOrphaned, "blocked", "orphaned"},
		{"needs_rework", taskflow.ExecutionNeedsRework, "needs_rework", "needs_rework"},
		{"blocked", taskflow.ExecutionBlocked, "blocked", "blocked"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-finalizer-"+tc.name+".md")
			writeDoingTask(t, root, taskPath)

			provider := markdown.New(root, markdown.Hooks{})
			service := taskflow.NewService(provider.Flow())
			defer service.Close()

			var upserted tasklifecycle.Task
			finalizer := executionfinalizer.New(service, provider.Load, func(task tasklifecycle.Task) {
				upserted = task
			})

			updated, err := finalizer.Apply(taskflow.ExecutionResult{
				TaskID:      relPath(root, taskPath),
				ExecutionID: "claim-113-" + tc.name,
				Agent:       "cursor",
				Outcome:     tc.outcome,
				Summary:     string(tc.outcome),
				Error:       "detail-" + tc.name,
				Question:    "question-" + tc.name,
				Artifacts:   []string{"a.go"},
				Tests:       []string{"go test"},
			})
			if err != nil {
				t.Fatal(err)
			}
			if updated.Status != tc.wantStatus {
				t.Fatalf("status = %q, want %q", updated.Status, tc.wantStatus)
			}
			if upserted.Status != tc.wantStatus {
				t.Fatalf("upserted status = %q, want %q", upserted.Status, tc.wantStatus)
			}
			if len(updated.Comments) == 0 {
				t.Fatal("expected operator-visible comment")
			}
			text := updated.Comments[len(updated.Comments)-1].Text
			if !strings.Contains(text, "Execution outcome: `"+string(tc.outcome)+"`") {
				t.Fatalf("comment missing outcome: %q", text)
			}
			_ = context.Background()
		})
	}
}

type recordingPublisher struct {
	events []eventbus.Event
	err    error
}

func (p *recordingPublisher) Publish(event eventbus.Event) error {
	if p.err != nil {
		return p.err
	}
	p.events = append(p.events, event)
	return nil
}

func TestFinalizerPublishesTaskEvents(t *testing.T) {
	cases := []struct {
		outcome  taskflow.ExecutionOutcome
		wantType string
		wantText string
	}{
		{taskflow.ExecutionCompleted, "task.needs_review", "is ready for review"},
		{taskflow.ExecutionNeedsInput, "task.needs_attention", "needs a decision before continuing"},
	}
	for _, tc := range cases {
		t.Run(string(tc.outcome), func(t *testing.T) {
			root := t.TempDir()
			taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-publish-"+string(tc.outcome)+".md")
			writeDoingTask(t, root, taskPath)
			provider := markdown.New(root, markdown.Hooks{})
			service := taskflow.NewService(provider.Flow())
			defer service.Close()

			pub := &recordingPublisher{}
			finalizer := executionfinalizer.New(service, provider.Load, nil).WithPublisher(pub)
			locator := relPath(root, taskPath)
			updated, err := finalizer.Apply(taskflow.ExecutionResult{
				TaskID:   locator,
				Outcome:  tc.outcome,
				Summary:  "worker summary",
				Question: "Which API?",
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(pub.events) != 1 {
				t.Fatalf("events = %#v, want one", pub.events)
			}
			event := pub.events[0]
			if event.Channel != eventbus.ChannelTask || event.Type != tc.wantType {
				t.Fatalf("event = %#v", event)
			}
			if event.Fields["task_id"] != locator {
				t.Fatalf("task_id = %q", event.Fields["task_id"])
			}
			if event.Fields["task_ref"] != updated.Ref || updated.Ref == "" {
				t.Fatalf("task_ref = %q, want the task's ref %q", event.Fields["task_ref"], updated.Ref)
			}
			if !strings.Contains(event.Text, tc.wantText) || !strings.Contains(event.Text, locator) {
				t.Fatalf("text = %q", event.Text)
			}
			if tc.outcome == taskflow.ExecutionNeedsInput && !strings.Contains(event.Text, "Which API?") {
				t.Fatalf("text = %q, want the question", event.Text)
			}
			if updated.Status == "" {
				t.Fatal("status was not written")
			}
		})
	}
}

func TestFinalizerPublishesBlockedOutcomes(t *testing.T) {
	for _, outcome := range []taskflow.ExecutionOutcome{
		taskflow.ExecutionFailed,
		taskflow.ExecutionTimedOut,
		taskflow.ExecutionCancelled,
		taskflow.ExecutionOrphaned,
		taskflow.ExecutionBlocked,
	} {
		t.Run(string(outcome), func(t *testing.T) {
			root := t.TempDir()
			taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-blocked-"+string(outcome)+".md")
			writeDoingTask(t, root, taskPath)
			provider := markdown.New(root, markdown.Hooks{})
			service := taskflow.NewService(provider.Flow())
			defer service.Close()

			pub := &recordingPublisher{}
			finalizer := executionfinalizer.New(service, provider.Load, nil).WithPublisher(pub)
			locator := relPath(root, taskPath)
			updated, err := finalizer.Apply(taskflow.ExecutionResult{
				TaskID:  locator,
				Outcome: outcome,
				Summary: "startup failed",
				Error:   "chdir missing",
			})
			if err != nil {
				t.Fatal(err)
			}
			if updated.Status != "blocked" {
				t.Fatalf("status = %q, want blocked", updated.Status)
			}
			if len(pub.events) != 1 {
				t.Fatalf("events = %#v, want one", pub.events)
			}
			event := pub.events[0]
			if event.Channel != eventbus.ChannelTask || event.Type != "task.needs_attention" {
				t.Fatalf("event = %#v", event)
			}
			if !strings.Contains(event.Text, "is blocked") || !strings.Contains(event.Text, "chdir missing") {
				t.Fatalf("text = %q", event.Text)
			}
		})
	}
}

func TestFinalizerDoesNotPublishNeedsRework(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-quiet-needs-rework.md")
	writeDoingTask(t, root, taskPath)
	provider := markdown.New(root, markdown.Hooks{})
	service := taskflow.NewService(provider.Flow())
	defer service.Close()

	pub := &recordingPublisher{}
	finalizer := executionfinalizer.New(service, provider.Load, nil).WithPublisher(pub)
	if _, err := finalizer.Apply(taskflow.ExecutionResult{
		TaskID:  relPath(root, taskPath),
		Outcome: taskflow.ExecutionNeedsRework,
		Summary: "still on the board",
	}); err != nil {
		t.Fatal(err)
	}
	if len(pub.events) != 0 {
		t.Fatalf("events = %#v, want none", pub.events)
	}
}

func TestFinalizerKeepsStatusWhenPublishFails(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-04-publish-error.md")
	writeDoingTask(t, root, taskPath)
	provider := markdown.New(root, markdown.Hooks{})
	service := taskflow.NewService(provider.Flow())
	defer service.Close()

	pub := &recordingPublisher{err: errors.New("bus down")}
	finalizer := executionfinalizer.New(service, provider.Load, nil).WithPublisher(pub)
	updated, err := finalizer.Apply(taskflow.ExecutionResult{
		TaskID:  relPath(root, taskPath),
		Outcome: taskflow.ExecutionCompleted,
		Summary: "shipped",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "needs_review" {
		t.Fatalf("status = %q, want needs_review despite publish error", updated.Status)
	}
}

func writeDoingTask(t *testing.T, root, taskPath string) {
	t.Helper()
	project := filepath.Join(root, "Work", "core-eggs-gd", "PROJECT.md")
	if err := os.MkdirAll(filepath.Dir(taskPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(project, []byte("---\nid: core-eggs-gd\ntitle: Core\nkind: standalone_repository\nstatus: draft\nreview_status: draft\nrepositories:\n  - core.eggs.gd\n---\n\n# Core\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	body := `---
schema_version: 1
id: work-finalizer
ref: CORE-113
title: "Finalizer"
type: maintenance
status: doing
priority: 1
project: core-eggs-gd
repositories:
  - core.eggs.gd
assignee: cursor
created_at: 2026-08-04T01:00:00+03:00
updated_at: 2026-08-04T01:00:00+03:00
---

# Finalizer
`
	if err := os.WriteFile(taskPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func relPath(root, abs string) string {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return abs
	}
	return rel
}
