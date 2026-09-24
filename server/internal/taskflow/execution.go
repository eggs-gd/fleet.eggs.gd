package taskflow

import (
	"encoding/json"
	"strings"
)

// NormalizeExecutionOutcome maps a raw worker/runtime outcome string onto the
// canonical ExecutionOutcome vocabulary. Legacy CORE-97 `waiting_input`
// maps to needs_input. Unknown values return ok=false.
func NormalizeExecutionOutcome(raw string) (ExecutionOutcome, bool) {
	switch strings.TrimSpace(raw) {
	case "completed":
		return ExecutionCompleted, true
	case "failed":
		return ExecutionFailed, true
	case "needs_input", "waiting_input":
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

// legacyWorkerResult is the CORE-97 JSON shape workers still emit during
// migration. ParseExecutionResult accepts both this and the canonical
// ExecutionResult field names.
type legacyWorkerResult struct {
	TaskID              string   `json:"task_id,omitempty"`
	Outcome             string   `json:"outcome"`
	Summary             string   `json:"summary,omitempty"`
	Tests               []string `json:"tests,omitempty"`
	Artifacts           []string `json:"artifacts,omitempty"`
	Question            string   `json:"question,omitempty"`
	Error               string   `json:"error,omitempty"`
	Blockers            []string `json:"blockers,omitempty"`
	SuggestedNextStatus string   `json:"suggested_next_status,omitempty"`
	ReviewNotes         string   `json:"review_notes,omitempty"`
}

// ParseExecutionResult scans free-form worker output for the compact JSON
// result payload and returns the last recognized candidate. Accepts both
// canonical ExecutionResult fields and legacy CORE-97 WorkerResult fields.
// The example placeholder outcome embedded in launch prompts is never a
// recognized value, so echoed prompts cannot false-trigger finalization.
func ParseExecutionResult(text string) *ExecutionResult {
	var found *ExecutionResult
	for _, candidate := range jsonObjectCandidates(text) {
		var raw legacyWorkerResult
		if err := json.Unmarshal([]byte(candidate), &raw); err != nil {
			continue
		}
		outcome, ok := NormalizeExecutionOutcome(raw.Outcome)
		if !ok {
			continue
		}
		result := ExecutionResult{
			TaskID:    strings.TrimSpace(raw.TaskID),
			Outcome:   outcome,
			Summary:   strings.TrimSpace(raw.Summary),
			Tests:     append([]string{}, raw.Tests...),
			Artifacts: append([]string{}, raw.Artifacts...),
			Question:  strings.TrimSpace(raw.Question),
			Error:     strings.TrimSpace(raw.Error),
		}
		// Fold legacy informational fields into comment-facing text. Workers
		// never apply SuggestedNextStatus; Finalizer keeps it informational.
		if result.Question == "" && len(raw.Blockers) > 0 {
			result.Question = joinNonEmpty(raw.Blockers)
		}
		if notes := strings.TrimSpace(raw.ReviewNotes); notes != "" {
			if result.Summary == "" {
				result.Summary = notes
			} else {
				result.Summary += "\n\nReview notes: " + notes
			}
		}
		if hint := strings.TrimSpace(raw.SuggestedNextStatus); hint != "" {
			result.Summary = strings.TrimSpace(result.Summary + "\n\nWorker-suggested next status: `" + hint +
				"` (informational only; Core's lifecycle layer decides the actual transition).")
		}
		copied := result
		found = &copied
	}
	return found
}

func joinNonEmpty(values []string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		parts = append(parts, value)
	}
	return strings.Join(parts, "; ")
}

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
