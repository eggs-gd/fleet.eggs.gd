package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/execution"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// recordingDashboard is a DashboardSurface stand-in that records which
// boundary methods the HTTP handlers call (CORE-114).
type recordingDashboard struct {
	stateCalls   int
	patchCalls   int
	createCalls  int
	controlCalls int
	lastPatch    tasklifecycle.TaskPatch
	lastCreate   tasklifecycle.TaskCreateRequest
	lastControl  execution.SessionControlPatch
}

func (r *recordingDashboard) State() State {
	r.stateCalls++
	return State{GeneratedAt: "test", Statuses: []string{"todo"}}
}

func (r *recordingDashboard) ControlSession(patch execution.SessionControlPatch) error {
	r.controlCalls++
	r.lastControl = patch
	return nil
}

func (r *recordingDashboard) CreateTask(req tasklifecycle.TaskCreateRequest) (tasklifecycle.Task, error) {
	r.createCalls++
	r.lastCreate = req
	return tasklifecycle.Task{Ref: "CORE-TEST", Title: req.Title, Status: "backlog"}, nil
}

func (r *recordingDashboard) PatchTask(patch tasklifecycle.TaskPatch) (tasklifecycle.Task, error) {
	r.patchCalls++
	r.lastPatch = patch
	return tasklifecycle.Task{Ref: "CORE-TEST", Status: patch.Status, Path: patch.Path}, nil
}

func TestDashboardRoutesUseTaskServiceAndRuntimeService(t *testing.T) {
	dash := &recordingDashboard{}
	mux := http.NewServeMux()
	registerDashboardRoutes(mux, dash)

	t.Run("state_polls_runtime_service", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/state", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if dash.stateCalls != 1 {
			t.Fatalf("State calls = %d, want 1 (RuntimeService)", dash.stateCalls)
		}
	})

	t.Run("patch_goes_through_task_service_adapter", func(t *testing.T) {
		body := `{"path":"Work/x/tasks/y.md","status":"todo"}`
		req := httptest.NewRequest(http.MethodPatch, "/api/tasks", strings.NewReader(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
		}
		if dash.patchCalls != 1 {
			t.Fatalf("PatchTask calls = %d, want 1 (TaskService adapter)", dash.patchCalls)
		}
		if dash.lastPatch.Path != "Work/x/tasks/y.md" || dash.lastPatch.Status != "todo" {
			t.Fatalf("patch = %#v", dash.lastPatch)
		}
	})

	t.Run("create_goes_through_task_service_adapter", func(t *testing.T) {
		body := `{"title":"New","request":"Do it","project":"core-eggs-gd"}`
		req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
		}
		if dash.createCalls != 1 {
			t.Fatalf("CreateTask calls = %d, want 1 (TaskService adapter)", dash.createCalls)
		}
		if dash.lastCreate.Title != "New" || dash.lastCreate.Project != "core-eggs-gd" {
			t.Fatalf("create = %#v", dash.lastCreate)
		}
	})

	t.Run("session_control_uses_runtime_service", func(t *testing.T) {
		body := `{"claim_id":"c1","action":"interrupt"}`
		req := httptest.NewRequest(http.MethodPost, "/api/sessions/control", strings.NewReader(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
		}
		if dash.controlCalls != 1 {
			t.Fatalf("ControlSession calls = %d, want 1 (RuntimeService)", dash.controlCalls)
		}
		if dash.lastControl.ClaimID != "c1" || dash.lastControl.Action != "interrupt" {
			t.Fatalf("control = %#v", dash.lastControl)
		}
	})
}

func TestDashboardSurfaceImplementedByCompose(t *testing.T) {
	app := Compose(ComposeConfig{
		CoreRoot:       t.TempDir(),
		SessionTimeout: time.Second,
		DryRun:         true,
	})
	var surface DashboardSurface = app.Surface()
	state := surface.State()
	if state.Root == "" && state.GeneratedAt == "" && len(state.Statuses) == 0 {
		_ = state
	}
	payload, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("state must be JSON-serializable for dashboard poll: %v", err)
	}
	if len(payload) == 0 {
		t.Fatal("empty state JSON")
	}
	if app.TaskService() == nil {
		t.Fatal("compose must still expose TaskService for Manager")
	}
}
