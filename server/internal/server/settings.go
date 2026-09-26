package server

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/settings"
	"github.com/eggs-gd/fleet.eggs.gd/internal/taskprovider"
)

type settingsRuntime struct {
	cfg     Config
	dash    *Dashboard
	scanner *settings.Scanner
}

func registerSettingsRoutes(mux *http.ServeMux, rt *settingsRuntime) {
	mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, rt.snapshot())
		case http.MethodPatch:
			rt.patch(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/settings/recheck", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		overlay, err := settings.LoadOverlay(rt.cfg.CoreRoot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		settings.ApplyOverlay(overlay)
		writeJSON(w, rt.snapshot())
	})
	mux.HandleFunc("/api/settings/rescan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		overlay, err := settings.LoadOverlay(rt.cfg.CoreRoot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		scanRoots, _ := settings.EffectiveScanRoots(overlay)
		if err := rt.scanner.Start(scanRoots, filepath.Join(rt.cfg.CoreRoot, "_registry")); err != nil {
			if errors.Is(err, settings.ErrScanBusy) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			if errors.Is(err, settings.ErrScanRootRequired) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		writeJSON(w, rt.scanner.Status())
	})
}

func (rt *settingsRuntime) snapshot() settings.Snapshot {
	if rt == nil {
		return settings.Snapshot{}
	}
	return settingsSnapshot(rt.cfg, rt.dash, rt.scanner)
}

func (rt *settingsRuntime) patch(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	patch, err := settings.DecodePatch(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	overlay, err := settings.LoadOverlay(rt.cfg.CoreRoot)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	merged := settings.MergeOverlay(overlay, patch)
	warnings, err := settings.ValidateOverlay(merged)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := settings.SaveOverlay(rt.cfg.CoreRoot, merged); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	settings.ApplyOverlay(merged)
	writeJSON(w, settings.PatchResult{Snapshot: rt.snapshot(), Warnings: warnings})
}

func settingsSnapshot(cfg Config, dash *Dashboard, scanner *settings.Scanner) settings.Snapshot {
	state := State{}
	if dash != nil {
		state = dash.State()
	}
	started := cfg.StartedAt
	if started.IsZero() {
		started = time.Now()
	}
	scan := settings.ScanStatus{State: settings.ScanIdle}
	if scanner != nil {
		scan = scanner.Status()
	}
	return settings.Build(settings.Input{
		Version:            cfg.Version,
		Addr:               cfg.Addr,
		StartedAt:          started,
		CoreRoot:           cfg.CoreRoot,
		RuntimeRoot:        cfg.RuntimeRoot,
		DryRun:             cfg.DryRun,
		SessionTimeout:     cfg.SessionTimeout,
		SessionTimeoutFlag: cfg.SessionTimeoutFlag,
		Workspaces:         state.Workspaces,
		Projects:           state.Projects,
		Registry:           state.Registry,
		Tasks:              state.Tasks,
		RuntimeSessions:    state.RuntimeSessions,
		SessionGroups:      state.SessionGroups,
		OrphanedTasks:      state.OrphanedTasks,
		Scan:               scan,
	})
}

func newSettingsRuntime(cfg Config, app *App) *settingsRuntime {
	rt := &settingsRuntime{cfg: cfg}
	if app != nil {
		rt.dash = app.Dashboard
		rt.scanner = settings.NewScanner(settings.DefaultSniff(cfg.CoreRoot), func() error {
			return taskprovider.ReloadRegistry(app.Store, cfg.CoreRoot)
		})
	} else {
		rt.scanner = settings.NewScanner(nil, nil)
	}
	return rt
}
