package server

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eggs-gd/core.eggs.gd/internal/execution"
	"github.com/eggs-gd/core.eggs.gd/internal/manager"
	"github.com/eggs-gd/core.eggs.gd/internal/settings"
)

const managerRelayTimeout = 5 * time.Minute
const managerThreadListTimeout = 15 * time.Second

// ManagerBindingStatus is the cheap read the Manager Bar polls to show
// where it currently sends messages — a plain overlay read, not the full
// (and comparatively expensive) settings.Build() snapshot.
type ManagerBindingStatus struct {
	Bound    bool   `json:"bound"`
	Agent    string `json:"agent,omitempty"`
	ThreadID string `json:"threadId,omitempty"`
}

func managerBindingStatus(coreRoot string) ManagerBindingStatus {
	overlay, err := settings.LoadOverlay(coreRoot)
	if err != nil {
		return ManagerBindingStatus{}
	}
	agent := strings.TrimSpace(overlay.Manager.Agent)
	threadID := strings.TrimSpace(overlay.Manager.ThreadID)
	return ManagerBindingStatus{
		Bound:    agent != "" && threadID != "",
		Agent:    agent,
		ThreadID: threadID,
	}
}

// routeManagerText intercepts POST /api/manager/text ahead of
// manager.Service when a Manager session is bound (internal/settings
// overlay, set via Settings > Manager): the message forwards straight into
// that session instead of going through fast-path/LLM command parsing.
// handled is false when there is no bound session, in which case the
// caller must still call manager.Service.SubmitText.
func routeManagerText(ctx context.Context, coreRoot, text string) (response manager.Response, handled bool) {
	overlay, err := settings.LoadOverlay(coreRoot)
	if err != nil || strings.TrimSpace(overlay.Manager.Agent) == "" || strings.TrimSpace(overlay.Manager.ThreadID) == "" {
		return manager.Response{}, false
	}

	var logWriter io.Writer = io.Discard
	if logFile, err := os.OpenFile(filepath.Join(coreRoot, "_registry", "manager-relay.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		defer logFile.Close()
		logWriter = logFile
	}

	threadID := overlay.Manager.ThreadID
	result, err := execution.RelayManagerMessage(ctx, overlay.Manager.Agent, coreRoot, threadID, text, managerRelayTimeout, logWriter)
	if err != nil {
		return manager.Response{OK: false, Failure: &manager.Failure{Code: manager.FailureProviderError, Message: err.Error()}}, true
	}
	if result.ThreadID != "" && result.ThreadID != threadID {
		rebindManagerThread(coreRoot, result.ThreadID)
	}
	return manager.Response{
		OK: true,
		Result: &manager.Result{
			Action: "manager_relay",
			Status: "relayed",
			Ref:    result.ThreadID,
			Detail: result.ReplyText,
		},
	}, true
}

// listManagerThreads gives Settings > Manager a picker of recent, bindable
// sessions for an agent instead of asking the operator to go dig a raw
// thread id out of the agent's own CLI/history. Errors come back to the
// caller as-is (e.g. "not implemented yet" for claude/cursor, or a codex
// resolution failure) so the UI can show them rather than an empty list.
func listManagerThreads(ctx context.Context, coreRoot, agent string) ([]execution.ManagerThread, error) {
	var logWriter io.Writer = io.Discard
	if logFile, err := os.OpenFile(filepath.Join(coreRoot, "_registry", "manager-relay.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		defer logFile.Close()
		logWriter = logFile
	}
	return execution.ListManagerThreads(ctx, agent, managerThreadListTimeout, logWriter)
}

// rebindManagerThread persists a thread id Codex rotated to mid-relay (a
// resume that could not continue the saved thread falls back to starting a
// fresh one — see RelayCodexManagerMessage) so the next message continues
// there instead of retrying the stale id every time.
func rebindManagerThread(coreRoot, threadID string) {
	overlay, err := settings.LoadOverlay(coreRoot)
	if err != nil {
		return
	}
	overlay.Manager.ThreadID = threadID
	_ = settings.SaveOverlay(coreRoot, overlay)
}
