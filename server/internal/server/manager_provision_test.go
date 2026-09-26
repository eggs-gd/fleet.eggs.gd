package server

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/settings"
)

func TestManagerSetupNeededOnlyOnFreshUnboundRoot(t *testing.T) {
	root := t.TempDir()
	if managerSetupNeeded(root, false) {
		t.Fatal("existing root should not offer setup")
	}
	if !managerSetupNeeded(root, true) {
		t.Fatal("fresh unbound root should offer setup")
	}
	if err := settings.SaveOverlay(root, settings.Overlay{Manager: settings.ManagerOverlay{Agent: "codex", ThreadID: "thread-1"}}); err != nil {
		t.Fatal(err)
	}
	if managerSetupNeeded(root, true) {
		t.Fatal("fresh root with a manager should not offer setup")
	}
}

func TestCreateManagerSessionBindsCursorWithoutLaunch(t *testing.T) {
	root := t.TempDir()
	started := false
	status, _, err := createManagerSession(context.Background(), Config{CoreRoot: root, Addr: "127.0.0.1:8787"}, "cursor", "", func(context.Context, string, string) (string, error) {
		started = true
		return "nope", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if started {
		t.Fatal("cursor manager must not start a provider process")
	}
	if !status.Bound || status.Agent != "cursor" || status.ThreadID != "workspace" {
		t.Fatalf("status = %#v", status)
	}
}

func TestCreateManagerSessionWritesMCPBeforeStart(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	var gotDir string
	status, warning, err := createManagerSession(context.Background(), Config{CoreRoot: root, Addr: "127.0.0.1:8787"}, "codex", "", func(_ context.Context, agent, workingDir string) (string, error) {
		if agent != "codex" {
			return "", errors.New(agent)
		}
		gotDir = workingDir
		mcp := settings.InspectManagerMCP(workingDir, "codex", "127.0.0.1:8787", "")
		if !mcp.Matches {
			return "", errors.New("mcp missing before start")
		}
		return "thread-9", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotDir != root && gotDir != absPath(root) {
		t.Fatalf("cwd = %s", gotDir)
	}
	if !status.Bound || status.ThreadID != "thread-9" || status.Workspace == "" {
		t.Fatalf("status = %#v", status)
	}
	if warning == "" {
		t.Fatal("expected the codex trust warning")
	}
}

func TestAdoptManagerSessionRequiresMatchingCwd(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	_, _, err := adoptManagerSession(context.Background(), Config{CoreRoot: root, RuntimeRoot: root, Addr: "127.0.0.1:8787"}, "claude", "session-1")
	if err == nil {
		t.Fatal("expected adopt to fail when the provider cannot list cwd")
	}
}

func absPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
