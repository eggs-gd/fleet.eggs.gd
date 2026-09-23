package appconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingReturnsZeroConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataRoot != "" {
		t.Fatalf("expected empty DataRoot, got %q", cfg.DataRoot)
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := Save(Config{DataRoot: "/tmp/example-data"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataRoot != "/tmp/example-data" {
		t.Fatalf("DataRoot = %q, want /tmp/example-data", cfg.DataRoot)
	}
}

func TestExpandHomeExpandsTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	got, err := ExpandHome("~/fleet_data")
	if err != nil {
		t.Fatalf("ExpandHome: %v", err)
	}
	want := filepath.Join(home, "fleet_data")
	if got != want {
		t.Fatalf("ExpandHome = %q, want %q", got, want)
	}
}

func TestExpandHomeRejectsEmpty(t *testing.T) {
	if _, err := ExpandHome("   "); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestExpandHomeAbsolutizesPlainPath(t *testing.T) {
	got, err := ExpandHome("relative/dir")
	if err != nil {
		t.Fatalf("ExpandHome: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute path, got %q", got)
	}
}

func TestPointerPathUnderHomeFleet(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err := PointerPath()
	if err != nil {
		t.Fatalf("PointerPath: %v", err)
	}
	want := filepath.Join(home, ".fleet", "app.json")
	if path != want {
		t.Fatalf("PointerPath = %q, want %q", path, want)
	}
}

func TestSaveIsAtomicNoLeftoverTempFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := Save(Config{DataRoot: "/tmp/x"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(home, ".fleet"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "app.json" {
		t.Fatalf("unexpected entries in ~/.fleet: %v", entries)
	}
}
