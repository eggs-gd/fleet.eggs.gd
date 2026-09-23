package appconfig

import (
	"path/filepath"
	"testing"
)

func TestResolveDataRootPrefersFlag(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	root, source, err := ResolveDataRoot(dir)
	if err != nil {
		t.Fatalf("ResolveDataRoot: %v", err)
	}
	if root != dir {
		t.Fatalf("root = %q, want %q", root, dir)
	}
	if source != SourceFlag {
		t.Fatalf("source = %q, want %q", source, SourceFlag)
	}
}

func TestResolveDataRootUsesSavedConfigOverDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configured := filepath.Join(home, "custom-data")
	if err := Save(Config{DataRoot: configured}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	root, source, err := ResolveDataRoot("")
	if err != nil {
		t.Fatalf("ResolveDataRoot: %v", err)
	}
	if root != configured {
		t.Fatalf("root = %q, want %q", root, configured)
	}
	if source != SourceConfig {
		t.Fatalf("source = %q, want %q", source, SourceConfig)
	}
}

func TestResolveDataRootFallsBackToDefaultAndPersistsIt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root, source, err := ResolveDataRoot("")
	if err != nil {
		t.Fatalf("ResolveDataRoot: %v", err)
	}
	want := filepath.Join(home, ".fleet", "workspace")
	if root != want {
		t.Fatalf("root = %q, want %q", root, want)
	}
	if source != SourceDefault {
		t.Fatalf("source = %q, want %q", source, SourceDefault)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataRoot != want {
		t.Fatalf("persisted DataRoot = %q, want %q", cfg.DataRoot, want)
	}

	// Second resolution should now read the persisted config, not
	// recompute+resave the default.
	root2, source2, err := ResolveDataRoot("")
	if err != nil {
		t.Fatalf("ResolveDataRoot (2nd): %v", err)
	}
	if root2 != want || source2 != SourceConfig {
		t.Fatalf("2nd resolution = (%q, %q), want (%q, %q)", root2, source2, want, SourceConfig)
	}
}
