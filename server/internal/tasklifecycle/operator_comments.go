package tasklifecycle

import "strings"

// ProviderErrorContext carries the session identifiers a task-facing
// provider-error comment can reference, without requiring the caller
// (corechain) to hand this package its RuntimeSession type. Every field is
// optional; empty ones are simply omitted from the rendered comment.
type ProviderErrorContext struct {
	Agent            string
	ClaimID          string
	BackgroundID     string
	ThreadID         string
	TurnID           string
	CursorChatID     string
	RemoteControlURL string
	LogPath          string
}

// ProviderErrorComment renders the task-facing blocked-task comment for a
// classified provider error. It leads with the concise actionable reason and
// keeps raw provider noise out of the primary comment body.
func ProviderErrorComment(pe *ProviderError, ctx ProviderErrorContext) string {
	if pe == nil {
		return ""
	}
	provider := firstNonEmpty(pe.Provider, ctx.Agent)
	comment := pe.Reason
	if comment == "" {
		comment = "Provider blocker detected for `" + provider + "`."
	}
	if pe.Kind != "" {
		comment += "\n\nDetected reason: `" + pe.Kind + "`."
	}
	if pe.SuggestedAction != "" {
		comment += "\nSuggested action: " + pe.SuggestedAction
	}
	if pe.RetryPolicy != "" {
		comment += "\nRetry policy: `" + pe.RetryPolicy + "` (no automatic retry until an operator changes something)."
	}
	ids := compactIdentifiers(
		identifierLine("claim", ctx.ClaimID),
		identifierLine("background", ctx.BackgroundID),
		identifierLine("thread", ctx.ThreadID),
		identifierLine("turn", ctx.TurnID),
		identifierLine("cursor_chat", ctx.CursorChatID),
		identifierLine("session", ctx.RemoteControlURL),
	)
	if len(ids) > 0 {
		comment += "\n\nSession identifiers: " + strings.Join(ids, ", ") + "."
	}
	if ctx.LogPath != "" {
		comment += "\n\nSession log: `" + ctx.LogPath + "`"
	}
	comment += "\n\nMove this task to `needs_rework` or `todo` after the fix for an operator-approved retry, or to `needs_review` if the operator has already verified the artifacts."
	return comment
}

func identifierLine(label string, value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return label + " `" + strings.TrimSpace(value) + "`"
}

func compactIdentifiers(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

// GenericProviderBlockedComment is the fallback comment for a provider
// failure that ClassifyProviderError could not classify into a
// ProviderError.
func GenericProviderBlockedComment(errorMessage string, logPath string) string {
	comment := "Provider blocker detected: " + firstNonEmpty(errorMessage, "unknown provider failure")
	if logPath != "" {
		comment += "\n\nSession log: `" + logPath + "`"
	}
	return comment
}

// LaunchStartFailedComment is the task-facing comment for a launch that
// failed before a process/session ever started.
func LaunchStartFailedComment(errorMessage string) string {
	return "Launch failed: " + errorMessage
}

// LaunchProcessFailedComment is the task-facing comment for a launched
// process/session that failed after starting.
func LaunchProcessFailedComment(errorMessage string, logPath string) string {
	comment := "Launch process failed: " + firstNonEmpty(errorMessage, "unknown process failure")
	if logPath != "" {
		comment += "\n\nSession log: `" + logPath + "`"
	}
	return comment
}

// OperatorAttentionComment is the task-facing comment recorded when a live
// execution needs operator attention (waiting on input, idle past threshold,
// stalled) without being blocked.
func OperatorAttentionComment(reason string, logPath string) string {
	comment := firstNonEmpty(reason, "Agent execution needs operator attention.")
	if logPath != "" {
		comment += "\n\nSession log: `" + logPath + "`"
	}
	return comment
}

// LaunchNotReadyComment is the task-facing comment for a pickup-status task
// the launcher decided not to start yet (a launch-log classification
// reason, not a hard failure).
func LaunchNotReadyComment(reason string) string {
	return "Launch is not ready: " + reason +
		"\n\nMove this task to `needs_rework` after fixing the blocker or choosing the next launch strategy."
}

// ActiveExecutionGuardComment is the task-facing comment recorded when the
// runtime reverts a task back to `doing` because a live execution has not
// reached a terminal state yet. This encodes the rule that worker agents do
// not directly finalize task lifecycle while an execution is active — only
// the runtime/orchestrator does, once the execution succeeds, fails, times
// out, or is cancelled.
func ActiveExecutionGuardComment(claimID string) string {
	return "Runtime kept this task in `doing` because active execution `" + claimID +
		"` has not reached a terminal state. Worker completion must be reported as a result payload; " +
		"the runtime/orchestrator finalizes the task lifecycle after execution succeeds, fails, times out, or is cancelled."
}

// OperatorReleaseReason is the short, session-identifying reason recorded
// when an operator releases a runtime session without live provider
// control.
func OperatorReleaseReason(action string, claimID string) string {
	switch action {
	case "mark_dead":
		return "Operator marked runtime execution " + claimID + " dead without provider control."
	case "mark_provider_unavailable":
		return "Operator marked runtime execution " + claimID + " provider_unavailable without provider control."
	case "cancel_without_provider_control":
		return "Operator cancelled runtime execution " + claimID + " without provider control."
	default:
		return "Operator released runtime execution " + claimID + " without provider control."
	}
}

// OperatorReleaseComment is the full task-facing comment recorded for an
// operator release.
func OperatorReleaseComment(action string, claimID string, executionStatus string, logPath string, targetStatus string) string {
	comment := OperatorReleaseReason(action, claimID)
	comment += "\n\nExecution status is now `" + executionStatus + "`."
	if logPath != "" {
		comment += "\n\nSession log: `" + logPath + "`"
	}
	if targetStatus != "" {
		comment += "\n\nTask target after release: `" + targetStatus + "`."
	}
	comment += "\n\nThis release is auditable via runtime events; further status changes use normal task transitions."
	return comment
}

// OperatorReleaseDefaultTaskStatus is the task status an operator release
// resolves to when the caller does not specify one explicitly.
func OperatorReleaseDefaultTaskStatus(action string) string {
	switch action {
	case "mark_dead", "mark_provider_unavailable":
		return "blocked"
	case "cancel_without_provider_control":
		return "needs_rework"
	default:
		return "needs_review"
	}
}

// AllowedOperatorReleaseTaskStatus reports whether status is a valid target
// for an operator release (a deliberately narrower set than every status
// AllowedTaskStatusTransition would otherwise permit).
func AllowedOperatorReleaseTaskStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "needs_review", "needs_rework", "blocked":
		return true
	default:
		return false
	}
}

// RemoteControlSessionLiveComment announces a newly phone-visible Claude
// remote-control session on the task.
func RemoteControlSessionLiveComment(url string) string {
	return "Remote-control session is live: " + url
}

// CursorSessionLiveComment announces a newly started Cursor CLI-visible
// session on the task.
func CursorSessionLiveComment(chatID string, operatorCommand string, logPath string) string {
	return "Cursor CLI-visible session is live. Chat id: `" + chatID + "`.\n\nResume/list from the daemon host with: `" + operatorCommand + "` or `cursor-agent ls`.\n\nSession log: `" + logPath + "`"
}

// StaleProviderSessionComment is the task-facing comment recorded during
// startup reconciliation when a persisted session can no longer be verified
// as live.
func StaleProviderSessionComment(pe *ProviderError, ctx ProviderErrorContext, fallbackReason string, executionStatus string) string {
	comment := "Persisted Claude background session is no longer active or inspectable: " + fallbackReason
	if pe != nil {
		comment = ProviderErrorComment(pe, ctx)
	}
	comment += "\n\nCore marked this execution `" + executionStatus + "` during startup reconciliation so this task is no longer guarded by a stale resumable session."
	comment += "\n\nThe operator can inspect the task artifacts/logs and move this task to `needs_review`, `needs_rework`, or `todo` as appropriate."
	return comment
}
