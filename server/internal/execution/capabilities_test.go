package execution

import (
	"testing"
)

const capabilityUnknown = CapabilityUnknown

type sessionReuseLevel = SessionReuseLevel

const (
	sessionReuseVerified    = SessionReuseVerified
	sessionReuseUnverified  = SessionReuseUnverified
	sessionReuseUnsupported = SessionReuseUnsupported
)

// TestProviderSessionReuseCapabilityMatrix pins the per-backend reuse rules
// CORE-77 depends on: Codex may be attempted automatically because a failed
// thread/resume is safely observable before any visible side effect, Claude
// background-remote reuse stays unverified/no-attempt until empirically
// confirmed, and unknown/process backends stay unsupported.
func TestProviderSessionReuseCapabilityMatrix(t *testing.T) {
	cases := []struct {
		backend        string
		wantLevel      sessionReuseLevel
		wantCanAttempt bool
	}{
		{BackendCursorVisible, sessionReuseVerified, true},
		{BackendCodexAppServer, sessionReuseUnverified, true},
		{BackendBackgroundRemote, sessionReuseVerified, true},
		{BackendGeminiHeadless, sessionReuseVerified, true},
		{"process", sessionReuseUnsupported, false},
		{"terminal", sessionReuseUnsupported, false},
		{"", sessionReuseUnsupported, false},
		{"totally-unknown-backend", sessionReuseUnsupported, false},
	}
	for _, tc := range cases {
		capability := ProviderSessionReuseCapability(tc.backend)
		if capability.Level != tc.wantLevel {
			t.Errorf("backend %q: level = %q, want %q", tc.backend, capability.Level, tc.wantLevel)
		}
		if capability.CanAttempt != tc.wantCanAttempt {
			t.Errorf("backend %q: can_attempt = %v, want %v", tc.backend, capability.CanAttempt, tc.wantCanAttempt)
		}
		if capability.Reason == "" {
			t.Errorf("backend %q: expected a non-empty reason", tc.backend)
		}
	}
}

func TestProviderCapabilityFlagsCursorVisible(t *testing.T) {
	capability := ProviderCapabilityFlags("cursor", BackendCursorVisible)
	if !capability.CanStartVisibleSession || !capability.CanListSessions || !capability.CanExposeOperatorPath || !capability.CanDetectTerminalState {
		t.Fatalf("cursor visible launch capabilities should all be true: %#v", capability)
	}
	if capability.CanDetectRunningSession != CapabilityYes || capability.CanResumeSession != CapabilityYes || capability.CanQueryThreadStatus != CapabilityNo {
		t.Fatalf("cursor recovery capabilities unexpected: %#v", capability)
	}
	if capability.CanShowAppVisibleLink != CapabilityNo || capability.CanAcceptOperatorInput != CapabilityYes || capability.CanConfirmTerminalOutcome != CapabilityYes {
		t.Fatalf("cursor operator/recovery capabilities unexpected: %#v", capability)
	}
	if capability.OperatorVisibility != string(VisibilityCLIVisible) || capability.TerminalStateDetection != "process_exit" {
		t.Fatalf("unexpected cursor visibility/detection metadata: %#v", capability)
	}
	if !VisibilityClass(capability.OperatorVisibility).AllowsDaemonAutoLaunch() {
		t.Fatal("cli_visible should allow daemon auto-launch")
	}
}

func TestProviderCapabilityFlagsCodexFleetVisible(t *testing.T) {
	capability := ProviderCapabilityFlags("codex", BackendCodexAppServer)
	if capability.OperatorVisibility != string(VisibilityFleetVisible) {
		t.Fatalf("codex visibility = %q, want fleet_visible", capability.OperatorVisibility)
	}
	if !capability.CanStartVisibleSession {
		t.Fatal("codex should claim can_start_visible_session for Codex Remote + Fleet control")
	}
	if capability.CanShowAppVisibleLink.IsYes() {
		t.Fatal("codex must not claim a Claude-style app-visible deep link")
	}
	if !VisibilityClass(capability.OperatorVisibility).AllowsDaemonAutoLaunch() {
		t.Fatal("fleet_visible Codex path must allow normal daemon auto-launch")
	}
}

func TestHeadlessVisibilityStillBlocksDaemonAutoLaunch(t *testing.T) {
	if VisibilityClass(VisibilityHeadless).AllowsDaemonAutoLaunch() {
		t.Fatal("headless must remain blocked from daemon auto-launch")
	}
	if VisibilityClass(VisibilityUnknown).AllowsDaemonAutoLaunch() {
		t.Fatal("unknown visibility must remain blocked from daemon auto-launch")
	}
}

func TestProviderCapabilityFlagsClaudeAppVisible(t *testing.T) {
	capability := ProviderCapabilityFlags("claude", BackendBackgroundRemote)
	if capability.OperatorVisibility != string(VisibilityAppVisible) {
		t.Fatalf("claude visibility = %q, want app_visible", capability.OperatorVisibility)
	}
	if !capability.CanShowAppVisibleLink.IsYes() || !capability.CanStartVisibleSession {
		t.Fatalf("claude should expose app-visible link capability: %#v", capability)
	}
	if !VisibilityClass(capability.OperatorVisibility).AllowsDaemonAutoLaunch() {
		t.Fatal("app_visible should allow daemon auto-launch")
	}
}

func TestProviderCapabilityFlagsUnknownBackend(t *testing.T) {
	capability := ProviderCapabilityFlags("codex", "mystery-backend")
	if capability.CanDetectRunningSession != capabilityUnknown || capability.CanResumeSession != capabilityUnknown || capability.CanQueryThreadStatus != capabilityUnknown {
		t.Fatalf("unknown backend should expose unknown recovery capabilities: %#v", capability)
	}
	if capability.CanShowAppVisibleLink != capabilityUnknown || capability.CanAcceptOperatorInput != capabilityUnknown || capability.CanConfirmTerminalOutcome != capabilityUnknown {
		t.Fatalf("unknown backend operator capabilities should be unknown: %#v", capability)
	}
}
