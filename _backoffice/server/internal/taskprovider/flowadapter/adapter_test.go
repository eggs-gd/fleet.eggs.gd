package flowadapter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/eggs-gd/core.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/core.eggs.gd/internal/taskprovider/markdown"
)

func TestServiceOverMarkdownProvider(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Work", "core-eggs-gd", "PROJECT.md"), "# Core\n")
	writeFile(t, filepath.Join(root, "_registry", "counters.json"), `{
  "work_ref_prefix": "CORE",
  "next_work_ref": 101,
  "inbox_ref_prefix": "INBOX",
  "next_inbox_ref": 1,
  "life_ref_prefix": "LIFE",
  "next_life_ref": 1,
  "notes": []
}
`)

	provider := markdown.New(root, markdown.Hooks{})
	svc := taskflow.NewService(provider.Flow())
	defer svc.Close()

	created, err := svc.Create(context.Background(), taskflow.CreateTask{
		Title:       "Serialized service smoke",
		Description: "Prove Markdown workflows still work through TaskService.",
		Project:     "core-eggs-gd",
		Repository:  "core.eggs.gd",
		Status:      taskflow.StatusTodo,
		Type:        "maintenance",
		Assignee:    "cursor",
		Priority:    1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Ref == "" || created.Locator == "" {
		t.Fatalf("created missing ref/locator: %+v", created)
	}
	if created.Status != taskflow.StatusTodo {
		t.Fatalf("status = %s, want todo", created.Status)
	}

	got, err := svc.Get(context.Background(), created.Locator)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Serialized service smoke" {
		t.Fatalf("title = %q", got.Title)
	}

	if err := svc.Transition(context.Background(), created.Locator, taskflow.StatusDoing, taskflow.TransitionMeta{
		Actor:   "launcher",
		Comment: "claim for CORE-102 smoke",
	}); err != nil {
		t.Fatal(err)
	}

	err = svc.Transition(context.Background(), created.Locator, taskflow.StatusArchived, taskflow.TransitionMeta{})
	if !errors.Is(err, taskflow.ErrInvalidTransition) {
		t.Fatalf("doing->archived err = %v, want ErrInvalidTransition", err)
	}

	if err := svc.ReportExecution(context.Background(), taskflow.ExecutionResult{
		TaskID:    created.Locator,
		Outcome:   taskflow.ExecutionCompleted,
		Summary:   "Markdown-backed TaskService path works.",
		Artifacts: []string{"internal/taskprovider/markdown/"},
		Tests:     []string{"go test ./internal/taskprovider/markdown ./internal/taskflow"},
	}); err != nil {
		t.Fatal(err)
	}

	final, err := svc.Get(context.Background(), created.Locator)
	if err != nil {
		t.Fatal(err)
	}
	if final.Status != taskflow.StatusNeedsReview {
		t.Fatalf("status = %s, want needs_review", final.Status)
	}
	if len(final.Comments) < 2 {
		t.Fatalf("comments = %d, want at least claim + execution", len(final.Comments))
	}

	listed, err := svc.List(context.Background(), taskflow.TaskFilter{
		Project: "core-eggs-gd",
		Status:  taskflow.StatusNeedsReview,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("List len = %d, want 1", len(listed))
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
