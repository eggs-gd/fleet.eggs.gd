package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAppConfigRouteGet(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	mux := http.NewServeMux()
	registerAppConfigRoutes(mux, Config{CoreRoot: root, DataRootSource: "flag"})

	req := httptest.NewRequest(http.MethodGet, "/api/app-config", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var view appConfigView
	if err := json.Unmarshal(rec.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view.EffectiveRoot != root || view.Source != "flag" {
		t.Fatalf("view = %#v", view)
	}
}

func TestAppConfigRouteRejectsUnsupportedMethod(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	mux := http.NewServeMux()
	registerAppConfigRoutes(mux, Config{CoreRoot: t.TempDir()})

	req := httptest.NewRequest(http.MethodPost, "/api/app-config", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestAppConfigPatchMovesDataAndPersistsPointer(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	base := t.TempDir()
	oldRoot := filepath.Join(base, "old-data")
	if err := os.MkdirAll(filepath.Join(oldRoot, "Work"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(oldRoot, "Work", "INDEX.md"), []byte("index"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	newRoot := filepath.Join(base, "relocated-data")

	mux := http.NewServeMux()
	registerAppConfigRoutes(mux, Config{CoreRoot: oldRoot, DataRootSource: "config"})

	body, _ := json.Marshal(map[string]string{"dataRoot": newRoot})
	req := httptest.NewRequest(http.MethodPatch, "/api/app-config", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	var view appConfigView
	if err := json.Unmarshal(rec.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if !view.RestartRequired {
		t.Fatal("expected RestartRequired=true after a relocation")
	}
	if view.DataRoot != newRoot {
		t.Fatalf("DataRoot = %q, want %q", view.DataRoot, newRoot)
	}

	if _, err := os.Stat(filepath.Join(newRoot, "Work", "INDEX.md")); err != nil {
		t.Fatalf("expected content moved to new root: %v", err)
	}
	if _, err := os.Stat(oldRoot); !os.IsNotExist(err) {
		t.Fatalf("expected old root to be gone, err = %v", err)
	}
}

func TestAppConfigPatchRejectsEmptyDataRoot(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	mux := http.NewServeMux()
	registerAppConfigRoutes(mux, Config{CoreRoot: t.TempDir()})

	body, _ := json.Marshal(map[string]string{"dataRoot": "  "})
	req := httptest.NewRequest(http.MethodPatch, "/api/app-config", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
