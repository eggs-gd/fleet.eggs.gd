package taskflow

import "strings"

// NormalizeExecutionOutcome maps a raw worker/runtime outcome string onto the
// canonical ExecutionOutcome vocabulary. Unknown values return ok=false.
func NormalizeExecutionOutcome(raw string) (ExecutionOutcome, bool) {
	switch strings.TrimSpace(raw) {
	case "completed":
		return ExecutionCompleted, true
	case "failed":
		return ExecutionFailed, true
	case "needs_input":
		return ExecutionNeedsInput, true
	case "needs_rework":
		return ExecutionNeedsRework, true
	case "blocked":
		return ExecutionBlocked, true
	case "cancelled":
		return ExecutionCancelled, true
	case "timed_out":
		return ExecutionTimedOut, true
	case "orphaned":
		return ExecutionOrphaned, true
	default:
		return "", false
	}
}
