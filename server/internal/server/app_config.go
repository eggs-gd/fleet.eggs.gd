package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/appconfig"
)

// appConfigView is the App-level (not Data-level) pointer to where the Data
// root lives. It is intentionally separate from settings.Snapshot: that
// reflects state read FROM the Data root, whereas this describes how the
// Data root's own location was found. See internal/appconfig's doc comment.
type appConfigView struct {
	DataRoot        string `json:"dataRoot"`
	EffectiveRoot   string `json:"effectiveRoot"`
	Source          string `json:"source"`
	RestartRequired bool   `json:"restartRequired"`
}

func registerAppConfigRoutes(mux *http.ServeMux, cfg Config) {
	mux.HandleFunc("/api/app-config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, currentAppConfigView(cfg))
		case http.MethodPatch:
			patchAppConfig(w, r, cfg)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func currentAppConfigView(cfg Config) appConfigView {
	stored, _ := appconfig.Load()
	return appConfigView{
		DataRoot:      stored.DataRoot,
		EffectiveRoot: cfg.CoreRoot,
		Source:        cfg.DataRootSource,
	}
}

func patchAppConfig(w http.ResponseWriter, r *http.Request, cfg Config) {
	var req struct {
		DataRoot string `json:"dataRoot"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.DataRoot) == "" {
		http.Error(w, "dataRoot is required", http.StatusBadRequest)
		return
	}
	resolved, err := appconfig.ExpandHome(req.DataRoot)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := appconfig.MoveDataRoot(cfg.CoreRoot, resolved); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err := appconfig.Save(appconfig.Config{DataRoot: resolved}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	view := currentAppConfigView(cfg)
	view.RestartRequired = resolved != cfg.CoreRoot
	writeJSON(w, view)
}
