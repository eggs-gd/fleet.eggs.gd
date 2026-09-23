package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eggs-gd/core.eggs.gd/internal/providerconfig"
	"github.com/eggs-gd/core.eggs.gd/internal/taskprovider"
	tpopen "github.com/eggs-gd/core.eggs.gd/internal/taskprovider/open"
	"github.com/eggs-gd/core.eggs.gd/internal/taskprovider/plane"
)

func TestNewTaskProviderSelectsMarkdownByDefault(t *testing.T) {
	root := t.TempDir()
	provider, wiring := tpopen.Open(tpopen.OpenOptions{
		Root:  root,
		Board: taskprovider.NewBoard(root),
	})
	if provider.Type() != "markdown" {
		t.Fatalf("Type() = %q, want markdown", provider.Type())
	}
	if wiring.PollInterval > 0 {
		t.Fatalf("markdown wiring must not poll, got PollInterval=%s", wiring.PollInterval)
	}
	if _, ok := provider.(interface{ RebuildDerivedViews() error }); !ok {
		t.Fatal("markdown provider must expose RebuildDerivedViews")
	}
	if _, ok := provider.(taskprovider.ChangeSource); ok {
		t.Fatal("markdown provider must not implement ChangeSource")
	}
}

func TestNewTaskProviderSelectsPlane(t *testing.T) {
	root := t.TempDir()
	provider, wiring := tpopen.Open(tpopen.OpenOptions{
		Root: root,
		Settings: providerconfig.Settings{
			Type: "plane",
			Plane: providerconfig.PlaneSettings{
				Workspace:   "eggs_gd",
				ProjectID:   "proj-1",
				CoreProject: "core-eggs-gd",
			},
		},
		Board: taskprovider.NewBoard(root),
	})
	if provider.Type() != "plane" {
		t.Fatalf("Type() = %q, want plane", provider.Type())
	}
	if _, ok := provider.(interface{ RebuildDerivedViews() error }); ok {
		t.Fatal("plane provider must not expose RebuildDerivedViews")
	}
	if wiring.PollInterval < 45*time.Second {
		t.Fatalf("PollInterval = %s, want >= 45s", wiring.PollInterval)
	}
	if _, ok := provider.(taskprovider.ChangeSource); !ok {
		t.Fatal("plane provider must implement ChangeSource")
	}
}

func TestTaskproviderNewServiceAcceptsPlaneProvider(t *testing.T) {
	root := t.TempDir()
	provider, _ := tpopen.Open(tpopen.OpenOptions{
		Root: root,
		Settings: providerconfig.Settings{
			Type: "plane",
			Plane: providerconfig.PlaneSettings{
				Workspace:   "eggs_gd",
				ProjectID:   "proj-1",
				CoreProject: "core-eggs-gd",
			},
		},
		Board: taskprovider.NewBoard(root),
	})
	planeProvider, ok := provider.(*plane.Provider)
	if !ok {
		t.Fatalf("provider type = %T, want *plane.Provider", provider)
	}
	if planeProvider.Flow() == nil {
		t.Fatal("expected non-nil native Flow view")
	}
	if taskprovider.NewService(provider) == nil {
		t.Fatal("expected taskprovider.NewService to accept plane provider")
	}
}

func TestRuntimeBootstrapWithPlaneProviderSkipsWorkIndexRebuild(t *testing.T) {
	mux := http.NewServeMux()
	empty := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{}})
	}
	mux.HandleFunc("/api/v1/workspaces/eggs_gd/projects/proj-1/states/", empty)
	mux.HandleFunc("/api/v1/workspaces/eggs_gd/projects/proj-1/labels/", empty)
	mux.HandleFunc("/api/v1/workspaces/eggs_gd/projects/proj-1/work-items/", empty)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	root := t.TempDir()
	app := Compose(ComposeConfig{
		CoreRoot: root,
		TaskProvider: providerconfig.Settings{
			Type: "plane",
			Plane: providerconfig.PlaneSettings{
				Workspace:   "eggs_gd",
				BaseURL:     srv.URL,
				ProjectID:   "proj-1",
				CoreProject: "core-eggs-gd",
			},
		},
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if _, ok := app.TaskStore.(taskprovider.ChangeSource); !ok {
		t.Fatal("plane taskStore must implement ChangeSource")
	}
	if _, ok := app.TaskStore.(interface{ RebuildDerivedViews() error }); ok {
		t.Fatal("plane runtime must not rebuild derived Markdown views")
	}
	if _, err := os.Stat(filepath.Join(root, "Work", "INDEX.md")); !os.IsNotExist(err) {
		t.Fatalf("expected Work/INDEX.md to not be written under the Plane provider, stat err = %v", err)
	}
	if state := app.State(); len(state.Tasks) != 0 {
		t.Fatalf("expected zero tasks from an empty Plane project, got %d", len(state.Tasks))
	}
}
