package taskprovider

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func writeCard(t *testing.T, root, id, extra string, mtime time.Time) {
	t.Helper()
	path := filepath.Join(root, "Work", id, "PROJECT.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	text := "---\nid: " + id + "\ntitle: " + id + "\nkind: standalone_repository\n" + extra + "---\n\n# " + id + "\n"
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func TestSyncCardsRefreshesOnlyCardsThatChanged(t *testing.T) {
	root := t.TempDir()
	base := time.Now().Add(-time.Hour)
	writeCard(t, root, "acme", "", base)
	b := NewBoard(root)

	first, err := SyncCards(b, root)
	if err != nil || len(first) != 1 || first[0] != "acme" {
		t.Fatalf("first sync = %v, %v", first, err)
	}
	if again, _ := SyncCards(b, root); len(again) != 0 {
		t.Fatalf("an unchanged card was reloaded: %v", again)
	}
	generated := b.Snapshot().GeneratedAt

	// A hand edit (new mtime) is picked up, and a card the scanner creates
	// later appears, without restarting serve.
	writeCard(t, root, "acme", "milestones:\n  - id: v1\n    title: v1\n    state: parked\n", base.Add(time.Minute))
	writeCard(t, root, "newproj", "", base)
	changed, err := SyncCards(b, root)
	if err != nil || len(changed) != 2 {
		t.Fatalf("second sync = %v, %v", changed, err)
	}
	snap := b.Snapshot()
	var found bool
	for _, p := range snap.Projects {
		if p.ID == "acme" && len(p.Milestones) == 1 {
			found = true
		}
	}
	if !found || len(snap.Workspaces) != 2 {
		t.Fatalf("board after sync: workspaces=%d projects=%+v", len(snap.Workspaces), snap.Projects)
	}
	_ = generated
}

func TestSyncCardsRetriesACardThatFailedToLoad(t *testing.T) {
	root := t.TempDir()
	base := time.Now().Add(-time.Hour)
	writeCard(t, root, "acme", "", base)
	b := NewBoard(root)
	if _, err := SyncCards(b, root); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(root, "Work", "acme", "PROJECT.md")
	writeCard(t, root, "acme", "milestones:\n  - id: v1\n    title: v1\n    state: parked\n", base.Add(time.Minute))
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncCards(b, root); err == nil {
		t.Skip("file stayed readable (running as root?)")
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	// Same size and mtime as the failed attempt: it must still be reloaded.
	changed, err := SyncCards(b, root)
	if err != nil || len(changed) != 1 {
		t.Fatalf("retry = %v, %v", changed, err)
	}
}

func TestSyncCardsRemovesRepositoriesDroppedFromTheCard(t *testing.T) {
	root := t.TempDir()
	base := time.Now().Add(-time.Hour)
	group := func(repos string, mtime time.Time) {
		writeCard(t, root, "acme", "repositories:\n"+repos, mtime)
		path := filepath.Join(root, "Work", "acme", "PROJECT.md")
		raw, _ := os.ReadFile(path)
		text := strings.Replace(string(raw), "kind: standalone_repository", "kind: workspace_group", 1)
		_ = os.WriteFile(path, []byte(text), 0o644)
		_ = os.Chtimes(path, mtime, mtime)
	}
	group("  - acme/web\n  - acme/api\n", base)
	b := NewBoard(root)
	if _, err := SyncCards(b, root); err != nil {
		t.Fatal(err)
	}
	ids := func() string {
		var out []string
		for _, p := range b.Snapshot().Projects {
			out = append(out, p.ID)
		}
		sort.Strings(out)
		return strings.Join(out, ",")
	}
	if got := ids(); got != "acme,acme/api,acme/web" {
		t.Fatalf("before = %s", got)
	}

	group("  - acme/web\n", base.Add(time.Minute))
	if _, err := SyncCards(b, root); err != nil {
		t.Fatal(err)
	}
	if got := ids(); got != "acme,acme/web" {
		t.Fatalf("after removing a repository = %s", got)
	}
	for _, p := range b.Snapshot().Projects {
		if p.ID == "acme" && strings.Join(p.Repositories, ",") != "acme/web" {
			t.Fatalf("workspace project kept the old repositories: %v", p.Repositories)
		}
	}
}
