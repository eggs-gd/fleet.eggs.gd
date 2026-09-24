package tasklifecycle

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Retry policies for provider failures. Auth/quota/billing/missing-binary
// failures must not be auto-retried until an operator changes something.
const (
	RetryPolicyManual      = "manual"
	RetryPolicyAfterReset  = "after_reset"
	RetryPolicyUnavailable = "unavailable"
)

// ProviderError is a task-facing classification of a worker provider
// failure (auth, quota, billing, transport, ...), attached to execution
// state so blocked-task comments can explain what actually went wrong.
type ProviderError struct {
	Provider        string `json:"provider"`
	Kind            string `json:"kind"`
	Reason          string `json:"reason"`
	Detail          string `json:"detail,omitempty"`
	SuggestedAction string `json:"suggested_action,omitempty"`
	RetryPolicy     string `json:"retry_policy,omitempty"`
}

var (
	ansiEscapeSequence = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[@-Z\\-_]`)
	retryAfterPattern  = regexp.MustCompile(`(?i)retry[- ]after[:\s]+(\d+\s*(?:seconds?|minutes?|hours?|s|m|h)?|\d{4}-\d{2}-\d{2}[^\s]*)`)
)

// ClassifyProviderError inspects a provider operation's error/output and
// returns a task-facing classification, or nil when nothing recognizable was
// found.
//
// A recognized worker JSON outcome in the same output wins over incidental
// keyword matches (rate limit / 429 / auth / quota / "connection refused"
// appearing in tests, docs, or summaries). Real transport/control-plane
// failures carried by err still classify.
func ClassifyProviderError(provider string, backend string, operation string, err error, output string) *ProviderError {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "agent"
	}
	text := stripProviderNoise(firstNonEmpty(output, errorString(err)))
	hasOutcome := ParseWorkerResult(text) != nil

	if pe := classifyMissingExecutable(provider, operation, err, text); pe != nil {
		if err != nil || !hasOutcome {
			return pe
		}
	}
	if pe := classifyTransportProviderError(provider, operation, err, text); pe != nil {
		// err-backed transport failures always win. Text-only transport
		// keyword hits must not override a valid worker outcome (CORE-98).
		if err != nil || !hasOutcome {
			return pe
		}
	}
	if hasOutcome {
		return nil
	}
	return classifyProviderText(provider, operation, text)
}

func classifyMissingExecutable(provider string, operation string, err error, text string) *ProviderError {
	lower := strings.ToLower(text)
	if errors.Is(err, exec.ErrNotFound) ||
		strings.Contains(lower, "executable file not found") ||
		strings.Contains(lower, "not found in $path") ||
		strings.Contains(lower, "binary could not be resolved") ||
		strings.Contains(lower, "is not available on the daemon path") ||
		(strings.Contains(lower, "is not on path") && strings.Contains(lower, "binary")) {
		binary := providerBinaryName(provider)
		pathHint := pathEnvHint()
		detail := firstNonEmpty(text, errorString(err))
		if pathHint != "" && !strings.Contains(strings.ToLower(detail), "path=") {
			detail = strings.TrimSpace(detail + " PATH=" + pathHint)
		}
		return providerError(provider, "missing_executable", operation,
			providerDisplayName(provider)+" CLI (`"+binary+"`) is not available on the daemon PATH. Install "+providerDisplayName(provider)+" or configure the launcher with the absolute binary path, then retry this task.",
			detail)
	}
	return nil
}

func classifyTransportProviderError(provider string, operation string, err error, text string) *ProviderError {
	// Scan err's own message too, not just text: when a background session
	// has both a transcript (text) and a live dial/socket error (err), the
	// caller's contract is that a real transport error always wins even if
	// text alone (e.g. a worker-reported "completed" outcome) shows nothing
	// suspicious. errors.As only catches *net.OpError; a plain errors.New
	// wrapping "connection refused" needs the keyword scan below too.
	errText := errorString(err)
	scan := strings.ToLower(text)
	if errText != "" {
		scan += " " + strings.ToLower(errText)
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) || errors.Is(err, os.ErrDeadlineExceeded) ||
		strings.Contains(scan, "econnrefused") ||
		strings.Contains(scan, "connection refused") ||
		strings.Contains(scan, "control socket") ||
		strings.Contains(scan, "daemon control socket") ||
		(strings.Contains(scan, "no such file or directory") && strings.Contains(scan, "socket")) {
		return providerError(provider, "control_socket_unavailable", operation, "Provider daemon or control socket is unavailable.", firstNonEmpty(text, errText))
	}
	return nil
}

func classifyProviderText(provider string, operation string, text string) *ProviderError {
	lower := strings.ToLower(text)
	switch {
	case containsAny(lower, "secitemcopymatching failed", "keychain"):
		return providerError(provider, "local_keychain_unavailable", operation, "Provider authentication/local keychain state is unavailable to the daemon user.", text)
	case containsAny(lower, ".cursor/projects", "operation not permitted", "eperm"):
		return providerError(provider, "local_state_unwritable", operation, "Provider local state directory is not writable from this runtime environment.", text)
	case containsAny(lower, "exhausted credits", "credit balance", "insufficient balance", "not enough credits", "out of credits"):
		return providerError(provider, "exhausted_credits", operation,
			providerDisplayName(provider)+" credits exhausted. Add credits or switch provider before retrying.",
			text)
	case containsAny(lower, "billing", "subscription") && containsAny(lower, "blocked", "required", "past due", "payment", "disabled", "inactive", "unavailable"):
		return providerError(provider, "billing_blocked", operation,
			providerDisplayName(provider)+" billing or subscription is blocked. Fix billing on the provider account, then retry this task.",
			text)
	case containsAny(lower, "rate limit", "rate_limit", "too many requests", "429"):
		// Checked before the generic quota case below: "rate limit exceeded"
		// contains the substring "limit exceeded", which would otherwise
		// misclassify a plain rate-limit hit as quota_exceeded.
		reason := providerDisplayName(provider) + " rate limit was hit. Wait before retrying."
		if hint := retryAfterHint(text); hint != "" {
			reason = providerDisplayName(provider) + " rate limit was hit. Retry after " + hint + "."
		}
		return providerError(provider, "rate_limited", operation, reason, text)
	case containsAny(lower, "quota exceeded", "exceeded your quota", "usage limit", "limit exceeded", "resource exhausted", "quota exhausted"):
		return providerError(provider, "quota_exceeded", operation,
			providerDisplayName(provider)+" quota exhausted. Wait for reset, add credits, or switch provider before retrying.",
			text)
	case containsAny(lower,
		"authentication", "unauthorized", "invalid api key", "api key",
		"not logged in", "login required", "please log in", "please login",
		"please run /login", "run /login", "sign in", "signin required",
		"logged out", "log in again", "login again", "re-authenticate",
		"reauthenticate", "session expired", "token expired", "expired credentials",
		"credentials expired", "oauth", "auth error", "authentication_error",
		"401 unauthorized", "http 401"):
		return providerError(provider, "auth_required", operation,
			providerDisplayName(provider)+" login required. Sign in to "+providerDisplayName(provider)+" on the daemon host, then retry this task.",
			text)
	case containsAny(lower, "permission denied"):
		return providerError(provider, "auth_required", operation,
			providerDisplayName(provider)+" login required. Sign in to "+providerDisplayName(provider)+" on the daemon host, then retry this task.",
			text)
	default:
		return nil
	}
}

// NewProviderError builds a ProviderError directly, for callers that already
// know the classification (e.g. "the operator explicitly released a stuck
// session") instead of deriving it from raw process output via
// ClassifyProviderError.
func NewProviderError(provider string, kind string, operation string, reason string, detail string) *ProviderError {
	return providerError(provider, kind, operation, reason, detail)
}

// ProviderErrorFromLaunchBlocker turns a launch-gate failure reason into a
// classified provider error when the gate is a known setup/auth problem.
func ProviderErrorFromLaunchBlocker(provider string, reason string) *ProviderError {
	provider = strings.TrimSpace(provider)
	reason = strings.TrimSpace(reason)
	if provider == "" || reason == "" {
		return nil
	}
	lower := strings.ToLower(reason)
	if strings.HasPrefix(lower, "agent_executable:") || strings.Contains(lower, "binary could not be resolved") {
		detail := reason
		if idx := strings.Index(reason, ":"); idx >= 0 {
			detail = strings.TrimSpace(reason[idx+1:])
		}
		return ClassifyProviderError(provider, "", "launch plan", errors.New(detail), detail)
	}
	return nil
}

func providerError(provider string, kind string, operation string, reason string, detail string) *ProviderError {
	detail = compactProviderDetail(stripProviderNoise(detail))
	if operation != "" && detail != "" {
		detail = operation + ": " + detail
	}
	pe := &ProviderError{
		Provider:        provider,
		Kind:            kind,
		Reason:          reason,
		Detail:          detail,
		SuggestedAction: suggestedActionForKind(provider, kind, reason),
		RetryPolicy:     retryPolicyForKind(kind),
	}
	return pe
}

func suggestedActionForKind(provider string, kind string, reason string) string {
	name := providerDisplayName(provider)
	switch kind {
	case "missing_executable":
		return "Install " + name + " or configure the launcher with the absolute binary path, then move this task to todo/needs_rework."
	case "auth_required", "auth_failed", "local_keychain_unavailable":
		return "Sign in to " + name + " on the daemon host, then move this task to todo/needs_rework."
	case "quota_exceeded", "exhausted_credits":
		return "Wait for quota reset, add credits, or switch provider, then move this task to todo/needs_rework."
	case "billing_blocked":
		return "Fix " + name + " billing/subscription, then move this task to todo/needs_rework."
	case "rate_limited":
		if hint := retryAfterHint(reason); hint != "" {
			return "Wait until " + hint + ", then move this task to todo/needs_rework for an explicit retry."
		}
		return "Wait for the rate limit to clear, then move this task to todo/needs_rework for an explicit retry."
	case "remote_control_unavailable":
		return "Confirm " + name + " Desktop/login and remote-control registration on the daemon host, then retry."
	case "control_socket_unavailable":
		return "Restore the " + name + " daemon/control socket, then retry."
	case "local_state_unwritable":
		return "Fix local filesystem permissions for the " + name + " state directory, then retry."
	default:
		return "Fix the provider account/session or daemon, then move this task to todo/needs_rework."
	}
}

func retryPolicyForKind(kind string) string {
	switch kind {
	case "auth_required", "auth_failed", "quota_exceeded", "exhausted_credits",
		"billing_blocked", "missing_executable", "local_keychain_unavailable",
		"local_state_unwritable", "remote_control_unavailable", "control_socket_unavailable":
		return RetryPolicyManual
	case "rate_limited":
		return RetryPolicyAfterReset
	default:
		return RetryPolicyManual
	}
}

func providerDisplayName(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "claude":
		return "Claude"
	case "codex":
		return "Codex"
	case "cursor":
		return "Cursor"
	case "plane":
		return "Plane"
	default:
		name := strings.TrimSpace(provider)
		if name == "" {
			return "Provider"
		}
		return strings.ToUpper(name[:1]) + name[1:]
	}
}

func providerBinaryName(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "cursor":
		return "cursor-agent"
	case "claude":
		return "claude"
	case "codex":
		return "codex"
	default:
		return firstNonEmpty(strings.TrimSpace(provider), "agent")
	}
}

func pathEnvHint() string {
	path := strings.TrimSpace(os.Getenv("PATH"))
	if path == "" {
		return "(empty)"
	}
	const limit = 180
	if len(path) <= limit {
		return path
	}
	return path[:limit] + "..."
}

func retryAfterHint(text string) string {
	match := retryAfterPattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func stripProviderNoise(text string) string {
	text = ansiEscapeSequence.ReplaceAllString(text, "")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.TrimSpace(text)
}

func compactProviderDetail(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	text = whitespace.ReplaceAllString(text, " ")
	const limit = 360
	if len(text) <= limit {
		return text
	}
	return strings.TrimSpace(text[:limit]) + "..."
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
