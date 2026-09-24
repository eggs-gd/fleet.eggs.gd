package appconfig

import (
	"bytes"
	"io/fs"
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
		".mcp.json",
		"Inbox/README.md",
		"Inbox/items/README.md",
		"Work/README.md",
		"Work/INDEX.md",
		"Fleet/README.md",
		"Fleet/LAUNCH_POLICY.md",
		"Fleet/ROUTING.md",
		"Fleet/owner.md",
		"Fleet/claude.md",
		"Fleet/codex.md",
		"Fleet/cursor.md",
		"Fleet/gemini.md",
		"Archive/README.md",
		"_registry/README.md",
		"_docs/README.md",
		"_docs/TASK_LIFECYCLE.md",
		"_docs/MANAGER.md",
		"_docs/OPERATING_MODEL.md",
		"_docs/PERSONAL_SPACE.md",
		"_docs/templates/WORK_ITEM.md",
	} {
		if _, err := os.Stat(filepath.Join(root, want)); err != nil {
			t.Errorf("expected %s to exist: %v", want, err)
		}
	}
}

func TestEnsureDataRootTemplateHasNoOperatorOrAppLeakage(t *testing.T) {
	root := filepath.Join(t.TempDir(), "fresh-data")
	if _, err := EnsureDataRoot(root); err != nil {
		t.Fatalf("EnsureDataRoot: %v", err)
	}

	banned := [][]byte{[]byte("Alex"), []byte("core.eggs.gd"), []byte("fleet.eggs.gd"), []byte("_backoffice")}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, token := range banned {
			if bytes.Contains(data, token) {
				t.Errorf("%s contains %q; the bootstrap template must stay free of this operator's identity and App-repo internals", path, token)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir: %v", err)
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
