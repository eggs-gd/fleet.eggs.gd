package server

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/settings"
)

func TestManagerBindingStatusReflectsOverlay(t *testing.T) {
	root := t.TempDir()
	if got := managerBindingStatus(root); got.Bound || got.Agent != "" || got.ThreadID != "" {
		t.Fatalf("unbound status = %#v", got)
	}

	if err := settings.SaveOverlay(root, settings.Overlay{Manager: settings.ManagerOverlay{Agent: "codex", ThreadID: "thread-1", Workspace: root}}); err != nil {
		t.Fatal(err)
	}
	got := managerBindingStatus(root)
	if !got.Bound || got.Agent != "codex" || got.ThreadID != "thread-1" || got.Workspace != root {
		t.Fatalf("bound status = %#v", got)
	}
}

func TestListManagerThreadsUnimplementedAgentFails(t *testing.T) {
	root := t.TempDir()
	_, err := listManagerThreads(context.Background(), root, "claude")
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("err = %v", err)
	}
}

func TestListManagerThreadsCodexSurfacesResolutionError(t *testing.T) {
	root := t.TempDir()
	t.Setenv("FLEET_CODEX_BINARY", filepath.Join(root, "definitely-missing-codex-binary"))
	_, err := listManagerThreads(context.Background(), root, "codex")
	if err == nil {
		t.Fatal("expected an error when codex cannot be resolved")
	}
}
