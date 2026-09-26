package server

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/execution"
	"github.com/eggs-gd/fleet.eggs.gd/internal/runtimedb"
	"github.com/eggs-gd/fleet.eggs.gd/internal/settings"
)

const managerThreadListTimeout = 15 * time.Second

// ManagerBindingStatus is the overlay identity of the Manager session.
// Fleet does not send chat into that session.
type ManagerBindingStatus struct {
	Bound     bool   `json:"bound"`
	Agent     string `json:"agent,omitempty"`
	ThreadID  string `json:"threadId,omitempty"`
	Workspace string `json:"workspace,omitempty"`
}

func managerBindingStatus(coreRoot string) ManagerBindingStatus {
	overlay, err := settings.LoadOverlay(coreRoot)
	if err != nil {
		return ManagerBindingStatus{}
	}
	agent := strings.TrimSpace(overlay.Manager.Agent)
	threadID := strings.TrimSpace(overlay.Manager.ThreadID)
	return ManagerBindingStatus{
		Bound:     agent != "" && threadID != "",
		Agent:     agent,
		ThreadID:  threadID,
		Workspace: strings.TrimSpace(overlay.Manager.Workspace),
	}
}

// listManagerThreads lists sessions the provider can enumerate, including cwd
// when the provider returns it. The settings page keeps only those whose root
// is the data root.
func listManagerThreads(ctx context.Context, runtimeRoot, agent string) ([]execution.ManagerThread, error) {
	logWriter, closeLog := openManagerRelayLog(runtimeRoot)
	defer closeLog()
	return execution.ListManagerThreads(ctx, agent, managerThreadListTimeout, logWriter)
}

func filterThreadsByCwd(threads []execution.ManagerThread, cwd string) []execution.ManagerThread {
	want, err := filepath.Abs(cwd)
	if err != nil {
		want = cwd
	}
	out := make([]execution.ManagerThread, 0)
	for _, thread := range threads {
		got := strings.TrimSpace(thread.Cwd)
		if got == "" {
			continue
		}
		abs, absErr := filepath.Abs(got)
		if absErr != nil {
			abs = got
		}
		if abs == want {
			out = append(out, thread)
		}
	}
	return out
}

func openManagerRelayLog(runtimeRoot string) (io.Writer, func()) {
	log, err := runtimedb.OpenRelay(runtimeRoot)
	if err != nil {
		return io.Discard, func() {}
	}
	return log, func() { _ = log.Close() }
}
