package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eggs-gd/fleet.eggs.gd/internal/settings"
)

func TestSettingsRouteGetAndRejectsPost(t *testing.T) {
	root := t.TempDir()
	mux := http.NewServeMux()
	registerSettingsRoutes(mux, &settingsRuntime{
		cfg:     Config{CoreRoot: root, Version: "0.0.1"},
		scanner: settings.NewScanner(nil, nil),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var snap settings.Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}
	if snap.General.Version != "0.0.1" {
		t.Fatalf("snap = %#v", snap)
	}
	if snap.Projects.ScanRootsWritable != true {
		t.Fatal("scan roots must be writable")
	}

	post := httptest.NewRequest(http.MethodPost, "/api/settings", nil)
	denied := httptest.NewRecorder()
	mux.ServeHTTP(denied, post)
	if denied.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want 405", denied.Code)
	}
}

func TestSettingsPatchRejectsUnknownKeysAndRelativeExecutable(t *testing.T) {
	root := t.TempDir()
	mux := http.NewServeMux()
	registerSettingsRoutes(mux, &settingsRuntime{
		cfg:     Config{CoreRoot: root},
		scanner: settings.NewScanner(nil, nil),
	})

	unknown := httptest.NewRequest(http.MethodPatch, "/api/settings", strings.NewReader(`{"theme":"dark"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, unknown)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown key status = %d body=%s", rec.Code, rec.Body.String())
	}

	relative := httptest.NewRequest(http.MethodPatch, "/api/settings", strings.NewReader(`{"agents":{"codex":{"executable":"codex"}}}`))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, relative)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "absolute") {
		t.Fatalf("relative executable status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSettingsRecheckDoesNotWriteOverlay(t *testing.T) {
	root := t.TempDir()
	mux := http.NewServeMux()
	registerSettingsRoutes(mux, &settingsRuntime{
		cfg:     Config{CoreRoot: root},
		scanner: settings.NewScanner(nil, nil),
	})

	path := filepath.Join(root, settings.OverlayFileName)
	req := httptest.NewRequest(http.MethodPost, "/api/settings/recheck", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("recheck status = %d body=%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("recheck must not create overlay: %v", err)
	}
}

func TestSettingsRescanConflictWhenBusy(t *testing.T) {
	root := t.TempDir()
	scanRoot := filepath.Join(root, "projects")
	if err := os.MkdirAll(scanRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "core.local.yaml"), []byte("scanRoots:\n  - "+scanRoot+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	block := make(chan struct{})
	scanner := settings.NewScanner(func([]string, string) error {
		close(started)
		<-block
		return nil
	}, nil)
	mux := http.NewServeMux()
	registerSettingsRoutes(mux, &settingsRuntime{
		cfg:     Config{CoreRoot: root},
		scanner: scanner,
	})

	first := httptest.NewRequest(http.MethodPost, "/api/settings/rescan", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, first)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("first rescan status = %d body=%s", rec.Code, rec.Body.String())
	}
	<-started

	second := httptest.NewRequest(http.MethodPost, "/api/settings/rescan", nil)
	busy := httptest.NewRecorder()
	mux.ServeHTTP(busy, second)
	if busy.Code != http.StatusConflict {
		t.Fatalf("busy rescan status = %d body=%s", busy.Code, busy.Body.String())
	}
	close(block)
}

func TestSettingsPatchPersistsEnabled(t *testing.T) {
	root := t.TempDir()
	mux := http.NewServeMux()
	registerSettingsRoutes(mux, &settingsRuntime{
		cfg:     Config{CoreRoot: root},
		scanner: settings.NewScanner(nil, nil),
	})
	req := httptest.NewRequest(http.MethodPatch, "/api/settings", strings.NewReader(`{"agents":{"codex":{"enabled":false}}}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	overlay, err := settings.LoadOverlay(root)
	if err != nil {
		t.Fatal(err)
	}
	agent := overlay.Agent("codex")
	if agent.Enabled == nil || *agent.Enabled {
		t.Fatalf("overlay enabled = %#v", agent.Enabled)
	}
	var result settings.PatchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	for _, item := range result.Snapshot.Agents.Providers {
		if item.ID == "codex" && item.Enabled.Value {
			t.Fatal("GET snapshot after PATCH still shows codex enabled")
		}
	}
}
