package projectscan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeRegistry(t *testing.T, root string, repos ...string) (registry, work string) {
	t.Helper()
	registry = filepath.Join(root, "_registry")
	work = filepath.Join(root, "Work")
	if err := os.MkdirAll(registry, 0o755); err != nil {
		t.Fatal(err)
	}
	items := make([]string, 0, len(repos))
	ids := make([]string, 0, len(repos))
	for _, rel := range repos {
		id := filepath.Base(rel)
		items = append(items, `{"id": "`+id+`", "name": "`+id+`", "relative_path": "`+rel+`", "stack": ["go"]}`)
		ids = append(ids, `"`+id+`"`)
	}
	reposJSON := `{"repositories": [` + strings.Join(items, ",") + `]}`
	groups := `{"groups": [{"kind": "workspace_group", "reason": "same top-level folder", "suggested_project_id": "space", "repository_ids": [` + strings.Join(ids, ",") + `]}]}`
	for name, body := range map[string]string{"repositories.json": reposJSON, "project-groups.json": groups} {
		if err := os.WriteFile(filepath.Join(registry, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return registry, work
}

func TestMaintainCardsCreatesAndKeepsHumanFields(t *testing.T) {
	root := t.TempDir()
	registry, work := writeRegistry(t, root, "Space/api", "Space/web")

	report, err := MaintainCards(registry, work)
	if err != nil || len(report.Created) != 1 || report.Created[0] != "space" {
		t.Fatalf("first pass = %+v, %v", report, err)
	}
	card := filepath.Join(work, "space", "PROJECT.md")
	if _, err := os.Stat(filepath.Join(work, "space", "tasks")); err != nil {
		t.Fatalf("tasks dir missing: %v", err)
	}
	if report, _ := MaintainCards(registry, work); len(report.Created)+len(report.Repaired) != 0 {
		t.Fatalf("second pass changed a healthy card: %+v", report)
	}

	raw, _ := os.ReadFile(card)
	edited := strings.Replace(string(raw), "repositories:", "aliases: [billing, \"the space\"]\ndefault_assignee: codex\nrepositories:", 1)
	edited += "\n## Activity Log\n\n- 2026-08-01 alias: billing\n"
	if err := os.WriteFile(card, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	registry, work = writeRegistry(t, root, "Space/api", "Space/web", "Space/docs")
	report, err = MaintainCards(registry, work)
	if err != nil || len(report.Repaired) != 1 {
		t.Fatalf("repair pass = %+v, %v", report, err)
	}
	got, _ := os.ReadFile(card)
	for _, want := range []string{"aliases:\n  - billing\n  - the space\n", "default_assignee: codex", `  - "Space/docs"`, "## Activity Log\n\n- 2026-08-01 alias: billing"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("card lacks %q:\n%s", want, got)
		}
	}
}

func TestMaintainCardsRestoresMissingFrontmatterAndSkipsUnknownCards(t *testing.T) {
	root := t.TempDir()
	registry, work := writeRegistry(t, root, "Space/api")
	if err := os.MkdirAll(filepath.Join(work, "space"), 0o755); err != nil {
		t.Fatal(err)
	}
	card := filepath.Join(work, "space", "PROJECT.md")
	if err := os.WriteFile(card, []byte("# Space\n\nmy own notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	life := filepath.Join(work, "_life", "PROJECT.md")
	if err := os.MkdirAll(filepath.Dir(life), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(life, []byte("hand made, no frontmatter\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := MaintainCards(registry, work)
	if err != nil || len(report.Repaired) != 1 {
		t.Fatalf("pass = %+v, %v", report, err)
	}
	got, _ := os.ReadFile(card)
	if !strings.HasPrefix(string(got), "---\nid: \"space\"") || !strings.Contains(string(got), "my own notes") {
		t.Fatalf("card = %s", got)
	}
	if kept, _ := os.ReadFile(life); string(kept) != "hand made, no frontmatter\n" {
		t.Fatalf("unknown card was touched: %s", kept)
	}
}

func TestMaintainCardsWithoutARegistryIsNotAnError(t *testing.T) {
	root := t.TempDir()
	if _, err := MaintainCards(filepath.Join(root, "_registry"), filepath.Join(root, "Work")); err != nil {
		t.Fatal(err)
	}
}
