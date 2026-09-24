package appconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMoveDataRootRelocatesContent(t *testing.T) {
	base := t.TempDir()
	oldRoot := filepath.Join(base, "old")
	newRoot := filepath.Join(base, "nested", "new")
	if err := os.MkdirAll(filepath.Join(oldRoot, "Work"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(oldRoot, "Work", "INDEX.md"), []byte("index"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := MoveDataRoot(oldRoot, newRoot); err != nil {
		t.Fatalf("MoveDataRoot: %v", err)
	}

	if _, err := os.Stat(oldRoot); !os.IsNotExist(err) {
		t.Fatalf("expected old root to be gone, stat err = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(newRoot, "Work", "INDEX.md"))
	if err != nil {
		t.Fatalf("ReadFile at new root: %v", err)
	}
	if string(data) != "index" {
		t.Fatalf("content = %q, want %q", data, "index")
	}
}

func TestMoveDataRootNoopWhenSamePath(t *testing.T) {
	root := t.TempDir()
	if err := MoveDataRoot(root, root); err != nil {
		t.Fatalf("MoveDataRoot: %v", err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("root should still exist: %v", err)
	}
}

func TestMoveDataRootNoopWhenOldMissing(t *testing.T) {
	base := t.TempDir()
	oldRoot := filepath.Join(base, "does-not-exist")
	newRoot := filepath.Join(base, "new")
	if err := MoveDataRoot(oldRoot, newRoot); err != nil {
		t.Fatalf("MoveDataRoot: %v", err)
	}
	if _, err := os.Stat(newRoot); !os.IsNotExist(err) {
		t.Fatal("expected newRoot to not be created when oldRoot never existed")
	}
}

func TestMoveDataRootRefusesNonEmptyDestination(t *testing.T) {
	base := t.TempDir()
	oldRoot := filepath.Join(base, "old")
	newRoot := filepath.Join(base, "new")
	if err := os.MkdirAll(oldRoot, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.MkdirAll(newRoot, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(newRoot, "existing.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	err := MoveDataRoot(oldRoot, newRoot)
	if err == nil {
		t.Fatal("expected error for non-empty destination")
	}
	if _, statErr := os.Stat(oldRoot); statErr != nil {
		t.Fatalf("old root should be untouched after refusal: %v", statErr)
	}
}

func TestMoveDataRootAllowsEmptyDestinationDir(t *testing.T) {
	base := t.TempDir()
	oldRoot := filepath.Join(base, "old")
	newRoot := filepath.Join(base, "new")
	if err := os.MkdirAll(oldRoot, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(oldRoot, "marker.txt"), []byte("m"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.MkdirAll(newRoot, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := MoveDataRoot(oldRoot, newRoot); err != nil {
		t.Fatalf("MoveDataRoot: %v", err)
	}
	if _, err := os.Stat(filepath.Join(newRoot, "marker.txt")); err != nil {
		t.Fatalf("expected marker.txt at new root: %v", err)
	}
}
