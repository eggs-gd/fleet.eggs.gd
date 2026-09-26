package server

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/appconfig"
	"github.com/eggs-gd/fleet.eggs.gd/internal/execution"
	"github.com/eggs-gd/fleet.eggs.gd/internal/settings"
)

type managerSessionStarter func(ctx context.Context, agent, workingDir string) (string, error)

func registerManagerProvisionRoutes(mux *http.ServeMux, cfg Config) {
	mux.HandleFunc("/api/manager/setup", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]bool{"needed": managerSetupNeeded(cfg.CoreRoot, cfg.DataRootCreated)})
	})
	mux.HandleFunc("/api/manager/session", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Agent     string `json:"agent"`
			Workspace string `json:"workspace"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		status, warning, err := createManagerSession(r.Context(), cfg, req.Agent, req.Workspace, execution.StartManagerSession)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, struct {
			ManagerBindingStatus
			Warning string `json:"warning,omitempty"`
		}{status, warning})
	})
	mux.HandleFunc("/api/manager/adopt", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Agent    string `json:"agent"`
			ThreadID string `json:"threadId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		status, warning, err := adoptManagerSession(r.Context(), cfg, req.Agent, req.ThreadID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, struct {
			ManagerBindingStatus
			Warning string `json:"warning,omitempty"`
		}{status, warning})
	})
}

func managerSetupNeeded(coreRoot string, fresh bool) bool {
	if !fresh {
		return false
	}
	return !managerBindingStatus(coreRoot).Bound
}

func createManagerSession(ctx context.Context, cfg Config, agent, workspace string, start managerSessionStarter) (ManagerBindingStatus, string, error) {
	agent = strings.TrimSpace(agent)
	root, err := allowedManagerWorkspace(cfg.CoreRoot, workspace)
	if err != nil {
		return ManagerBindingStatus{}, "", err
	}
	mcpStatus, err := settings.EnsureManagerMCP(root, agent, cfg.Addr, cfg.LaunchToken)
	if err != nil {
		return ManagerBindingStatus{}, "", err
	}
	if !mcpStatus.Written || !mcpStatus.Matches {
		return ManagerBindingStatus{}, "", errString("fleet MCP was not written for " + agent)
	}
	threadID := execution.CursorWorkspaceBinding
	if agent != "cursor" {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		threadID, err = start(ctx, agent, root)
		if err != nil {
			return ManagerBindingStatus{}, "", err
		}
	}
	if err := saveManagerIdentity(root, agent, threadID, root); err != nil {
		return ManagerBindingStatus{}, "", err
	}
	warningText := ""
	if agent == "codex" {
		warningText = settings.CodexTrustWarning
	}
	return managerBindingStatus(root), warningText, nil
}

func adoptManagerSession(ctx context.Context, cfg Config, agent, threadID string) (ManagerBindingStatus, string, error) {
	agent = strings.TrimSpace(agent)
	threadID = strings.TrimSpace(threadID)
	if agent == "" || threadID == "" {
		return ManagerBindingStatus{}, "", errString("agent and threadId are required")
	}
	root, err := filepath.Abs(cfg.CoreRoot)
	if err != nil {
		return ManagerBindingStatus{}, "", err
	}
	if _, err := settings.EnsureManagerMCP(root, agent, cfg.Addr, cfg.LaunchToken); err != nil {
		return ManagerBindingStatus{}, "", err
	}
	threads, err := listManagerThreads(ctx, cfg.RuntimeRoot, agent)
	if err != nil {
		return ManagerBindingStatus{}, "", errString("this provider does not list session roots")
	}
	for _, thread := range threads {
		if thread.ID != threadID {
			continue
		}
		cwd := strings.TrimSpace(thread.Cwd)
		if cwd == "" {
			return ManagerBindingStatus{}, "", errString("session root is not the data root")
		}
		abs, absErr := filepath.Abs(cwd)
		if absErr != nil {
			abs = cwd
		}
		if abs != root {
			return ManagerBindingStatus{}, "", errString("session root is not the data root")
		}
		if err := saveManagerIdentity(root, agent, threadID, root); err != nil {
			return ManagerBindingStatus{}, "", err
		}
		warning := ""
		if agent == "codex" {
			warning = settings.CodexTrustWarning
		}
		return managerBindingStatus(root), warning, nil
	}
	return ManagerBindingStatus{}, "", errString("session root is not the data root")
}

func allowedManagerWorkspace(coreRoot, requested string) (string, error) {
	coreAbs, err := filepath.Abs(coreRoot)
	if err != nil {
		return "", err
	}
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return coreAbs, nil
	}
	abs, err := filepath.Abs(requested)
	if err != nil {
		return "", err
	}
	if abs == coreAbs {
		return abs, nil
	}
	stored, err := appconfig.Load()
	if err != nil {
		return "", err
	}
	storedAbs, err := appconfig.ExpandHome(stored.DataRoot)
	if err != nil {
		return "", err
	}
	storedAbs, err = filepath.Abs(storedAbs)
	if err != nil {
		return "", err
	}
	if abs == storedAbs {
		return abs, nil
	}
	return "", errString("workspace is not the data root")
}

func saveManagerIdentity(root, agent, threadID, workspace string) error {
	overlay, err := settings.LoadOverlay(root)
	if err != nil {
		return err
	}
	overlay.Manager = settings.ManagerOverlay{
		Agent:     agent,
		ThreadID:  threadID,
		Workspace: workspace,
		BoundAt:   time.Now().UTC().Format(time.RFC3339),
	}
	return settings.SaveOverlay(root, overlay)
}

type errString string

func (e errString) Error() string { return string(e) }
