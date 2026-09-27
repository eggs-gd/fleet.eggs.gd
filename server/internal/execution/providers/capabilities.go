package providers

import "github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"

type (
	SessionReuseCapability = executionapi.SessionReuseCapability
)

// Capabilities and SessionReuse belong to each adapter, next to the command it
// builds. ProviderCapabilityFlags and ProviderSessionReuseCapability look an
// adapter up by name or backend, and Settings shows the same answers.

func (Claude) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		Provider:                  "claude",
		Backend:                   BackendBackgroundRemote,
		CanDetectRunningSession:   executionapi.CapabilityYes,
		CanResumeSession:          executionapi.CapabilityYes,
		CanQueryThreadStatus:      executionapi.CapabilityYes,
		CanShowAppVisibleLink:     executionapi.CapabilityYes,
		CanAcceptOperatorInput:    executionapi.CapabilityYes,
		CanConfirmTerminalOutcome: executionapi.CapabilityYes,
		CanStartVisibleSession:    true,
		CanListSessions:           true,
		CanExposeOperatorPath:     true,
		CanDetectTerminalState:    true,
		OperatorVisibility:        string(executionapi.VisibilityAppVisible),
		TerminalStateDetection:    "`claude agents --json` polling",
		Notes:                     "`--bg --remote-control` gives a link to the session in the Claude app on the phone or the desktop. To resume, the session must be stopped first (`claude stop <id>`), then `claude --bg --resume <full-session-id> \"<prompt>\"` continues it under the same id and link with no other flags. The short id from `claude agents`, `stop` and `logs` is not enough: it forks a copy.",
	}
}

func (Claude) SessionReuse() SessionReuseCapability {
	return SessionReuseCapability{
		Backend:    BackendBackgroundRemote,
		Level:      executionapi.SessionReuseVerified,
		CanAttempt: true,
		Reason:     "Confirmed live: `claude --bg --resume <full-session-id> \"<prompt>\"`, issued after `claude stop <id>` on the prior session and with no other flags, continues the same session under the same id and link. It is a fresh CLI invocation carrying the prompt as an argument, not input written into a running process.",
	}
}

func (Codex) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		Provider:                  "codex",
		Backend:                   BackendCodexAppServer,
		CanDetectRunningSession:   executionapi.CapabilityYes,
		CanResumeSession:          executionapi.CapabilityYes,
		CanQueryThreadStatus:      executionapi.CapabilityYes,
		CanShowAppVisibleLink:     executionapi.CapabilityNo,
		CanAcceptOperatorInput:    executionapi.CapabilityYes,
		CanConfirmTerminalOutcome: executionapi.CapabilityYes,
		CanStartVisibleSession:    true,
		CanListSessions:           true,
		CanExposeOperatorPath:     true,
		CanDetectTerminalState:    true,
		OperatorVisibility:        string(executionapi.VisibilityFleetVisible),
		TerminalStateDetection:    "app-server turn and session events",
		Notes:                     "Fleet drives the app-server over JSON-RPC and offers continue, interrupt and cancel in the dashboard. The thread appears in Codex Remote on paired clients under the Fleet title. There is no link, and it may not appear in the ChatGPT desktop history.",
	}
}

func (Codex) SessionReuse() SessionReuseCapability {
	return SessionReuseCapability{
		Backend:    BackendCodexAppServer,
		Level:      executionapi.SessionReuseUnverified,
		CanAttempt: true,
		Reason:     "The app-server documents a `thread/resume` call. Fleet tries it when a prior thread id is known and falls back to `thread/start` on any RPC error, so the attempt is safe although end-to-end reuse is not confirmed.",
	}
}

func (Cursor) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		Provider:                  "cursor",
		Backend:                   BackendCursorVisible,
		CanDetectRunningSession:   executionapi.CapabilityYes,
		CanResumeSession:          executionapi.CapabilityYes,
		CanQueryThreadStatus:      executionapi.CapabilityNo,
		CanShowAppVisibleLink:     executionapi.CapabilityNo,
		CanAcceptOperatorInput:    executionapi.CapabilityYes,
		CanConfirmTerminalOutcome: executionapi.CapabilityYes,
		CanStartVisibleSession:    true,
		CanListSessions:           true,
		CanExposeOperatorPath:     true,
		CanDetectTerminalState:    true,
		OperatorVisibility:        string(executionapi.VisibilityCLIVisible),
		TerminalStateDetection:    "process_exit",
		Notes:                     "Fleet creates a chat with `cursor-agent create-chat`, stores its id, and shows `cursor-agent --resume <chat_id> --workspace <path>` for a person to continue it. There is no link. `cursor-agent ls` lists sessions. After a restart a live process can be reattached, and a dead process with a chat id can be resumed.",
	}
}

func (Cursor) SessionReuse() SessionReuseCapability {
	return SessionReuseCapability{
		Backend:    BackendCursorVisible,
		Level:      executionapi.SessionReuseVerified,
		CanAttempt: true,
		Reason:     "The chat id is created before the first prompt. On the next task Fleet passes the stored `cursor_chat_id` to `cursor-agent --resume`.",
	}
}

func (Gemini) Capabilities() ProviderCapabilities {
	return ProviderCapabilities{
		Provider:                  "gemini",
		Backend:                   BackendGeminiHeadless,
		CanDetectRunningSession:   executionapi.CapabilityNo,
		CanResumeSession:          executionapi.CapabilityYes,
		CanQueryThreadStatus:      executionapi.CapabilityNo,
		CanShowAppVisibleLink:     executionapi.CapabilityNo,
		CanAcceptOperatorInput:    executionapi.CapabilityYes,
		CanConfirmTerminalOutcome: executionapi.CapabilityYes,
		CanStartVisibleSession:    true,
		CanListSessions:           false,
		CanExposeOperatorPath:     true,
		CanDetectTerminalState:    true,
		OperatorVisibility:        string(executionapi.VisibilityCLIVisible),
		TerminalStateDetection:    "process exit and the terminal `result` event of the stream-json output",
		Notes:                     "Each turn is one `agy --print` call that blocks until its `result` event. The conversation and project ids are stored, and `agy --project <id> --conversation <id>` reopens the conversation in a terminal between turns. There is no link and no live process to reattach after a restart, and `agy` cannot list conversations. The Antigravity remote-control daemon is not used yet.",
	}
}

func (Gemini) SessionReuse() SessionReuseCapability {
	return SessionReuseCapability{
		Backend:    BackendGeminiHeadless,
		Level:      executionapi.SessionReuseVerified,
		CanAttempt: true,
		Reason:     "Confirmed live: a second call with `--conversation <id>` keeps the same conversation id and remembers the first turn. A stale or unknown id surfaces as a normal provider error, with no fallback.",
	}
}

var capabilityRegistry = NewRegistry()

// ProviderCapabilityFlags returns the registered capabilities for a provider
// and backend pair, or "unknown" for a pair no adapter has registered.
func ProviderCapabilityFlags(provider string, backend string) executionapi.ProviderCapabilities {
	if agent, ok := capabilityRegistry.Agent(provider); ok {
		if caps := agent.Capabilities(); caps.Backend == backend {
			return caps
		}
	}
	return executionapi.UnknownCapabilities(provider, backend)
}

// ProviderSessionReuseCapability returns the session-reuse rule of the adapter
// that owns a backend.
func ProviderSessionReuseCapability(backend string) executionapi.SessionReuseCapability {
	for _, agent := range capabilityRegistry.Agents() {
		if reuse := agent.SessionReuse(); reuse.Backend == backend {
			return reuse
		}
	}
	switch backend {
	case "process", "terminal", "unsupported", "":
		return executionapi.UnsupportedReuse(backend, "No resume mechanism is known for this backend.")
	default:
		return executionapi.UnsupportedReuse(backend, "Unrecognized backend; treat as unsupported until an adapter registers it.")
	}
}
