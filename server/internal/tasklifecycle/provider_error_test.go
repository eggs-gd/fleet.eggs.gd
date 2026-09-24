package tasklifecycle

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestClassifyProviderErrorIgnoresRateLimitProseWhenWorkerOutcomePresent(t *testing.T) {
	// CORE-96 / CORE-98 shape: successful Claude transcript that happens to
	// mention a "429 rate-limit failure mode" test case must not become a
	// provider blocker once the worker has reported a valid JSON outcome.
	transcript := `Implemented Plane provider adapter.
Added regression coverage for the 429 rate-limit failure mode.
Also documented auth/billing/quota handling in the architecture note.

{"outcome":"completed","summary":"Plane provider shipped with tests.","artifacts":["internal/taskprovider/provider.go"],"tests":["go test ./internal/taskprovider/..."]}`

	if pe := ClassifyProviderError("claude", "background-remote", "claude logs", nil, transcript); pe != nil {
		t.Fatalf("ClassifyProviderError = %#v, want nil when a completed worker outcome is present", pe)
	}
}

func TestClassifyProviderErrorStillDetectsRealRateLimitWithoutOutcome(t *testing.T) {
	text := "Error: rate limit exceeded (HTTP 429). Too many requests. Retry-After: 60 seconds"
	pe := ClassifyProviderError("claude", "background-remote", "claude logs", nil, text)
	if pe == nil || pe.Kind != "rate_limited" {
		t.Fatalf("ClassifyProviderError = %#v, want rate_limited", pe)
	}
	if pe.RetryPolicy != RetryPolicyAfterReset {
		t.Fatalf("retry_policy = %q, want %q", pe.RetryPolicy, RetryPolicyAfterReset)
	}
	if !strings.Contains(pe.Reason, "60 seconds") {
		t.Fatalf("reason = %q, want retry-after guidance", pe.Reason)
	}
}

func TestClassifyProviderErrorStillDetectsQuotaWithoutOutcome(t *testing.T) {
	text := "Quota exceeded: usage limit exceeded for this billing period."
	pe := ClassifyProviderError("claude", "background-remote", "claude logs", nil, text)
	if pe == nil || pe.Kind != "quota_exceeded" {
		t.Fatalf("ClassifyProviderError = %#v, want quota_exceeded", pe)
	}
	if pe.RetryPolicy != RetryPolicyManual {
		t.Fatalf("retry_policy = %q, want manual", pe.RetryPolicy)
	}
	if !strings.Contains(pe.Reason, "Claude quota exhausted") {
		t.Fatalf("reason = %q, want actionable Claude quota message", pe.Reason)
	}
}

func TestClassifyProviderErrorTransportFailureWinsEvenWithOutcome(t *testing.T) {
	transcript := `{"outcome":"completed","summary":"done"}`
	err := errors.New("dial unix /tmp/claude.sock: connection refused")
	pe := ClassifyProviderError("claude", "background-remote", "claude logs", err, transcript)
	if pe == nil || pe.Kind != "control_socket_unavailable" {
		t.Fatalf("ClassifyProviderError = %#v, want control_socket_unavailable", pe)
	}
}

func TestClassifyProviderErrorIgnoresTransportProseWhenWorkerOutcomePresent(t *testing.T) {
	transcript := `Documented connection refused / control socket failure modes for operators.
{"outcome":"completed","summary":"docs updated"}`
	if pe := ClassifyProviderError("claude", "background-remote", "claude logs", nil, transcript); pe != nil {
		t.Fatalf("ClassifyProviderError = %#v, want nil for successful outcome with transport prose", pe)
	}
}

func TestClassifyProviderErrorIgnoresAuthProseWhenWorkerOutcomePresent(t *testing.T) {
	transcript := `Documented the authentication / unauthorized / api key failure modes for operators.
{"outcome":"completed","summary":"docs updated"}`
	if pe := ClassifyProviderError("cursor", "cursor-visible", "cursor-agent output", nil, transcript); pe != nil {
		t.Fatalf("ClassifyProviderError = %#v, want nil for successful outcome with auth prose", pe)
	}
}

func TestClassifyProviderErrorRateLimitDetailMentionsOperation(t *testing.T) {
	pe := ClassifyProviderError("claude", "background-remote", "claude logs", nil, "rate limit hit")
	if pe == nil {
		t.Fatal("expected provider error")
	}
	if !strings.Contains(pe.Detail, "claude logs") {
		t.Fatalf("detail = %q, want operation prefix", pe.Detail)
	}
}

func TestClassifyProviderErrorClaudeExpiredLogin(t *testing.T) {
	text := "Error: not logged in. Please run /login to authenticate with Claude."
	pe := ClassifyProviderError("claude", "background-remote", "claude logs", nil, text)
	if pe == nil || pe.Kind != "auth_required" {
		t.Fatalf("ClassifyProviderError = %#v, want auth_required", pe)
	}
	if pe.RetryPolicy != RetryPolicyManual {
		t.Fatalf("retry_policy = %q, want manual", pe.RetryPolicy)
	}
	if !strings.Contains(pe.Reason, "Claude login required") {
		t.Fatalf("reason = %q, want Claude login message", pe.Reason)
	}
	if pe.SuggestedAction == "" {
		t.Fatal("expected suggested_action")
	}
}

func TestClassifyProviderErrorClaudeANSILoginStillClassifies(t *testing.T) {
	text := "\x1b[31mAuthentication error\x1b[0m: session expired / logged out. Sign in again."
	pe := ClassifyProviderError("claude", "background-remote", "claude logs", nil, text)
	if pe == nil || pe.Kind != "auth_required" {
		t.Fatalf("ClassifyProviderError = %#v, want auth_required for ANSI login failure", pe)
	}
	if strings.Contains(pe.Detail, "\x1b[") {
		t.Fatalf("detail still contains ANSI escapes: %q", pe.Detail)
	}
}

func TestClassifyProviderErrorCodexQuotaAndBilling(t *testing.T) {
	quota := ClassifyProviderError("codex", "codex-app-server", "codex app-server", nil, "usage limit exceeded for this organization")
	if quota == nil || quota.Kind != "quota_exceeded" {
		t.Fatalf("quota = %#v, want quota_exceeded", quota)
	}
	if !strings.Contains(quota.Reason, "Codex quota exhausted") {
		t.Fatalf("quota reason = %q", quota.Reason)
	}

	billing := ClassifyProviderError("codex", "codex-app-server", "codex app-server", nil, "subscription required: billing blocked / payment past due")
	if billing == nil || billing.Kind != "billing_blocked" {
		t.Fatalf("billing = %#v, want billing_blocked", billing)
	}
}

func TestClassifyProviderErrorCursorAuthAndMissingExecutable(t *testing.T) {
	auth := ClassifyProviderError("cursor", "cursor-visible", "cursor-agent create-chat", nil, "Unauthorized: please log in to Cursor Agent")
	if auth == nil || auth.Kind != "auth_required" {
		t.Fatalf("auth = %#v, want auth_required", auth)
	}

	missing := ClassifyProviderError("cursor", "cursor-visible", "launch", exec.ErrNotFound, "cursor-agent: executable file not found in $PATH")
	if missing == nil || missing.Kind != "missing_executable" {
		t.Fatalf("missing = %#v, want missing_executable", missing)
	}
	if !strings.Contains(missing.Reason, "cursor-agent") {
		t.Fatalf("missing reason = %q, want binary name", missing.Reason)
	}
	if !strings.Contains(missing.Detail, "PATH=") {
		t.Fatalf("missing detail = %q, want PATH context", missing.Detail)
	}
}

func TestClassifyProviderErrorCodexMissingExecutable(t *testing.T) {
	pe := ClassifyProviderError("codex", "codex-app-server", "launch plan",
		errors.New("Codex CLI is not available on the daemon PATH"),
		"Codex CLI is not available on the daemon PATH. Install the standalone Codex CLI or configure the Codex binary path, then retry this task.")
	if pe == nil || pe.Kind != "missing_executable" {
		t.Fatalf("ClassifyProviderError = %#v, want missing_executable", pe)
	}
	if !strings.Contains(pe.Reason, "Codex CLI") {
		t.Fatalf("reason = %q", pe.Reason)
	}
}

func TestClassifyProviderErrorGenericRemoteControlDoesNotOverrideAuth(t *testing.T) {
	// Auth signals in hollow-session transcripts must win over leaving the
	// caller to hardcode remote_control_unavailable.
	text := "logged out / expired credentials. Please log in again before using remote control."
	pe := ClassifyProviderError("claude", "background-remote", "claude logs", nil, text)
	if pe == nil || pe.Kind != "auth_required" {
		t.Fatalf("ClassifyProviderError = %#v, want auth_required", pe)
	}
}

func TestProviderErrorFromLaunchBlockerMissingExecutable(t *testing.T) {
	pe := ProviderErrorFromLaunchBlocker("codex", "agent_executable: Codex CLI is not available on the daemon PATH. Install the standalone Codex CLI or configure the Codex binary path, then retry this task.")
	if pe == nil || pe.Kind != "missing_executable" {
		t.Fatalf("ProviderErrorFromLaunchBlocker = %#v, want missing_executable", pe)
	}
}

func TestClassifyProviderErrorIgnoresQuotaProseWhenWorkerOutcomePresent(t *testing.T) {
	transcript := `Documented quota exceeded / usage limit / billing blocked examples for operators.
{"outcome":"completed","summary":"docs updated"}`
	if pe := ClassifyProviderError("codex", "codex-app-server", "codex output", nil, transcript); pe != nil {
		t.Fatalf("ClassifyProviderError = %#v, want nil for successful outcome with quota prose", pe)
	}
}
