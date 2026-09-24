package tasklifecycle

import (
	"strings"
	"testing"
)

func TestProviderErrorCommentIncludesIdentifiersAndLogPath(t *testing.T) {
	comment := ProviderErrorComment(&ProviderError{
		Provider:        "claude",
		Kind:            "auth_required",
		Reason:          "Claude login required. Sign in to Claude on the daemon host, then retry this task.",
		SuggestedAction: "Sign in to Claude on the daemon host, then move this task to todo/needs_rework.",
		RetryPolicy:     RetryPolicyManual,
		Detail:          "login required",
	}, ProviderErrorContext{
		Agent:   "claude",
		ClaimID: "claim-1",
		LogPath: "/tmp/session.log",
	})
	for _, want := range []string{
		"Claude login required",
		"auth_required",
		"Sign in to Claude",
		"manual",
		"claim `claim-1`",
		"/tmp/session.log",
		"needs_rework",
		"needs_review",
	} {
		if !strings.Contains(comment, want) {
			t.Fatalf("comment missing %q:\n%s", want, comment)
		}
	}
	if strings.Contains(comment, "Provider detail:") {
		t.Fatalf("comment should not dump raw provider detail as primary text:\n%s", comment)
	}
}

func TestProviderErrorCommentEmptyWhenNoError(t *testing.T) {
	if got := ProviderErrorComment(nil, ProviderErrorContext{}); got != "" {
		t.Fatalf("comment = %q, want empty", got)
	}
}

func TestOperatorReleaseDefaultTaskStatusPerAction(t *testing.T) {
	cases := map[string]string{
		"mark_dead":                       "blocked",
		"mark_provider_unavailable":       "blocked",
		"cancel_without_provider_control": "needs_rework",
		"release":                         "needs_review",
	}
	for action, want := range cases {
		if got := OperatorReleaseDefaultTaskStatus(action); got != want {
			t.Fatalf("OperatorReleaseDefaultTaskStatus(%q) = %q, want %q", action, got, want)
		}
	}
}

func TestAllowedOperatorReleaseTaskStatus(t *testing.T) {
	for _, status := range []string{"needs_review", "needs_rework", "blocked"} {
		if !AllowedOperatorReleaseTaskStatus(status) {
			t.Fatalf("%q should be allowed", status)
		}
	}
	for _, status := range []string{"doing", "todo", "done", "archived", "backlog"} {
		if AllowedOperatorReleaseTaskStatus(status) {
			t.Fatalf("%q should not be allowed", status)
		}
	}
}

func TestOperatorReleaseCommentIncludesExecutionStatusAndTarget(t *testing.T) {
	comment := OperatorReleaseComment("mark_dead", "claim-2", "dead", "/tmp/x.log", "blocked")
	for _, want := range []string{"claim-2", "dead", "/tmp/x.log", "blocked"} {
		if !strings.Contains(comment, want) {
			t.Fatalf("comment missing %q:\n%s", want, comment)
		}
	}
}

func TestActiveExecutionGuardCommentReferencesClaim(t *testing.T) {
	comment := ActiveExecutionGuardComment("claim-3")
	if !strings.Contains(comment, "claim-3") || !strings.Contains(comment, "doing") {
		t.Fatalf("comment = %q", comment)
	}
}

func TestLaunchNotReadyAndProcessFailedComments(t *testing.T) {
	if got := LaunchNotReadyComment("agent unavailable"); !strings.Contains(got, "agent unavailable") {
		t.Fatalf("comment = %q", got)
	}
	if got := LaunchProcessFailedComment("", "/tmp/y.log"); !strings.Contains(got, "unknown process failure") || !strings.Contains(got, "/tmp/y.log") {
		t.Fatalf("comment = %q", got)
	}
	if got := LaunchStartFailedComment("boom"); got != "Launch failed: boom" {
		t.Fatalf("comment = %q", got)
	}
}
