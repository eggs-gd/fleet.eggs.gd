package tasklifecycle

import (
	"strings"
	"testing"
)

func TestWorkerResultNeedsInput(t *testing.T) {
	if WorkerResultNeedsInput(nil) {
		t.Fatal("nil result is not HITL")
	}
	if !WorkerResultNeedsInput(&WorkerResult{Outcome: "needs_input"}) {
		t.Fatal("needs_input is HITL")
	}
	if !WorkerResultNeedsInput(&WorkerResult{Outcome: "waiting_input"}) {
		t.Fatal("waiting_input is HITL")
	}
	if WorkerResultNeedsInput(&WorkerResult{Outcome: "blocked"}) {
		t.Fatal("blocked closes the task; it is not a HITL pause")
	}
	if WorkerResultNeedsInput(&WorkerResult{Outcome: "completed"}) {
		t.Fatal("completed is not HITL")
	}
}

func TestLegacyExecutionStatusMapsProcessStatuses(t *testing.T) {
	cases := []struct {
		status       string
		errorMessage string
		want         string
	}{
		{"provider_error", "anything", "provider_error"},
		{"exited", "", "succeeded"},
		{"exited", "boom", "failed"},
		{"failed", "connection timed out", "timed_out"},
		{"failed", "Timeout waiting for response", "timed_out"},
		{"failed", "boom", "failed"},
		{"cancelled", "", "cancelled"},
		{"running", "", "running"},
		{"starting", "", "starting"},
		{"waiting_for_visible_session", "", "waiting_for_visible_session"},
		{"waiting_input", "", "waiting_input"},
		{"", "", "failed"},
	}
	for _, tc := range cases {
		got := LegacyExecutionStatus(tc.status, tc.errorMessage)
		if got != tc.want {
			t.Fatalf("LegacyExecutionStatus(%q, %q) = %q, want %q", tc.status, tc.errorMessage, got, tc.want)
		}
	}
}

func TestFinalizeSucceededExecutionCompletedResult(t *testing.T) {
	status, comment := FinalizeSucceededExecution(&WorkerResult{Outcome: "completed", Summary: "Shipped it."}, "/tmp/log.txt")
	if status != "needs_review" {
		t.Fatalf("status = %q, want needs_review", status)
	}
	if !strings.Contains(comment, "Shipped it.") || !strings.Contains(comment, "/tmp/log.txt") {
		t.Fatalf("comment = %q", comment)
	}
}

func TestFinalizeSucceededExecutionNilResultDefaultsToCompleted(t *testing.T) {
	status, comment := FinalizeSucceededExecution(nil, "")
	if status != "needs_review" {
		t.Fatalf("status = %q, want needs_review", status)
	}
	if !strings.Contains(comment, "Agent execution completed") {
		t.Fatalf("comment = %q", comment)
	}
}

func TestFinalizeSucceededExecutionBlockedResult(t *testing.T) {
	status, comment := FinalizeSucceededExecution(&WorkerResult{Outcome: "blocked", Summary: "Need approval."}, "")
	if status != "blocked" {
		t.Fatalf("status = %q, want blocked", status)
	}
	if !strings.Contains(comment, "Need approval.") {
		t.Fatalf("comment = %q", comment)
	}
}

func TestFinalizeSucceededExecutionInvalidOutcomeBlocks(t *testing.T) {
	status, comment := FinalizeSucceededExecution(&WorkerResult{Outcome: "something_else"}, "/tmp/log.txt")
	if status != "blocked" {
		t.Fatalf("status = %q, want blocked", status)
	}
	if !strings.Contains(comment, "invalid result outcome") || !strings.Contains(comment, "something_else") {
		t.Fatalf("comment = %q", comment)
	}
}

func TestFinalizeSucceededExecutionNeedsReworkResult(t *testing.T) {
	status, comment := FinalizeSucceededExecution(&WorkerResult{Outcome: "needs_rework", Summary: "Half done."}, "")
	if status != "needs_rework" {
		t.Fatalf("status = %q, want needs_rework", status)
	}
	if !strings.Contains(comment, "Half done.") || !strings.Contains(comment, "needs rework") {
		t.Fatalf("comment = %q", comment)
	}
}

func TestFinalizeSucceededExecutionFailedResultMapsToBlocked(t *testing.T) {
	status, comment := FinalizeSucceededExecution(&WorkerResult{Outcome: "failed", Summary: "Could not reproduce."}, "")
	if status != "blocked" {
		t.Fatalf("status = %q, want blocked", status)
	}
	if !strings.Contains(comment, "reported failure") {
		t.Fatalf("comment = %q", comment)
	}
}

func TestFinalizeSucceededExecutionWaitingInputMapsToBlocked(t *testing.T) {
	status, comment := FinalizeSucceededExecution(&WorkerResult{Outcome: "waiting_input", Summary: "Need a decision."}, "")
	if status != "blocked" {
		t.Fatalf("status = %q, want blocked", status)
	}
	if !strings.Contains(comment, "waiting on operator input") {
		t.Fatalf("comment = %q", comment)
	}
}

func TestFinalizeSucceededExecutionComposesBlockersReviewNotesAndSuggestion(t *testing.T) {
	status, comment := FinalizeSucceededExecution(&WorkerResult{
		Outcome:             "blocked",
		Summary:             "Missing access.",
		Blockers:            []string{"need staging credentials", "need Alex to confirm scope"},
		ReviewNotes:         "Checked both providers first.",
		SuggestedNextStatus: "todo",
	}, "")
	if status != "blocked" {
		t.Fatalf("status = %q, want blocked", status)
	}
	for _, want := range []string{
		"need staging credentials",
		"need Alex to confirm scope",
		"Checked both providers first.",
		"Worker-suggested next status: `todo`",
		"informational only",
	} {
		if !strings.Contains(comment, want) {
			t.Fatalf("comment missing %q:\n%s", want, comment)
		}
	}
}

func TestParseWorkerResultExtractsCompactJSONPayload(t *testing.T) {
	result := ParseWorkerResult(`Done.

{"outcome":"completed","summary":"implemented","artifacts":["a.go"],"tests":["go test ./..."]}`)
	if result == nil {
		t.Fatal("result was not parsed")
	}
	if result.Outcome != "completed" || result.Summary != "implemented" {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0] != "a.go" {
		t.Fatalf("artifacts = %#v", result.Artifacts)
	}
}

func TestParseWorkerResultReturnsNilForEmptyOrUnrecognizedText(t *testing.T) {
	if got := ParseWorkerResult(""); got != nil {
		t.Fatalf("empty text = %#v, want nil", got)
	}
	if got := ParseWorkerResult("just talking, no JSON here"); got != nil {
		t.Fatalf("prose-only text = %#v, want nil", got)
	}
	if got := ParseWorkerResult(`{"summary":"no outcome field"}`); got != nil {
		t.Fatalf("missing-outcome text = %#v, want nil", got)
	}
}

// This is the CORE-97 false-positive guard: the launch prompt shows workers
// the exact JSON shape to report, and at least one provider (Claude's
// `claude logs`) echoes that prompt back into the same transcript
// ParseWorkerResult scans. A placeholder outcome value (as the prompt uses,
// see execution.BuildPrompt) must never be mistaken for a real result, and a
// later real result must still win over it.
func TestParseWorkerResultIgnoresEchoedPromptPlaceholderButFindsRealResultAfterIt(t *testing.T) {
	transcript := `You are a Core worker agent launched by the deterministic Core daemon.
...
- Still finish with a compact JSON result payload once real work is done:
  {"outcome":"<completed|blocked|needs_rework|failed|waiting_input>","summary":"...","artifacts":["..."],"tests":["..."]}

I read the task card and made the change.

{"outcome":"completed","summary":"Wired the outcome protocol.","artifacts":["internal/tasklifecycle/finalize.go"]}`

	result := ParseWorkerResult(transcript)
	if result == nil {
		t.Fatal("expected the real result to be found")
	}
	if result.Outcome != "completed" || result.Summary != "Wired the outcome protocol." {
		t.Fatalf("result = %#v, want the real (later) payload, not the echoed placeholder", result)
	}
}

func TestParseWorkerResultIgnoresPlaceholderOnlyTranscript(t *testing.T) {
	transcript := `Still finish with a compact JSON result payload once real work is done:
  {"outcome":"<completed|blocked|needs_rework|failed|waiting_input>","summary":"...","artifacts":["..."],"tests":["..."]}

Still working on it...`

	if result := ParseWorkerResult(transcript); result != nil {
		t.Fatalf("result = %#v, want nil: only the placeholder example is present, no real report yet", result)
	}
}

func TestParseWorkerResultPicksLastValidCandidateWhenMultiplePresent(t *testing.T) {
	transcript := `{"outcome":"blocked","summary":"first attempt hit a wall"}

Actually, resolved it.

{"outcome":"completed","summary":"done after all"}`

	result := ParseWorkerResult(transcript)
	if result == nil || result.Outcome != "completed" || result.Summary != "done after all" {
		t.Fatalf("result = %#v, want the last reported outcome to win", result)
	}
}
