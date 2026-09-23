package appconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDataRootCreatesTemplateForFreshRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "fresh-data")
	created, err := EnsureDataRoot(root)
	if err != nil {
		t.Fatalf("EnsureDataRoot: %v", err)
	}
	if !created {
		t.Fatal("expected created=true for a fresh root")
	}

	for _, want := range []string{
		"AGENTS.md",
		"README.md",
		"Inbox/README.md",
		"Inbox/items/README.md",
		"Work/README.md",
		"Work/INDEX.md",
		"Fleet/README.md",
		"Archive/README.md",
		"_registry/README.md",
		"_docs/README.md",
	} {
		if _, err := os.Stat(filepath.Join(root, want)); err != nil {
			t.Errorf("expected %s to exist: %v", want, err)
		}
	}
}

func TestEnsureDataRootLeavesExistingRootUntouched(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(marker, []byte("custom content"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	created, err := EnsureDataRoot(root)
	if err != nil {
		t.Fatalf("EnsureDataRoot: %v", err)
	}
	if created {
		t.Fatal("expected created=false for an already-existing root")
	}

	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "custom content" {
		t.Fatalf("existing file was overwritten: %q", data)
	}

	if _, err := os.Stat(filepath.Join(root, "Inbox")); err == nil {
		t.Fatal("expected no template files to be added to an existing root")
	}
}
