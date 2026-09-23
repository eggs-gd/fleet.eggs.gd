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

func ProviderCapabilityFlags(provider string, backend string) ProviderCapabilities {
	switch provider {
	case "cursor":
		if backend == BackendCursorVisible {
			return ProviderCapabilities{
				Provider:                  provider,
				Backend:                   backend,
				CanDetectRunningSession:   CapabilityYes,
				CanResumeSession:          CapabilityYes,
				CanQueryThreadStatus:      CapabilityNo,
				CanShowAppVisibleLink:     CapabilityNo,
				CanAcceptOperatorInput:    CapabilityYes,
				CanConfirmTerminalOutcome: CapabilityYes,
				CanStartVisibleSession:    true,
				CanListSessions:           true,
				CanExposeOperatorPath:     true,
				CanDetectTerminalState:    true,
				OperatorVisibility:        string(VisibilityCLIVisible),
				TerminalStateDetection:    "process_exit",
				Notes:                     "Cursor is cli_visible (CORE-79/CORE-70): Core starts the interactive CLI path, stores the chat id from `cursor-agent create-chat`, and exposes `cursor-agent --resume <chat_id> --workspace <path>`. There is no Claude-style phone/app remote-control URL. `cursor-agent ls` lists provider sessions. After restart, a live process can be reattached; a dead process with a chat id is orphaned-but-resumable.",
			}
		}
	case "claude":
		if backend == BackendBackgroundRemote {
			return ProviderCapabilities{
				Provider:                  provider,
				Backend:                   backend,
				CanDetectRunningSession:   CapabilityYes,
				CanResumeSession:          CapabilityYes,
				CanQueryThreadStatus:      CapabilityYes,
				CanShowAppVisibleLink:     CapabilityYes,
				CanAcceptOperatorInput:    CapabilityYes,
				CanConfirmTerminalOutcome: CapabilityYes,
				CanStartVisibleSession:    true,
				CanListSessions:           true,
				CanExposeOperatorPath:     true,
				CanDetectTerminalState:    true,
				OperatorVisibility:        string(VisibilityAppVisible),
				TerminalStateDetection:    "`claude agents --json` polling",
				Notes:                     "Claude is app_visible (CORE-70): `--bg --remote-control` exposes a phone/app remote-control URL. Resume is verified (2026-09-12, see _docs/AGENT_SESSION_REUSE.md): the session must be stopped first (`claude stop <id>`), then `claude --bg --resume <full-session-id> \"<prompt>\"` continues it under the same id and remote-control URL with no other flags repeated. The short `id` from `claude agents`/`stop`/`logs` is not enough for resume — it forks a copy; the full `sessionId` is required.",
			}
		}
	case "gemini":
		if backend == BackendGeminiHeadless {
			return ProviderCapabilities{
				Provider:                  provider,
				Backend:                   backend,
				CanDetectRunningSession:   CapabilityNo,
				CanResumeSession:          CapabilityYes,
				CanQueryThreadStatus:      CapabilityNo,
				CanShowAppVisibleLink:     CapabilityNo,
				CanAcceptOperatorInput:    CapabilityNo,
				CanConfirmTerminalOutcome: CapabilityYes,
				CanStartVisibleSession:    false,
				CanListSessions:           false,
				CanExposeOperatorPath:     true,
				CanDetectTerminalState:    true,
				OperatorVisibility:        string(VisibilityHeadless),
				TerminalStateDetection:    "process exit + stream-json terminal `result` event status",
				Notes:                     "Antigravity CLI (`agy`) runs headless and synchronous: `agy --project <id> --print <prompt> --output-format stream-json` blocks until the turn's terminal `result` event, so there is no live process to reattach to after a Core restart — CanDetectRunningSession is No. Resume is a fresh process invocation with `--conversation <id>` (same shape as Claude's `--bg --resume`), not empirically confirmed against a real Core restart yet, so treat it as a safe-to-attempt, unverified capability. `agy` has no discovered equivalent of `claude agents --json`/`codex thread/resume` for externally querying a conversation's liveness.",
			}
		}
	case "codex":
		if backend == BackendCodexAppServer {
			return ProviderCapabilities{
				Provider:                  provider,
				Backend:                   backend,
				CanDetectRunningSession:   CapabilityYes,
				CanResumeSession:          CapabilityYes,
				CanQueryThreadStatus:      CapabilityYes,
				CanShowAppVisibleLink:     CapabilityNo,
				CanAcceptOperatorInput:    CapabilityYes,
				CanConfirmTerminalOutcome: CapabilityYes,
				CanStartVisibleSession:    true,
				CanListSessions:           true,
				CanExposeOperatorPath:     true,
				CanDetectTerminalState:    true,
				OperatorVisibility:        string(VisibilityCoreVisible),
				TerminalStateDetection:    "app-server turn/session events",
				Notes:                     "Codex app-server is core_visible (CORE-130): Core owns JSON-RPC thread/turn control plus dashboard continue/interrupt/cancel. Sessions appear in Codex Remote on paired clients with Core-set thread titles; there is no Claude-style deep-link URL and they may not appear in ChatGPT desktop sidebar history. Normal daemon auto-launch is allowed; launch.mode=allow_core_visible is a legacy no-op.",
			}
		}
	}
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

func ProviderSessionReuseCapability(backend string) SessionReuseCapability {
	switch backend {
	case BackendCursorVisible:
		return SessionReuseCapability{
			Backend:    backend,
			Level:      SessionReuseVerified,
			CanAttempt: true,
			Reason:     "Cursor visible sessions are launched against an explicit chat id created before the first prompt. On relaunch, Core can safely pass the prior `cursor_chat_id` to `cursor-agent --resume <chat_id>`; the operator-visible identity is known before the process starts.",
		}
	case BackendCodexAppServer:
		return SessionReuseCapability{
			Backend:    backend,
			Level:      SessionReuseUnverified,
			CanAttempt: true,
			Reason:     "codex app-server documents a `thread/resume` RPC (_docs/AGENT_LAUNCHER.md); Core attempts it on relaunch when a prior thread id is known and falls back to `thread/start` on any RPC error, so the attempt is safe even though end-to-end reuse has not been empirically confirmed yet.",
		}
	case BackendGeminiHeadless:
		return SessionReuseCapability{
			Backend:    backend,
			Level:      SessionReuseUnverified,
			CanAttempt: true,
			Reason:     "`agy --conversation <id>` documents resuming a prior conversation as a fresh process invocation carrying a new prompt, the same shape confirmed for Claude's `--bg --resume`. Core attempts it on relaunch when a prior conversation id is known; there is no discovered fallback RPC to detect a stale/unknown id ahead of time, so a resume attempt against an invalid id surfaces as a normal provider error rather than a graceful fallback.",
		}
	case BackendBackgroundRemote:
		return SessionReuseCapability{
			Backend:    backend,
			Level:      SessionReuseVerified,
			CanAttempt: true,
			Reason:     "Verified 2026-09-12 (_docs/AGENT_SESSION_REUSE.md): `claude --bg --resume <full-session-id> \"<prompt>\"`, issued only after `claude stop <id>` on the prior session and with no other flags repeated, continues the same session under the same id and the same remote-control URL. This is a fresh non-interactive CLI invocation carrying a new prompt as argv, not stdin injection into a live process — the CORE-53 finding (stdin bytes into a running session never reached the remote chat) does not apply here.",
		}
	case "process", "terminal", "unsupported", "":
		return SessionReuseCapability{
			Backend:    backend,
			Level:      SessionReuseUnsupported,
			CanAttempt: false,
			Reason:     "No resume mechanism is known for this backend.",
		}
	default:
		return SessionReuseCapability{
			Backend:    backend,
			Level:      SessionReuseUnsupported,
			CanAttempt: false,
			Reason:     "Unrecognized backend; treat as unsupported until a capability rule is added.",
		}
	}
}
