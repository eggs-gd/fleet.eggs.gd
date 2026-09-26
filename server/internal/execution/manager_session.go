package execution

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/execution/providers"
)

// StartManagerSession creates a provider session whose working directory is
// the data root. It does not claim a task or a concurrency slot. Cursor is
// refused: cursor-agent cannot be shown in Cursor's own UI.
func StartManagerSession(ctx context.Context, agent, workingDir string) (string, error) {
	switch agent {
	case "cursor":
		return "", fmt.Errorf("cursor cannot be the manager")
	case "codex":
		ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		return providers.StartCodexManagerThread(ctx, workingDir, io.Discard)
	case "claude":
		ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		return providers.StartClaudeManagerSession(ctx, workingDir)
	case "gemini":
		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		return providers.StartGeminiManagerSession(ctx, workingDir)
	default:
		return "", fmt.Errorf("unknown manager agent %q", agent)
	}
}
