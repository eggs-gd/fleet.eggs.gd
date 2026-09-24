package executionapi

// VisibilityClass is Core's honest label for how an operator can see a
// daemon-launched agent session (CORE-70 / CORE-130).
type VisibilityClass string

const (
	VisibilityAppVisible  VisibilityClass = "app_visible"
	VisibilityCLIVisible  VisibilityClass = "cli_visible"
	VisibilityCoreVisible VisibilityClass = "core_visible"
	VisibilityHeadless    VisibilityClass = "headless"
	VisibilityUnknown     VisibilityClass = "unknown"
)

// AllowsDaemonAutoLaunch reports whether this visibility class is eligible for
// normal daemon auto-launch.
//
// CORE-130: core_visible is allowed when the adapter is LiveReady. Codex
// app-server sessions are operator-reachable through Codex Remote (paired
// clients) plus Core dashboard controls even without a Claude-style deep link
// or ChatGPT desktop sidebar entry. headless remains blocked.
func (visibility VisibilityClass) AllowsDaemonAutoLaunch() bool {
	switch visibility {
	case VisibilityAppVisible, VisibilityCLIVisible, VisibilityCoreVisible:
		return true
	default:
		return false
	}
}

// LaunchModeAllowCoreVisible is a legacy no-op kept for older task cards and
// fixtures. CORE-130 made normal Codex app-server auto-launch LiveReady by
// default, so this mode is no longer required to pass the visibility gate.
const LaunchModeAllowCoreVisible = "allow_core_visible"
