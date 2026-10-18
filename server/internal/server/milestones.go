package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

// registerMilestoneRoutes wires the per-project milestone REST surface.
// Milestones live on the project's own PROJECT.md card (internal/workfiles),
// not in the task store, so this talks to DashboardSurface's milestone
// methods directly rather than through TaskService.
func registerMilestoneRoutes(mux *http.ServeMux, dashboard DashboardSurface) {
	mux.HandleFunc("/api/projects/{id}/milestones", func(w http.ResponseWriter, r *http.Request) {
		project := strings.TrimSpace(r.PathValue("id"))
		if project == "" {
			http.Error(w, "project id is required", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			milestones, err := dashboard.Milestones(project)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, milestones)
		case http.MethodPost:
			var create struct {
				Title   string `json:"title"`
				Version string `json:"version"`
			}
			if err := json.NewDecoder(r.Body).Decode(&create); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			milestone, err := dashboard.CreateMilestone(project, create.Title, create.Version)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, milestone)
		case http.MethodPatch:
			var patch struct {
				ID      string  `json:"id"`
				State   *string `json:"state"`
				Title   *string `json:"title"`
				Version *string `json:"version"`
			}
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(patch.ID) == "" {
				http.Error(w, "milestone id is required", http.StatusBadRequest)
				return
			}
			milestone, err := dashboard.PatchMilestone(project, patch.ID, patch.State, patch.Title, patch.Version)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSON(w, milestone)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
