package markdown

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFsWalkerMarksFirstScanAsInitialAndLaterChangesAsModified(t *testing.T) {
	root := t.TempDir()
	taskPath := filepath.Join(root, "Work", "core-eggs-gd", "tasks", "2026-07-31-a.md")
	if err := os.MkdirAll(filepath.Dir(taskPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(taskPath, []byte("---\nref: CORE-2\nstatus: todo\n---\n\n# Second\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	walker := &FsWalker{
		root: root,
		seen: map[string]fileSignature{},
	}
	ch := make(chan FileEvent, 4)
	walker.scan(ch, context.Background())

	initial := <-ch
	if initial.Kind != FileEventInitial {
		t.Fatalf("initial kind = %q, want %q", initial.Kind, FileEventInitial)
	}

	info, err := os.Stat(taskPath)
	if err != nil {
		t.Fatal(err)
	}
	walker.seen[taskPath] = fileSignature{
		modTime: info.ModTime().Add(-1),
		size:    info.Size(),
	}
	walker.scan(ch, context.Background())

	modified := <-ch
	if modified.Kind != FileEventModified {
		t.Fatalf("modified kind = %q, want %q", modified.Kind, FileEventModified)
	}
}
