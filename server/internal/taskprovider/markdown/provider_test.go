package markdown

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/taskflow"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider"
)

func TestTaskFlowProviderCreateUpdateCommentTransition(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Work", "core-eggs-gd", "PROJECT.md"), "---\nid: core-eggs-gd\ntag: \"CORE\"\n---\n\n# Core\n")
	writeTestFile(t, filepath.Join(root, "_registry", "counters.json"), `{
  "work_ref_prefix": "CORE",
  "next_work_ref": 102,
  "inbox_ref_prefix": "INBOX",
  "next_inbox_ref": 1,
  "life_ref_prefix": "LIFE",
  "next_life_ref": 1,
  "notes": []
}
`)

	provider := New(root, Hooks{})
	flow := provider.Flow()
	svc := taskprovider.NewService(provider)
	defer svc.Close()

	created, err := svc.Create(context.Background(), taskflow.CreateTask{
		Title:       "Markdown provider contract",
		Description: "Prove create/update/comment/transition through TaskProvider.",
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
	if !strings.HasPrefix(created.Ref, "CORE-") || created.Locator == "" {
		t.Fatalf("created = %+v", created)
	}
	if _, err := os.Stat(filepath.Join(root, created.Locator)); err != nil {
		t.Fatalf("task card missing on disk: %v", err)
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
	reloaded, err := provider.Reload(created.Locator)
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
		t.Fatalf("List len = %d, want 1", len(listed))
	}
}

func TestCreateFromRequestDefaultsAndValidates(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Work", "core-eggs-gd", "PROJECT.md"), "---\nid: core-eggs-gd\ntag: \"CORE\"\n---\n\n# Core\n")
	writeTestFile(t, filepath.Join(root, "_registry", "counters.json"), `{
  "work_ref_prefix": "CORE",
  "next_work_ref": 7,
  "inbox_ref_prefix": "INBOX",
  "next_inbox_ref": 1,
  "life_ref_prefix": "LIFE",
  "next_life_ref": 1,
  "notes": []
}
`)
	provider := New(root, Hooks{})

	task, err := provider.CreateFromRequest(tasklifecycle.TaskCreateRequest{
		Title:   "Default backlog task",
		Request: "Keep it in backlog until prioritized.",
		Project: "core-eggs-gd",
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != "backlog" || task.Assignee != "unassigned" || task.Priority != 5 {
		t.Fatalf("task = %+v", task)
	}

	_, err = provider.CreateFromRequest(tasklifecycle.TaskCreateRequest{
		Title:   "Missing project",
		Request: "Should fail",
		Project: "does-not-exist",
	})
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("err = %v", err)
	}
}

func TestSlugifyAndUniqueFileBase(t *testing.T) {
	if got := tasklifecycle.SlugifyTitle("Create new Core tasks from the dashboard!"); got != "create-new-core-tasks-from-the-dashboard" {
		t.Fatalf("slug = %q", got)
	}
	if got := workProjectDirID("eggs-gd-prod/career-wizard"); got != "eggs-gd-prod" {
		t.Fatalf("workProjectDirID = %q", got)
	}

	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-02-create-task.md"), "x")
	base, err := uniqueTaskFileBase(root, "core-eggs-gd", "2026-08-02", "create-task")
	if err != nil {
		t.Fatal(err)
	}
	if base != "2026-08-02-create-task-2" {
		t.Fatalf("base = %q", base)
	}
}

func TestCreatedCardHoldsTheStructuredParts(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Work", "acme", "PROJECT.md"), "---\nid: acme\ntag: \"ACME\"\n---\n\n# Acme\n")
	writeTestFile(t, filepath.Join(root, "_registry", "counters.json"), `{"next":{"ACME":1,"INBOX":8}}`)

	provider := New(root, Hooks{})
	svc := taskprovider.NewService(provider)
	defer svc.Close()

	created, err := svc.Create(context.Background(), taskflow.CreateTask{
		Title:       "Fix the invoice total",
		Description: "The total ignores the discount.",
		Acceptance:  "- [ ] Totals match the ledger.",
		Context:     "Finance signs off on the ledger, so do not round per line. Decision: round once at the end.",
		SourceInbox: "INBOX-7",
		Project:     "acme",
		Status:      taskflow.StatusBacklog,
		Type:        "bug",
		Assignee:    "codex",
		Priority:    2,
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, created.Locator))
	if err != nil {
		t.Fatal(err)
	}
	card := string(raw)
	for _, want := range []string{
		"source_inbox: INBOX-7\n",
		"## Request\n\nThe total ignores the discount.",
		"## Acceptance Criteria\n\n- [ ] Totals match the ledger.",
		"## Context\n\nFinance signs off on the ledger",
	} {
		if !strings.Contains(card, want) {
			t.Errorf("card lacks %q:\n%s", want, card)
		}
	}
	if strings.Contains(card, "Describe the expected outcome") {
		t.Errorf("a placeholder was written although criteria were given:\n%s", card)
	}

	bare, err := svc.Create(context.Background(), taskflow.CreateTask{
		Title: "No context", Description: "Just do it.", Project: "acme", Status: taskflow.StatusBacklog, Type: "feature", Assignee: "codex", Priority: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(filepath.Join(root, bare.Locator))
	if strings.Contains(string(raw), "## Context") {
		t.Errorf("an empty context must not leave a heading:\n%s", raw)
	}
	if !strings.Contains(string(raw), "- [ ] Describe the expected outcome.") {
		t.Errorf("a card without criteria keeps its placeholder:\n%s", raw)
	}
}
