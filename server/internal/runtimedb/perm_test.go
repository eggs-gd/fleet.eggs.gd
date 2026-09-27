package runtimedb

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenKeepsTheDatabaseOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes are not meaningful on Windows")
	}
	root := filepath.Join(t.TempDir(), "runtime")
	if _, err := Open(root); err != nil {
		t.Fatal(err)
	}
	assertMode(t, root, 0o700)
	assertMode(t, Path(root), 0o600)
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(Path(root) + suffix); err == nil {
			assertMode(t, Path(root)+suffix, 0o600)
		}
	}
}

func TestOpenTightensADatabaseAnOlderVersionLeftReadable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes are not meaningful on Windows")
	}
	root := t.TempDir()
	if err := os.WriteFile(Path(root), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(root); err != nil {
		t.Fatal(err)
	}
	assertMode(t, Path(root), 0o600)
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("%s mode = %o, want %o", path, got, want)
	}
}
