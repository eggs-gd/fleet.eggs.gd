package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOverlayFromALegacyFileMovesToTheNewNameOnSave(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, LegacyOverlayFileName)
	if err := os.WriteFile(legacy, []byte("scanRoots:\n  - /code\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	overlay, err := LoadOverlay(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(overlay.ScanRoots, []string{"/code"}) {
		t.Fatalf("scan roots = %v, want the legacy file's", overlay.ScanRoots)
	}

	overlay.ScanRoots = append(overlay.ScanRoots, "/more")
	if err := SaveOverlay(root, overlay); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(OverlayPath(root)); err != nil {
		t.Fatalf("the new overlay file was not written: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("the legacy file should be gone after Save, stat err = %v", err)
	}
	again, err := LoadOverlay(root)
	if err != nil || !reflect.DeepEqual(again.ScanRoots, []string{"/code", "/more"}) {
		t.Fatalf("reload = %v, %v", again.ScanRoots, err)
	}
}

func TestSaveOverlayKeepsAnExistingDataRootFromCommittingTheNewFile(t *testing.T) {
	root := t.TempDir()
	gitignore := filepath.Join(root, ".gitignore")
	// A Data root from before the rename ignores only the old name, and has no final newline.
	if err := os.WriteFile(gitignore, []byte(".DS_Store\n/.env\n/core.local.yaml"), 0o644); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := SaveOverlay(root, Overlay{ScanRoots: []string{"/code"}}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatal(err)
	}
	want := ".DS_Store\n/.env\n/core.local.yaml\n/fleet.local.yaml\n"
	if string(got) != want {
		t.Errorf(".gitignore = %q, want %q (new name added once, old lines kept)", got, want)
	}
}

func TestSaveOverlayDoesNotInventAGitignore(t *testing.T) {
	root := t.TempDir()
	if err := SaveOverlay(root, Overlay{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".gitignore")); !os.IsNotExist(err) {
		t.Errorf("a Data root without a .gitignore should stay that way, stat err = %v", err)
	}
}
