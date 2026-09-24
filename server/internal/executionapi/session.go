package executionapi

import (
	"regexp"
	"strings"

	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// ClaudeSessionDetails holds Claude background-remote identity.
type ClaudeSessionDetails struct {
	BackgroundID string `json:"background_id,omitempty"`
	// SessionID is the full session UUID from `claude agents --json`
	// (distinct from BackgroundID, which is the short id `claude stop`/`logs`
	// take). `claude --bg --resume` only continues the same session under
	// its own id when given this full id — the short id starts a copy
	// instead (empirically verified 2026-09-12, see
	// _docs/AGENT_SESSION_REUSE.md).
	SessionID        string `json:"session_id,omitempty"`
	RemoteControlURL string `json:"remote_control_url,omitempty"`
}

// CodexSessionDetails holds Codex app-server thread/turn identity.
type CodexSessionDetails struct {
	CodexThreadID    string `json:"codex_thread_id,omitempty"`
	CodexTurnID      string `json:"codex_turn_id,omitempty"`
	CodexThreadTitle string `json:"codex_thread_title,omitempty"`
}

// CursorSessionDetails holds Cursor CLI-visible chat identity.
type CursorSessionDetails struct {
	CursorChatID    string `json:"cursor_chat_id,omitempty"`
	OperatorCommand string `json:"operator_command,omitempty"`
}

// GeminiSessionDetails holds Antigravity CLI (`agy`) conversation/project
// identity. GeminiProjectID is the antigravity project registered for the
// task's working directory (`agy --new-project`, discovered afterward from
// `~/.gemini/config/projects/*.json` since the CLI does not print it) —
// required for `agy` to auto-load AGENTS.md/GEMINI.md and project-scoped MCP
// config from that directory. GeminiConversationID is the per-turn identity
// resumed via `agy --conversation <id>`.
type GeminiSessionDetails struct {
	GeminiProjectID      string `json:"gemini_project_id,omitempty"`
	GeminiConversationID string `json:"gemini_conversation_id,omitempty"`
}

type RuntimeSession struct {
	ClaimID        string   `json:"claim_id"`
	TaskRef        string   `json:"task_ref"`
	TaskID         string   `json:"task_id"`
	TaskPath       string   `json:"task_path"`
	TaskTitle      string   `json:"task_title,omitempty"`
	ProjectID      string   `json:"project_id"`
	Repository     string   `json:"repository"`
	Agent          string   `json:"agent"`
	Launcher       string   `json:"launcher"`
	Backend        string   `json:"backend,omitempty"`
	VisibilityMode string   `json:"visibility_mode,omitempty"`
	HostID         string   `json:"host_id,omitempty"`
	HostName       string   `json:"host_name,omitempty"`
	Command        []string `json:"command"`
	LogPath        string   `json:"log_path,omitempty"`
	WorkingDir     string   `json:"working_dir"`
	ProcessID      int      `json:"process_id,omitempty"`
	ClaudeSessionDetails
	CodexSessionDetails
	CursorSessionDetails
	GeminiSessionDetails
	LastEvent            string                       `json:"last_event,omitempty"`
	LastMessage          string                       `json:"last_message,omitempty"`
	ClaimedAt            string                       `json:"claimed_at"`
	StartedAt            string                       `json:"started_at,omitempty"`
	LastSeenAt           string                       `json:"last_seen_at,omitempty"`
	LastEventAt          string                       `json:"last_event_at,omitempty"`
	LastOutputAt         string                       `json:"last_output_at,omitempty"`
	LastStatusAt         string                       `json:"last_status_change_at,omitempty"`
	ExitedAt             string                       `json:"exited_at,omitempty"`
	Status               string                       `json:"status"`
	ExecutionStatus      string                       `json:"execution_status,omitempty"`
	Result               *tasklifecycle.WorkerResult  `json:"result,omitempty"`
	ExitCode             int                          `json:"exit_code,omitempty"`
	ErrorMessage         string                       `json:"error_message,omitempty"`
	BlockingReason       string                       `json:"blocking_reason,omitempty"`
	ProviderError        *tasklifecycle.ProviderError `json:"provider_error,omitempty"`
	Capabilities         ProviderCapabilities         `json:"capabilities,omitempty"`
	SupersedesClaimID    string                       `json:"supersedes_claim_id,omitempty"`
	SupersededByClaimID  string                       `json:"superseded_by_claim_id,omitempty"`
	ReuseCapability      string                       `json:"reuse_capability,omitempty"`
	ResumeAttempted      bool                         `json:"resume_attempted,omitempty"`
	ResumeOutcome        string                       `json:"resume_outcome,omitempty"`
	ProviderControllable bool                         `json:"provider_controllable,omitempty"`
	// ToolUsage is compact per-session MCP/tool evidence (CORE-120).
	// Required/used/missing only — never full transcripts.
	ToolUsage *ToolUsageEvidence `json:"tool_usage,omitempty"`
}

type SessionControlRequest struct {
	Action string
	Input  string
	Done   chan error
}

func (session RuntimeSession) IsActive() bool {
	switch firstNonEmpty(session.ExecutionStatus, session.Status) {
	case "queued", "claimed", "starting", "waiting_for_visible_session", "running", "waiting_input", "operator_attention", "stalled", "resumable":
		return true
	default:
		return false
	}
}

// IsProviderStartup reports whether the session still owns a launch slot while
// the provider handshake (dispatch / create-chat / thread start / remote-control
// URL registration) has not finished.
func (session RuntimeSession) IsProviderStartup() bool {
	switch firstNonEmpty(session.ExecutionStatus, session.Status) {
	case "queued", "claimed", "starting", "waiting_for_visible_session":
		return true
	default:
		return false
	}
}

func (session RuntimeSession) ProviderThreadID() string {
	return firstNonEmpty(session.CodexThreadID, session.CursorChatID, session.BackgroundID)
}

func (session RuntimeSession) ProviderSessionID() string {
	return firstNonEmpty(session.RemoteControlURL, session.CursorChatID, session.BackgroundID, session.CodexThreadID)
}

var backgroundRemoteControlURLPattern = regexp.MustCompile(`Continue here, on your phone, or at (https://claude\.ai/code/session_[A-Za-z0-9]+)`)
var cursorChatIDPattern = regexp.MustCompile(`[A-Za-z0-9][A-Za-z0-9_-]{7,}`)

// ParseRemoteControlURL extracts the most recent live-status URL from raw
// `claude logs` output.
func ParseRemoteControlURL(logOutput string) string {
	matches := backgroundRemoteControlURLPattern.FindAllStringSubmatch(logOutput, -1)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1][1]
}

// ParseCursorChatID extracts the chat id from `cursor-agent create-chat`.
func ParseCursorChatID(output string) string {
	for _, field := range strings.Fields(output) {
		field = strings.Trim(field, "`'\" ,:;()[]{}")
		if cursorChatIDPattern.MatchString(field) {
			return field
		}
	}
	return ""
}

// CursorOperatorCommand returns the operator-visible resume command.
func CursorOperatorCommand(binary string, chatID string, workingDir string) string {
	return strings.Join([]string{binary, "--resume", chatID, "--workspace", workingDir}, " ")
}

// CodexRemoteThreadTitle names the provider-visible thread after the Core task.
func CodexRemoteThreadTitle(ref string, title string, id string) string {
	ref = strings.TrimSpace(ref)
	title = strings.TrimSpace(title)
	switch {
	case ref != "" && title != "":
		return ref + " · " + title
	case ref != "":
		return ref
	case title != "":
		return title
	default:
		return strings.TrimSpace(id)
	}
}
