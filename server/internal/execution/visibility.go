package execution

import "github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"

type VisibilityClass = executionapi.VisibilityClass

const (
	VisibilityAppVisible   = executionapi.VisibilityAppVisible
	VisibilityCLIVisible   = executionapi.VisibilityCLIVisible
	VisibilityFleetVisible = executionapi.VisibilityFleetVisible
	VisibilityHeadless     = executionapi.VisibilityHeadless
	VisibilityUnknown      = executionapi.VisibilityUnknown
)

// VisibilityModeForSession derives operator-facing visibility when the session
// record does not already carry an explicit mode.
func VisibilityModeForSession(session RuntimeSession) string {
	if session.VisibilityMode != "" {
		return string(executionapi.NormalizeVisibility(session.VisibilityMode))
	}
	if caps := session.Capabilities; caps.OperatorVisibility != "" {
		return string(executionapi.NormalizeVisibility(caps.OperatorVisibility))
	}
	if session.Backend == BackendCursorVisible || session.CursorChatID != "" {
		return string(VisibilityCLIVisible)
	}
	if session.RemoteControlURL != "" || session.Backend == BackendBackgroundRemote {
		return string(VisibilityAppVisible)
	}
	switch session.Backend {
	case BackendCodexAppServer, "terminal":
		return string(VisibilityFleetVisible)
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
