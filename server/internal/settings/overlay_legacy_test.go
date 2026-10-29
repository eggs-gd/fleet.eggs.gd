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
