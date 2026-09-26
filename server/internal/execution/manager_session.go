package execution

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/execution/providers"
)

// CursorWorkspaceBinding is stored in place of a thread id when Cursor is the
// Manager. Cursor has no session Fleet can create or resume: the Manager is the
// data root opened in Cursor, so there is nothing to look up by this value. It
// only marks the binding as "chosen, no thread". Anything that opens, resumes
// or links a Manager thread must skip it.
const CursorWorkspaceBinding = "workspace"

// StartManagerSession creates a provider session whose working directory is
// the data root. It does not claim a task or a concurrency slot. Cursor does
// not spawn cursor-agent: that chat stays on this computer and is not the
// phone session. The Manager is Cursor opened on the data root.
func StartManagerSession(ctx context.Context, agent, workingDir string) (string, error) {
	switch agent {
	case "cursor":
		if strings.TrimSpace(workingDir) == "" {
			return "", fmt.Errorf("cursor manager requires a workspace")
		}
		return CursorWorkspaceBinding, nil
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
