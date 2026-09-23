package execution

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/eggs-gd/core.eggs.gd/internal/execution/providers"
)

// ManagerRelayResult is the outcome of forwarding one Manager Bar message
// into a bound, project-less agent session (internal/settings Manager
// overlay) instead of routing it through fast-path/LLM command parsing.
type ManagerRelayResult struct {
	ThreadID  string
	ReplyText string
}

// RelayManagerMessage dispatches a single manager-chat turn to the bound
// agent's provider. It is the execution runtime's boundary for provider
// code on this path, matching how task launches never let internal/server
// or internal/manager call internal/execution/providers directly. Only
// "codex" is wired today; other known agents return an explicit error
// instead of silently going nowhere.
func RelayManagerMessage(ctx context.Context, agent, workingDir, threadID, text string, timeout time.Duration, logFile io.Writer) (ManagerRelayResult, error) {
	switch agent {
	case "codex":
		result, err := providers.RelayCodexManagerMessage(ctx, workingDir, threadID, text, timeout, logFile)
		return ManagerRelayResult{ThreadID: result.ThreadID, ReplyText: result.ReplyText}, err
	default:
		return ManagerRelayResult{}, fmt.Errorf("manager relay for %q is not implemented yet", agent)
	}
}

// ManagerThread is one candidate session an operator can bind the Manager
// Bar to, trimmed for a picker (id/name/cwd/updatedAt only).
type ManagerThread struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Cwd       string `json:"cwd,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// ListManagerThreads lists the bindable sessions for an agent, newest first.
// Only "codex" is wired today, matching RelayManagerMessage.
func ListManagerThreads(ctx context.Context, agent string, timeout time.Duration, logFile io.Writer) ([]ManagerThread, error) {
	switch agent {
	case "codex":
		summaries, err := providers.ListCodexManagerThreads(ctx, timeout, logFile)
		if err != nil {
			return nil, err
		}
		threads := make([]ManagerThread, 0, len(summaries))
		for _, s := range summaries {
			threads = append(threads, ManagerThread{ID: s.ID, Name: s.Name, Cwd: s.Cwd, UpdatedAt: s.UpdatedAt})
		}
		return threads, nil
	default:
		return nil, fmt.Errorf("listing manager sessions for %q is not implemented yet", agent)
	}
}
