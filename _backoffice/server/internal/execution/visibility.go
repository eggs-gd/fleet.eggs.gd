package execution

import "github.com/eggs-gd/core.eggs.gd/internal/executionapi"

type VisibilityClass = executionapi.VisibilityClass

const (
	VisibilityAppVisible  = executionapi.VisibilityAppVisible
	VisibilityCLIVisible  = executionapi.VisibilityCLIVisible
	VisibilityCoreVisible = executionapi.VisibilityCoreVisible
	VisibilityHeadless    = executionapi.VisibilityHeadless
	VisibilityUnknown     = executionapi.VisibilityUnknown

	LaunchModeAllowCoreVisible = executionapi.LaunchModeAllowCoreVisible
)

// VisibilityModeForSession derives operator-facing visibility when the session
// record does not already carry an explicit mode.
func VisibilityModeForSession(session RuntimeSession) string {
	if session.VisibilityMode != "" {
		return session.VisibilityMode
	}
	if caps := session.Capabilities; caps.OperatorVisibility != "" {
		return caps.OperatorVisibility
	}
	if session.Backend == BackendCursorVisible || session.CursorChatID != "" {
		return string(VisibilityCLIVisible)
	}
	if session.RemoteControlURL != "" || session.Backend == BackendBackgroundRemote {
		return string(VisibilityAppVisible)
	}
	switch session.Backend {
	case BackendCodexAppServer, "terminal":
		return string(VisibilityCoreVisible)
	case BackendGeminiHeadless, "process":
		return string(VisibilityHeadless)
	case "":
		return string(VisibilityUnknown)
	default:
		if len(session.Command) > 0 {
			return string(VisibilityHeadless)
		}
		return string(VisibilityUnknown)
	}
}
