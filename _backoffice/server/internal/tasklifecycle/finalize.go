package tasklifecycle

import (
	"encoding/json"
	"strings"
)

// LegacyExecutionStatus maps an older process-level status
// (exited/failed/cancelled/running/...) plus an optional error message onto
// the canonical execution_status vocabulary a task finalization decision
// switches on (succeeded/failed/timed_out/cancelled/provider_error/...).
func LegacyExecutionStatus(status string, errorMessage string) string {
	switch status {
	case "provider_error":
		return "provider_error"
	case "exited":
		if strings.TrimSpace(errorMessage) == "" {
			return "succeeded"
		}
		return "failed"
	case "failed":
		lower := strings.ToLower(errorMessage)
		if strings.Contains(lower, "timeout") || strings.Contains(lower, "timed out") {
			return "timed_out"
		}
		return "failed"
	case "cancelled":
		return "cancelled"
	case "running", "starting", "waiting_for_visible_session", "waiting_input":
		return status
	default:
		return firstNonEmpty(status, "failed")
	}
}

// FinalizeSucceededExecution decides the target task status and operator
// comment for an execution whose status resolved to "succeeded", based on
// the worker's reported result. A nil result is treated as an implicit
// "completed" outcome (a worker that exited cleanly without ever reporting a
// result payload).
//
// CORE-105: daemon finalization now prefers taskflow.ExecutionResult +
// TaskService.ReportExecution. This helper remains for domain-level mapping
// tests and any callers that still need the status/comment pair without a
// TaskService handle. Workers report Outcome; they never pick the task
// status themselves, and SuggestedNextStatus (if present) is folded into the
// comment as a hint for the human reviewer, not applied directly.
func FinalizeSucceededExecution(result *WorkerResult, logPath string) (status string, comment string) {
	if result == nil {
		result = &WorkerResult{Outcome: "completed", Summary: "Agent execution completed successfully."}
	}
	switch strings.TrimSpace(result.Outcome) {
	case "completed":
		return "needs_review", composeFinalizationComment("Agent execution completed.", result, logPath)
	case "blocked":
		return "blocked", composeFinalizationComment("Agent execution reported blocked.", result, logPath)
	case "needs_rework":
		return "needs_rework", composeFinalizationComment("Agent execution reported the work needs rework before it is done.", result, logPath)
	case "failed":
		return "blocked", composeFinalizationComment("Agent execution reported failure.", result, logPath)
	case "waiting_input", "needs_input":
		return "blocked", composeFinalizationComment("Agent execution reported it is waiting on operator input.", result, logPath)
	default:
		comment = "Agent execution succeeded but returned invalid result outcome `" + strings.TrimSpace(result.Outcome) + "`.\n\nSession log: `" + logPath + "`"
		return "blocked", comment
	}
}

func composeFinalizationComment(headline string, result *WorkerResult, logPath string) string {
	comment := headline
	if result.Summary != "" {
		comment += "\n\n" + result.Summary
	}
	if result.Question != "" {
		comment += "\n\nQuestion: " + result.Question
	}
	if len(result.Blockers) > 0 {
		comment += "\n\nBlockers:"
		for _, blocker := range result.Blockers {
			blocker = strings.TrimSpace(blocker)
			if blocker == "" {
				continue
			}
			comment += "\n- " + blocker
		}
	}
	if result.ReviewNotes != "" {
		comment += "\n\nReview notes: " + result.ReviewNotes
	}
	if result.SuggestedNextStatus != "" {
		comment += "\n\nWorker-suggested next status: `" + strings.TrimSpace(result.SuggestedNextStatus) +
			"` (informational only; Core's lifecycle layer decides the actual transition)."
	}
	if len(result.Artifacts) > 0 {
		comment += "\n\nArtifacts:"
		for _, artifact := range result.Artifacts {
			artifact = strings.TrimSpace(artifact)
			if artifact == "" {
				continue
			}
			comment += "\n- `" + artifact + "`"
		}
	}
	if len(result.Tests) > 0 {
		comment += "\n\nTests/checks:"
		for _, test := range result.Tests {
			test = strings.TrimSpace(test)
			if test == "" {
				continue
			}
			comment += "\n- `" + test + "`"
		}
	}
	if logPath != "" {
		comment += "\n\nSession log: `" + logPath + "`"
	}
	return comment
}

// ParseWorkerResult scans free-form worker output (a chat transcript, a CLI
// stdout capture, a JSON-RPC message's text) for the compact JSON result
// payload documented in Fleet/*.md and _docs/AGENT_LAUNCHER.md
// ({"outcome":"completed", ...}) and returns the last recognized one found.
//
// This is deliberately conservative: it only accepts a candidate JSON object
// whose "outcome" field is one of WorkerOutcomes (see IsValidWorkerOutcome),
// and if several candidates match it returns the last one. Two properties of
// daemon-launched sessions make that necessary rather than a nice-to-have:
//
//   - Some providers (confirmed for Claude's `claude logs`, now handled by the
//     live runner implementations in `internal/execution/providers`) echo the full launch
//     prompt — which itself shows the worker the exact JSON shape to use as
//     an example — back into the same transcript this function scans. An
//     outcome allowlist means that example can never look like a real result
//     on its own, regardless of how it is worded in the prompt.
//   - A worker's own prose can legitimately contain other brace-delimited
//     text (code blocks, diffs). "Last match wins" plus the outcome
//     allowlist means only a real, later, on-protocol payload overrides an
//     earlier one; unrelated braces essentially never parse into a
//     WorkerResult with a recognized outcome.
func ParseWorkerResult(text string) *WorkerResult {
	var found *WorkerResult
	for _, candidate := range jsonObjectCandidates(text) {
		var result WorkerResult
		if err := json.Unmarshal([]byte(candidate), &result); err != nil {
			continue
		}
		if !IsValidWorkerOutcome(result.Outcome) {
			continue
		}
		result.Outcome = strings.TrimSpace(result.Outcome)
		copied := result
		found = &copied
	}
	return found
}

// jsonObjectCandidates scans text for balanced top-level {...} substrings,
// tracking quoted-string/escape state so braces inside string values don't
// skew the depth count. It over-approximates on purpose (arbitrary prose can
// contain balanced braces that are not JSON at all); callers must
// independently validate parsed content the way ParseWorkerResult does.
func jsonObjectCandidates(text string) []string {
	var candidates []string
	depth := 0
	start := -1
	inString := false
	escaped := false
	for i, r := range text {
		if inString {
			switch {
			case escaped:
				escaped = false
			case r == '\\':
				escaped = true
			case r == '"':
				inString = false
			}
			continue
		}
		switch r {
		case '"':
			inString = true
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			if depth > 0 {
				depth--
				if depth == 0 && start >= 0 {
					candidates = append(candidates, text[start:i+1])
					start = -1
				}
			}
		}
	}
	return candidates
}
