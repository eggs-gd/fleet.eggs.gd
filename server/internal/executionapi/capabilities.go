package executionapi

const (
	BackendBackgroundRemote = "background-remote"
	BackendCodexAppServer   = "codex-app-server"
	BackendCursorVisible    = "cursor-visible"
	BackendGeminiHeadless   = "gemini-headless"
)

// SessionReuseLevel is Core's honesty marker for whether a launch backend's
// session-resume mechanism has been empirically confirmed to continue the same
// provider-visible session.
type SessionReuseLevel string

const (
	SessionReuseVerified    SessionReuseLevel = "verified"
	SessionReuseUnverified  SessionReuseLevel = "unverified"
	SessionReuseUnsupported SessionReuseLevel = "unsupported"
)

// CapabilityFlag is a tri-state provider capability used by restart recovery.
type CapabilityFlag string

const (
	CapabilityYes     CapabilityFlag = "yes"
	CapabilityNo      CapabilityFlag = "no"
	CapabilityUnknown CapabilityFlag = "unknown"
)

func (flag CapabilityFlag) IsYes() bool {
	return flag == CapabilityYes
}

func (flag CapabilityFlag) IsNo() bool {
	return flag == CapabilityNo
}

func (flag CapabilityFlag) IsUnknown() bool {
	return flag == CapabilityUnknown || flag == ""
}

// SessionReuseCapability records, per launch backend, whether Core may
// automatically attempt to resume a previous provider session on relaunch.
type SessionReuseCapability struct {
	Backend    string            `json:"backend"`
	Level      SessionReuseLevel `json:"level"`
	CanAttempt bool              `json:"can_attempt"`
	Reason     string            `json:"reason"`
}

// ProviderCapabilities is the explicit per-provider/backend contract Core
// exposes through runtime sessions and orphaned-task recovery.
type ProviderCapabilities struct {
	Provider string `json:"provider,omitempty"`
	Backend  string `json:"backend,omitempty"`

	CanDetectRunningSession   CapabilityFlag `json:"can_detect_running_session"`
	CanResumeSession          CapabilityFlag `json:"can_resume_session"`
	CanQueryThreadStatus      CapabilityFlag `json:"can_query_thread_status"`
	CanShowAppVisibleLink     CapabilityFlag `json:"can_show_app_visible_link"`
	CanAcceptOperatorInput    CapabilityFlag `json:"can_accept_operator_input"`
	CanConfirmTerminalOutcome CapabilityFlag `json:"can_confirm_terminal_outcome"`

	CanStartVisibleSession bool `json:"can_start_visible_session"`
	CanListSessions        bool `json:"can_list_sessions"`
	CanExposeOperatorPath  bool `json:"can_expose_operator_path"`
	CanDetectTerminalState bool `json:"can_detect_terminal_state"`

	OperatorVisibility     string `json:"operator_visibility,omitempty"`
	TerminalStateDetection string `json:"terminal_state_detection,omitempty"`
	Notes                  string `json:"notes,omitempty"`
}

func (caps ProviderCapabilities) ResumeSupported() bool {
	return caps.CanResumeSession.IsYes()
}

// UnknownCapabilities is what a provider and backend pair reports when no
// adapter has registered verified capabilities for it.
func UnknownCapabilities(provider string, backend string) ProviderCapabilities {
	return ProviderCapabilities{
		Provider:                  provider,
		Backend:                   backend,
		CanDetectRunningSession:   CapabilityUnknown,
		CanResumeSession:          CapabilityUnknown,
		CanQueryThreadStatus:      CapabilityUnknown,
		CanShowAppVisibleLink:     CapabilityUnknown,
		CanAcceptOperatorInput:    CapabilityUnknown,
		CanConfirmTerminalOutcome: CapabilityUnknown,
		Notes:                     "No verified visible launch/recovery capabilities are registered for this provider/backend pair.",
	}
}

// UnsupportedReuse is the session-reuse answer for a backend with no known
// resume mechanism.
func UnsupportedReuse(backend string, reason string) SessionReuseCapability {
	return SessionReuseCapability{
		Backend:    backend,
		Level:      SessionReuseUnsupported,
		CanAttempt: false,
		Reason:     reason,
	}
}
