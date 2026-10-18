package taskprovider

import (
	"os"
	"path/filepath"
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
