package execution

import (
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// ClassifyProviderError classifies a provider operation failure. The
// classification and the task-facing comment it renders into both live in
// tasklifecycle (task-facing errors and operator comments are lifecycle
// domain); this file only adapts corechain's RuntimeSession into the
// primitive tasklifecycle.ProviderErrorContext, since RuntimeSession is a
// corechain/runtime concept the lifecycle package does not know about.
func ClassifyProviderError(provider string, backend string, operation string, err error, output string) *tasklifecycle.ProviderError {
	return tasklifecycle.ClassifyProviderError(provider, backend, operation, err, output)
}

func ProviderErrorComment(session RuntimeSession) string {
	return tasklifecycle.ProviderErrorComment(session.ProviderError, providerErrorContextFor(session))
}

func providerErrorContextFor(session RuntimeSession) tasklifecycle.ProviderErrorContext {
	return tasklifecycle.ProviderErrorContext{
		Agent:            session.Agent,
		ClaimID:          session.ClaimID,
		BackgroundID:     session.BackgroundID,
		ThreadID:         session.CodexThreadID,
		TurnID:           session.CodexTurnID,
		CursorChatID:     session.CursorChatID,
		RemoteControlURL: session.RemoteControlURL,
		LogPath:          session.LogPath,
	}
}
