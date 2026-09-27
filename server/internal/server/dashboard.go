package server

import (
	"encoding/json"
	"net/http"

	"github.com/eggs-gd/fleet.eggs.gd/internal/execution"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// registerDashboardRoutes wires the backoffice dashboard REST surface
// (_docs/ARCHITECTURE.md).
//
// Dashboard uses only:
//
//	REST -> TaskService   (via DashboardSurface.CreateTask / PatchTask)
//	REST -> execution     (State / ControlSession)
//
// It polls state and sends commands. It does not subscribe to Go channels,
// participate in execution, or own business lifecycle logic beyond calling
// TaskService.
func registerDashboardRoutes(mux *http.ServeMux, dashboard DashboardSurface) {
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, dashboard.State())
	})
	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPatch:
			var patch tasklifecycle.TaskPatch
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			task, err := dashboard.PatchTask(patch)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, task)
		case http.MethodPost:
			var create tasklifecycle.TaskCreateRequest
			if err := json.NewDecoder(r.Body).Decode(&create); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			task, err := dashboard.CreateTask(create)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, task)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/sessions/control", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var patch execution.SessionControlPatch
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := dashboard.ControlSession(patch); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
}
