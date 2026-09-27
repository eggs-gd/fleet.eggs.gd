package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/eventbus"
	"github.com/eggs-gd/fleet.eggs.gd/internal/health"
)

func TestWatchScansWhenRepositoriesChange(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, OverlayFileName), []byte("scanRoot: /proj\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var scans int
	scanner := NewScanner(func([]string, string) error {
		scans++
		return nil
	}, func() error { return nil })

	calls := 0
	paths := func([]string) ([]string, error) {
		calls++
		if calls == 1 {
			return []string{"/proj/a"}, nil
		}
		return []string{"/proj/b", "/proj/a"}, nil
	}
	var seen watchSeen
	scanner.watchOnce(root, paths, &seen)
	scanner.watchOnce(root, paths, &seen)
	if scans != 2 {
		t.Fatalf("scans = %d, want 2", scans)
	}

	scanner.watchOnce(root, func([]string) ([]string, error) {
		return []string{"/proj/a", "/proj/b"}, nil
	}, &seen)
	if scans != 2 {
		t.Fatalf("unchanged scans = %d, want 2", scans)
	}
}

func TestWatchRetriesAfterFailedScan(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, OverlayFileName), []byte("scanRoot: /proj\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fail := true
	var scans int
	scanner := NewScanner(func([]string, string) error {
		scans++
		if fail {
			return fmt.Errorf("boom")
		}
		return nil
	}, nil)
	paths := func([]string) ([]string, error) {
		return []string{"/proj/a"}, nil
	}
	var seen watchSeen
	scanner.watchOnce(root, paths, &seen)
	if scans != 1 || seen.ready {
		t.Fatalf("scans = %d ready = %v", scans, seen.ready)
	}
	fail = false
	scanner.watchOnce(root, paths, &seen)
	if scans != 2 || !seen.ready {
		t.Fatalf("scans = %d ready = %v", scans, seen.ready)
	}
	scanner.watchOnce(root, paths, &seen)
	if scans != 2 {
		t.Fatalf("scans = %d, want 2", scans)
	}
}

func TestWatchScansWhenSecondRootChanges(t *testing.T) {
	root := t.TempDir()
	body := "scanRoots:\n  - /one\n  - /two\n"
	if err := os.WriteFile(filepath.Join(root, OverlayFileName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var scans int
	scanner := NewScanner(func([]string, string) error {
		scans++
		return nil
	}, nil)
	current := []string{"/one/a", "/two/b"}
	paths := func([]string) ([]string, error) {
		return append([]string{}, current...), nil
	}
	var seen watchSeen
	scanner.watchOnce(root, paths, &seen)
	scanner.watchOnce(root, paths, &seen)
	if scans != 1 {
		t.Fatalf("scans = %d, want 1", scans)
	}
	current = []string{"/one/a", "/two/c"}
	scanner.watchOnce(root, paths, &seen)
	if scans != 2 {
		t.Fatalf("scans = %d, want 2", scans)
	}
}

func TestWatchSkipsMissingScanRoot(t *testing.T) {
	root := t.TempDir()
	var scans int
	scanner := NewScanner(func([]string, string) error {
		scans++
		return nil
	}, nil)
	var seen watchSeen
	scanner.watchOnce(root, func([]string) ([]string, error) {
		return nil, fmt.Errorf("missing")
	}, &seen)
	if scans != 0 || seen.ready {
		t.Fatalf("scans = %d ready = %v", scans, seen.ready)
	}
}

func TestMaintainPublishesProjectDiscoveredRepoChangeAndMissing(t *testing.T) {
	root := t.TempDir()
	registry := filepath.Join(root, "_registry")
	if err := os.MkdirAll(registry, 0o755); err != nil {
		t.Fatal(err)
	}
	writeRepos := func(repos ...string) {
		items := make([]string, 0, len(repos))
		ids := make([]string, 0, len(repos))
		for _, rel := range repos {
			id := "id-" + rel
			items = append(items, `{"id": "`+id+`", "name": "`+id+`", "relative_path": "`+rel+`", "stack": ["go"]}`)
			ids = append(ids, `"`+id+`"`)
		}
		reposJSON := `{"repositories": [` + strings.Join(items, ",") + `]}`
		groups := `{"groups": [{"kind": "workspace_group", "reason": "same top-level folder", "suggested_project_id": "space", "repository_ids": [` + strings.Join(ids, ",") + `]}]}`
		if err := os.WriteFile(filepath.Join(registry, "repositories.json"), []byte(reposJSON), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(registry, "project-groups.json"), []byte(groups), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var events []eventbus.Event
	scanner := NewScanner(nil, nil)
	scanner.Health = health.New()
	scanner.Publish = func(event eventbus.Event) error {
		events = append(events, event)
		return nil
	}

	// First pass: the project is new. No history to compare against yet, so
	// only "discovered" fires, never "missing".
	writeRepos("Space/api", "Space/web")
	scanner.maintain(root, true)
	if len(events) != 1 || events[0].Type != "project.discovered" || events[0].Fields["project_id"] != "space" {
		t.Fatalf("first pass events = %#v", events)
	}

	// Second pass: same shape, nothing changed. Silence.
	events = nil
	scanner.maintain(root, true)
	if len(events) != 0 {
		t.Fatalf("an unchanged project must stay silent: %#v", events)
	}

	// Third pass: the registry now lists a third repository. The card's
	// stored list no longer matches.
	events = nil
	writeRepos("Space/api", "Space/web", "Space/docs")
	scanner.maintain(root, true)
	if len(events) != 1 || events[0].Type != "project.repos_changed" || events[0].Fields["project_id"] != "space" {
		t.Fatalf("repos-changed pass events = %#v", events)
	}

	// Fourth pass: the registry no longer has this project at all (every
	// repository under it disappeared). The card file is left alone
	// (Ground Rule 5), but the bus hears about it exactly once.
	events = nil
	if err := os.WriteFile(filepath.Join(registry, "repositories.json"), []byte(`{"repositories": []}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(registry, "project-groups.json"), []byte(`{"groups": []}`), 0o644); err != nil {
		t.Fatal(err)
	}
	scanner.maintain(root, true)
	if len(events) != 1 || events[0].Type != "project.missing" || events[0].Fields["project_id"] != "space" {
		t.Fatalf("missing pass events = %#v", events)
	}
	if _, err := os.Stat(filepath.Join(root, "Work", "space", "PROJECT.md")); err != nil {
		t.Fatalf("a missing project's card must not be deleted: %v", err)
	}

	// Fifth pass: still gone. Reported once, not every tick.
	events = nil
	scanner.maintain(root, true)
	if len(events) != 0 {
		t.Fatalf("a project already reported missing must not be reported again: %#v", events)
	}
}
