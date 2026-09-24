package server

import (
	"encoding/json"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider"
	tpopen "github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider/open"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreateFromRequestWritesTaskIncrementsCounterAndRefreshesIndex is a
// integration integration test: it verifies that creating a task through
// a Store-backed markdown.Provider (as the daemon does) refreshes the dashboard
// projection's derived Work/INDEX.md, not just the lifecycle-level task
// fields covered by tasklifecycle's own tests.
func TestCreateFromRequestWritesTaskIncrementsCounterAndRefreshesIndex(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "Work", "core-eggs-gd")
	writeTestFile(t, filepath.Join(projectDir, "PROJECT.md"), "# Core\n")
	writeTestFile(t, filepath.Join(root, "_registry", "counters.json"), `{
  "work_ref_prefix": "CORE",
  "next_work_ref": 42,
  "inbox_ref_prefix": "INBOX",
  "next_inbox_ref": 1,
  "life_ref_prefix": "LIFE",
  "next_life_ref": 1,
  "notes": []
}
`)

	store := tpopen.NewMarkdown(root, root, taskprovider.NewBoard(root))
	priority := 4
	task, err := store.CreateFromRequest(tasklifecycle.TaskCreateRequest{
		Title:      "Dashboard create flow",
		Request:    "Add a create-task action to the backoffice.",
		Project:    "core-eggs-gd",
		Repository: "core.eggs.gd",
		Status:     "backlog",
		Type:       "feature",
		Assignee:   "codex",
		Priority:   &priority,
	})
	if err != nil {
		t.Fatal(err)
	}

	if task.Ref != "CORE-42" {
		t.Fatalf("ref = %q, want CORE-42", task.Ref)
	}
	if task.Status != "backlog" {
		t.Fatalf("status = %q, want backlog", task.Status)
	}
	if task.Type != "feature" {
		t.Fatalf("type = %q, want feature", task.Type)
	}
	if task.Assignee != "codex" {
		t.Fatalf("assignee = %q, want codex", task.Assignee)
	}
	if task.Priority != 4 {
		t.Fatalf("priority = %d, want 4", task.Priority)
	}
	if task.SchemaVersion != 1 {
		t.Fatalf("schema_version = %d, want 1", task.SchemaVersion)
	}
	if len(task.Repositories) != 1 || task.Repositories[0] != "core.eggs.gd" {
		t.Fatalf("repositories = %#v", task.Repositories)
	}
	if !strings.Contains(task.RelativePath, "Work/core-eggs-gd/tasks/") {
		t.Fatalf("relative path = %q", task.RelativePath)
	}
	if !strings.Contains(task.Body, "## Request") {
		t.Fatalf("body missing Request section:\n%s", task.Body)
	}
	if !strings.Contains(task.Body, "Add a create-task action to the backoffice.") {
		t.Fatalf("body missing request text:\n%s", task.Body)
	}
	if !strings.Contains(task.Body, "Created from backoffice dashboard") {
		t.Fatalf("body missing activity log:\n%s", task.Body)
	}

	data, err := os.ReadFile(filepath.Join(root, "_registry", "counters.json"))
	if err != nil {
		t.Fatal(err)
	}
	var counters struct {
		NextWorkRef int `json:"next_work_ref"`
	}
	if err := json.Unmarshal(data, &counters); err != nil {
		t.Fatal(err)
	}
	if counters.NextWorkRef != 43 {
		t.Fatalf("next_work_ref = %d, want 43", counters.NextWorkRef)
	}

	index, err := os.ReadFile(filepath.Join(root, "Work", "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), "`CORE-42`") {
		t.Fatalf("index missing created task\n%s", index)
	}

	raw, err := os.ReadFile(task.Path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if !strings.Contains(content, "auto_commit: false") {
		t.Fatalf("created task missing launch defaults:\n%s", content)
	}
	if !strings.Contains(content, "schema_version: 1") {
		t.Fatalf("created task missing schema_version:\n%s", content)
	}
}
