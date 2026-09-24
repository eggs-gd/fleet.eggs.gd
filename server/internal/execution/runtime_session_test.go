package execution

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRuntimeSessionJSONUsesFlatProviderKeys(t *testing.T) {
	session := RuntimeSession{
		ClaimID: "claim-1",
		ClaudeSessionDetails: ClaudeSessionDetails{
			BackgroundID:     "bg-123",
			RemoteControlURL: "https://claude.ai/code/session_abc",
		},
		CodexSessionDetails: CodexSessionDetails{
			CodexThreadID: "thread-456",
		},
		CursorSessionDetails: CursorSessionDetails{
			CursorChatID: "chat-789",
		},
	}

	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	raw := string(data)
	for _, key := range []string{"background_id", "codex_thread_id", "cursor_chat_id"} {
		if !strings.Contains(raw, `"`+key+`"`) {
			t.Fatalf("expected flat key %q in JSON: %s", key, raw)
		}
	}
	for _, nested := range []string{"ClaudeSessionDetails", "CodexSessionDetails", "CursorSessionDetails"} {
		if strings.Contains(raw, nested) {
			t.Fatalf("expected no nested struct key %q in JSON: %s", nested, raw)
		}
	}

	var roundTrip RuntimeSession
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if roundTrip.BackgroundID != "bg-123" {
		t.Fatalf("BackgroundID = %q", roundTrip.BackgroundID)
	}
	if roundTrip.CodexThreadID != "thread-456" {
		t.Fatalf("CodexThreadID = %q", roundTrip.CodexThreadID)
	}
	if roundTrip.CursorChatID != "chat-789" {
		t.Fatalf("CursorChatID = %q", roundTrip.CursorChatID)
	}
}

func TestRuntimeSessionProviderIDHelpers(t *testing.T) {
	codex := RuntimeSession{
		CodexSessionDetails:  CodexSessionDetails{CodexThreadID: "thread-1"},
		CursorSessionDetails: CursorSessionDetails{CursorChatID: "chat-1"},
		ClaudeSessionDetails: ClaudeSessionDetails{BackgroundID: "bg-1"},
	}
	if got := codex.ProviderThreadID(); got != "thread-1" {
		t.Fatalf("ProviderThreadID() = %q, want thread-1", got)
	}
	if got := codex.ProviderSessionID(); got != "chat-1" {
		t.Fatalf("ProviderSessionID() = %q, want chat-1 (cursor before background)", got)
	}

	claudeOnly := RuntimeSession{
		ClaudeSessionDetails: ClaudeSessionDetails{
			BackgroundID:     "bg-only",
			RemoteControlURL: "https://claude.ai/code/session_xyz",
		},
	}
	if got := claudeOnly.ProviderThreadID(); got != "bg-only" {
		t.Fatalf("ProviderThreadID() = %q, want bg-only", got)
	}
	if got := claudeOnly.ProviderSessionID(); got != "https://claude.ai/code/session_xyz" {
		t.Fatalf("ProviderSessionID() = %q, want remote control URL", got)
	}
}
