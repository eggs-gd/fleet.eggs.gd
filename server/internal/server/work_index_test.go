package server

import (
	"github.com/eggs-gd/fleet.eggs.gd/internal/board"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider/markdown"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimePatchTaskRebuildsWorkIndex(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Work", "core-eggs-gd", "PROJECT.md"), `---
id: core-eggs-gd
title: Core
kind: standalone_repository
status: draft
review_status: draft
repositories:
  - core.eggs.gd
---

# Core
`)
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-a.md")
	writeTestFile(t, taskPath, testTaskMarkdown("CORE-2", "Second", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root})
	_, err := app.PatchTask(tasklifecycle.TaskPatch{
		Path:   "Work/core-eggs-gd/tasks/2026-07-31-a.md",
		Status: "doing",
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(root, "Work", "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	index := string(data)
	for _, expected := range []string{
		"| `todo` | 0 |",
		"| `doing` | 1 |",
		"| 5 | `CORE-2` | [Second](core-eggs-gd/tasks/2026-07-31-a.md)",
	} {
		if !strings.Contains(index, expected) {
			t.Fatalf("index missing %q\n%s", expected, index)
		}
	}
}

func TestRuntimeBootstrapLoadsInitialStateBeforeWatcher(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Work", "core-eggs-gd", "PROJECT.md"), `---
id: core-eggs-gd
title: Core
kind: standalone_repository
status: draft
review_status: draft
repositories:
  - core.eggs.gd
---

# Core
`)
	writeTestFile(t, filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-a.md"), testTaskMarkdown("CORE-2", "Second", "todo"))

	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}

	state := app.State()
	if len(state.Tasks) != 1 {
		t.Fatalf("tasks len = %d, want 1", len(state.Tasks))
	}
	if len(state.Workspaces) != 1 {
		t.Fatalf("workspaces len = %d, want 1", len(state.Workspaces))
	}
	if len(state.Projects) != 1 {
		t.Fatalf("projects len = %d, want 1", len(state.Projects))
	}
	if state.Tasks[0].ProjectID != "core-eggs-gd" {
		t.Fatalf("task project id = %q, want core-eggs-gd", state.Tasks[0].ProjectID)
	}
}

func TestWorkspaceGroupCreatesRepositoryProjects(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Work", "eggs-gd-prod", "PROJECT.md"), `---
id: eggs-gd-prod
title: eGGs.gd.prod
kind: workspace_group
status: draft
review_status: draft
repositories:
  - eGGs.gd.prod/career-wizard
  - eGGs.gd.prod/i-ching-site
---

# eGGs.gd.prod
`)
	writeTestFile(t, filepath.Join(root, "Work", "eggs-gd-prod", "tasks", "2026-08-01-career.md"), testTaskForProjectMarkdown("CORE-50", "Career", "todo", "eggs-gd-prod", "eGGs.gd.prod/career-wizard"))

	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}

	state := app.State()
	projectIDs := map[string]bool{}
	for _, project := range state.Projects {
		projectIDs[project.ID] = true
	}
	for _, expected := range []string{"eggs-gd-prod", "eggs-gd-prod/career-wizard", "eggs-gd-prod/i-ching-site"} {
		if !projectIDs[expected] {
			t.Fatalf("missing project %q in %#v", expected, state.Projects)
		}
	}
	if state.Tasks[0].ProjectID != "eggs-gd-prod/career-wizard" {
		t.Fatalf("task project id = %q, want eggs-gd-prod/career-wizard", state.Tasks[0].ProjectID)
	}
}

func TestRuntimeBootstrapExposesRepositoryTechnologyProfiles(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "Work", "audiophile", "PROJECT.md"), `---
id: audiophile
title: Audiophile
kind: workspace_group
status: draft
review_status: draft
repositories:
  - Audiophile/jivemax
  - Audiophile/lmsense
---

# Audiophile
`)
	writeTestFile(t, filepath.Join(root, "_registry", "repositories.json"), `{
  "repositories": [
    {
      "id": "jivemax",
      "name": "jivemax",
      "relative_path": "Audiophile/jivemax",
      "effective": {"languages": [], "frameworks": [], "runtimes": [], "tooling": []},
      "detected": {
        "languages": ["csharp", "lua"],
        "runtimes": ["dotnet"],
        "evidence": [
          {"kind": "marker", "path": "jivelite.sln", "technologies": ["csharp", "dotnet"], "detail": "solution file"}
        ]
      }
    },
    {
      "id": "lmsense",
      "name": "lmsense",
      "relative_path": "Audiophile/lmsense",
      "stack": ["typescript", "svelte"]
    }
  ],
  "effective_repositories": [
    {
      "id": "jivemax",
      "name": "jivemax",
      "relative_path": "Audiophile/jivemax",
      "effective": {"languages": ["lua"], "frameworks": [], "runtimes": [], "tooling": []}
    }
  ]
}`)
	writeTestFile(t, filepath.Join(root, "_registry", "project-groups.json"), `{"groups": []}`)

	app := Compose(ComposeConfig{CoreRoot: root})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}

	state := app.State()
	if state.Registry.RepositoriesCount != 2 {
		t.Fatalf("registry repositories count = %d, want 2", state.Registry.RepositoriesCount)
	}
	if len(state.Registry.Repositories) != 2 {
		t.Fatalf("registry repositories exposed = %d, want 2", len(state.Registry.Repositories))
	}
	project := findProjectByID(state.Projects, "audiophile/jivemax")
	if project == nil {
		t.Fatalf("missing jivemax project in %#v", state.Projects)
	}
	assertStringSet(t, project.Technology.EffectiveTags, []string{"lua"})
	assertStringSet(t, project.Technology.DetectedTags, []string{"csharp", "dotnet", "lua"})
	if len(project.Technology.Repositories) != 1 {
		t.Fatalf("project technology repositories = %#v, want one", project.Technology.Repositories)
	}
	if len(project.Technology.Repositories[0].Evidence) != 1 {
		t.Fatalf("repository evidence = %#v, want one item", project.Technology.Repositories[0].Evidence)
	}

	workspace := findWorkspaceByID(state.Workspaces, "audiophile")
	if workspace == nil {
		t.Fatalf("missing audiophile workspace in %#v", state.Workspaces)
	}
	assertStringSet(t, workspace.Technology.EffectiveTags, []string{"lua", "svelte", "typescript"})
	assertStringSet(t, workspace.Technology.DetectedTags, []string{"csharp", "dotnet", "lua", "svelte", "typescript"})
}

func TestLoadTaskFileReadsSchemaVersionAndDefaultsLegacyCards(t *testing.T) {
	root := t.TempDir()
	versionedPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-01-versioned.md")
	writeTestFile(t, versionedPath, testTaskMarkdown("CORE-41", "Versioned", "todo"))

	versioned, err := markdown.LoadTaskFile(root, versionedPath)
	if err != nil {
		t.Fatal(err)
	}
	if versioned.SchemaVersion != 1 {
		t.Fatalf("schema version = %d, want 1", versioned.SchemaVersion)
	}

	legacyPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-08-01-legacy.md")
	writeTestFile(t, legacyPath, strings.Replace(testTaskMarkdown("CORE-42", "Legacy", "todo"), "schema_version: 1\n", "", 1))

	legacy, err := markdown.LoadTaskFile(root, legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	if legacy.SchemaVersion != 0 {
		t.Fatalf("legacy schema version = %d, want 0", legacy.SchemaVersion)
	}
}

func TestTasksSortByPriorityThenRef(t *testing.T) {
	tasks := []tasklifecycle.Task{
		{Ref: "CORE-10", Priority: 3},
		{Ref: "CORE-2", Priority: 3},
		{Ref: "CORE-9", Priority: 1},
	}

	if !tasklifecycle.TaskPickupLess(tasks[2], tasks[1]) {
		t.Fatal("priority 1 should sort before priority 3")
	}
	if !tasklifecycle.TaskPickupLess(tasks[1], tasks[0]) {
		t.Fatal("CORE-2 should sort before CORE-10 inside the same priority")
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func findProjectByID(projects []board.Project, id string) *board.Project {
	for i := range projects {
		if projects[i].ID == id {
			return &projects[i]
		}
	}
	return nil
}

func findWorkspaceByID(workspaces []board.Workspace, id string) *board.Workspace {
	for i := range workspaces {
		if workspaces[i].ID == id {
			return &workspaces[i]
		}
	}
	return nil
}

func assertStringSet(t *testing.T, got []string, want []string) {
	t.Helper()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("strings = %#v, want %#v", got, want)
	}
}

func testTaskMarkdown(ref string, title string, status string) string {
	return testTaskForProjectMarkdown(ref, title, status, "core-eggs-gd", "core.eggs.gd")
}

func testTaskForProjectMarkdown(ref string, title string, status string, project string, repository string) string {
	return `---
schema_version: 1
id: work-test
ref: ` + ref + `
title: "` + title + `"
type: feature
status: ` + status + `
project: ` + project + `
repositories:
  - ` + repository + `
assignee: codex
created_at: 2026-07-31T10:00:00+03:00
updated_at: 2026-07-31T10:00:00+03:00
launch:
---

# ` + title + `
`
}
