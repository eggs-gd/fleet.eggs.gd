package server

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/manager"
	"github.com/eggs-gd/fleet.eggs.gd/internal/settings"
)

func TestRouteManagerTextUnboundFallsThrough(t *testing.T) {
	root := t.TempDir()
	_, handled := routeManagerText(context.Background(), root, root, "hello")
	if handled {
		t.Fatal("expected an unbound manager to fall through to manager.Service")
	}
}

func TestRouteManagerTextUnimplementedAgentFails(t *testing.T) {
	root := t.TempDir()
	if err := settings.SaveOverlay(root, settings.Overlay{Manager: settings.ManagerOverlay{Agent: "claude", ThreadID: "thread-1"}}); err != nil {
		t.Fatal(err)
	}
	response, handled := routeManagerText(context.Background(), root, root, "hello")
	if !handled {
		t.Fatal("expected a bound manager to be handled")
	}
	if response.OK {
		t.Fatal("expected failure for an agent with no relay implementation yet")
	}
	if response.Failure == nil || !strings.Contains(response.Failure.Message, "not implemented") {
		t.Fatalf("failure = %#v", response.Failure)
	}
}

func TestManagerBindingStatusReflectsOverlay(t *testing.T) {
	root := t.TempDir()
	if got := managerBindingStatus(root); got.Bound || got.Agent != "" || got.ThreadID != "" {
		t.Fatalf("unbound status = %#v", got)
	}

	if err := settings.SaveOverlay(root, settings.Overlay{Manager: settings.ManagerOverlay{Agent: "codex", ThreadID: "thread-1"}}); err != nil {
		t.Fatal(err)
	}
	got := managerBindingStatus(root)
	if !got.Bound || got.Agent != "codex" || got.ThreadID != "thread-1" {
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
	t.Setenv("CORE_CODEX_BINARY", filepath.Join(root, "definitely-missing-codex-binary"))
	_, err := listManagerThreads(context.Background(), root, "codex")
	if err == nil {
		t.Fatal("expected an error when codex cannot be resolved")
	}
}

func TestRouteManagerTextCodexBindingSurfacesProviderError(t *testing.T) {
	root := t.TempDir()
	if err := settings.SaveOverlay(root, settings.Overlay{Manager: settings.ManagerOverlay{Agent: "codex", ThreadID: "thread-1"}}); err != nil {
		t.Fatal(err)
	}
	// Force codex binary resolution to fail deterministically instead of
	// depending on (or actually spawning) whatever codex install the host
	// running this test happens to have.
	t.Setenv("CORE_CODEX_BINARY", filepath.Join(root, "definitely-missing-codex-binary"))

	response, handled := routeManagerText(context.Background(), root, root, "hello")
	if !handled {
		t.Fatal("expected a bound manager to be handled")
	}
	if response.OK {
		t.Fatal("expected failure when codex cannot be resolved")
	}
	if response.Failure == nil || response.Failure.Code != manager.FailureProviderError {
		t.Fatalf("failure = %#v", response.Failure)
	}
}
