package executionapi

// VisibilityClass says how a person can see and continue a daemon-launched
// agent session. It is the only place these terms are defined.
//
//	app_visible    a link in the vendor's own app, on the phone or the desktop
//	cli_visible    a command in a terminal that reopens and continues the session
//	fleet_visible  Fleet's own dashboard controls, plus the provider's remote
//	               view when it has one; no link
//	headless       a log only, nothing to open
//	unknown        the adapter did not say
type VisibilityClass string

const (
	VisibilityAppVisible   VisibilityClass = "app_visible"
	VisibilityCLIVisible   VisibilityClass = "cli_visible"
	VisibilityFleetVisible VisibilityClass = "fleet_visible"
	VisibilityHeadless     VisibilityClass = "headless"
	VisibilityUnknown      VisibilityClass = "unknown"
)

// legacyVisibilityCoreVisible is the former name of fleet_visible. Session
// records written before the rename still carry it.
const legacyVisibilityCoreVisible = "core_visible"

// NormalizeVisibility maps a stored or reported value to a current class.
func NormalizeVisibility(value string) VisibilityClass {
	switch value {
	case legacyVisibilityCoreVisible:
		return VisibilityFleetVisible
	case "":
		return ""
	}
	return VisibilityClass(value)
}

// AllowsDaemonAutoLaunch reports whether a person can reach a session of this
// class, which is what automatic launch requires. A headless session is
// refused: if it stalls or asks something, nobody would know.
func (visibility VisibilityClass) AllowsDaemonAutoLaunch() bool {
	switch NormalizeVisibility(string(visibility)) {
	case VisibilityAppVisible, VisibilityCLIVisible, VisibilityFleetVisible:
		return true
	default:
		return false
	}
}

// Label is the plain-language name shown in Settings.
func (visibility VisibilityClass) Label() string {
	switch NormalizeVisibility(string(visibility)) {
	case VisibilityAppVisible:
		return "Vendor app link"
	case VisibilityCLIVisible:
		return "Terminal resume command"
	case VisibilityFleetVisible:
		return "Fleet controls"
	case VisibilityHeadless:
		return "Log only"
	default:
		return "Unknown"
	}
}

// Summary explains what a person can do with a session of this class.
func (visibility VisibilityClass) Summary() string {
	switch NormalizeVisibility(string(visibility)) {
	case VisibilityAppVisible:
		return "Open the session from the vendor's app on the phone or the desktop and continue it there."
	case VisibilityCLIVisible:
		return "Run a command in a terminal to reopen the session and continue it. There is no link."
	case VisibilityFleetVisible:
		return "Continue, interrupt or cancel from the Fleet dashboard. The provider's remote view may also show it. There is no link."
	case VisibilityHeadless:
		return "Nothing to open. Only the log shows what happened."
	default:
		return "The adapter did not report how to reach the session."
	}
}
