package settings

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDetectMCPGoplsFallsBackToDefaultGOBINWhenNotOnPATH(t *testing.T) {
	// Regression test: the daemon's PATH is frequently narrower than an
	// interactive shell's (same class of problem as resolveClaudeBinary),
	// so `gopls` can be missing from `exec.LookPath` even though it is
	// installed at the well-known default GOBIN, ~/go/bin.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "") // simulate a daemon PATH that does not include ~/go/bin

	goplsPath := filepath.Join(home, "go", "bin", "gopls")
	if err := os.MkdirAll(filepath.Dir(goplsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(goplsPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	detected, expected, errText := detectMCP(mcpEntry{Command: "gopls", Args: []string{"mcp"}})
	if !detected {
		t.Fatalf("expected gopls to be detected via ~/go/bin fallback, got detected=%v expected=%q err=%q", detected, expected, errText)
	}
	if expected != goplsPath {
		t.Fatalf("expected = %q, want %q", expected, goplsPath)
	}
	if errText != "" {
		t.Fatalf("errText = %q, want empty", errText)
	}
}

func TestDetectMCPGoplsNotDetectedWhenMissingEverywhere(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "")

	detected, _, errText := detectMCP(mcpEntry{Command: "gopls", Args: []string{"mcp"}})
	if detected {
		t.Fatal("expected gopls to be reported as not detected")
	}
	if errText == "" {
		t.Fatal("expected a non-empty error explaining gopls was not found")
	}
}

func TestDetectMCPGoplsPrefersPATHOverFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PATH lookup semantics differ on windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)

	pathDir := t.TempDir()
	pathGopls := filepath.Join(pathDir, "gopls")
	if err := os.WriteFile(pathGopls, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", pathDir)

	detected, expected, _ := detectMCP(mcpEntry{Command: "gopls", Args: []string{"mcp"}})
	if !detected {
		t.Fatal("expected gopls to be detected via PATH")
	}
	if expected != pathGopls {
		t.Fatalf("expected = %q, want the PATH-resolved binary %q (PATH should win over the ~/go/bin fallback)", expected, pathGopls)
	}
}
