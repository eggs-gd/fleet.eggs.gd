package providers

import (
	"strings"
	"testing"
)

func TestParseRemoteControlURLIgnoresPromptEchoedHistoricalURL(t *testing.T) {
	log := strings.Join([]string{
		"You are a Fleet worker agent launched by the deterministic Fleet daemon.",
		"## Review Comments",
		"- 2026-08-02T02:09:24+03:00 - core: Remote-control session is live: https://claude.ai/code/session_01W8XF7nseQB2QdtbHpexTuh",
		"",
		"some rendered TUI chrome - Continue here, on your phone, or at https://claude.ai/code/session_01Np7y5FZ7tMmTgixAbmyXGb",
	}, "\n")
	if got := ParseRemoteControlURL(log); got != "https://claude.ai/code/session_01Np7y5FZ7tMmTgixAbmyXGb" {
		t.Fatalf("ParseRemoteControlURL() = %q", got)
	}
}

func TestParseRemoteControlURLNoAnnouncementYet(t *testing.T) {
	log := "## Review Comments\n- core: Remote-control session is live: https://claude.ai/code/session_01W8XF7nseQB2QdtbHpexTuh\n"
	if got := ParseRemoteControlURL(log); got != "" {
		t.Fatalf("ParseRemoteControlURL() = %q", got)
	}
}

func TestParseCursorChatID(t *testing.T) {
	cases := map[string]string{
		"chat_abc123XYZ\n":                "chat_abc123XYZ",
		"Created chat: `019fbf22-35f7`\n": "019fbf22-35f7",
		"noise\n\n":                       "",
	}
	for input, want := range cases {
		if got := ParseCursorChatID(input); got != want {
			t.Fatalf("ParseCursorChatID(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCursorOperatorCommand(t *testing.T) {
	got := CursorOperatorCommand("cursor-agent", "chat-123", "/tmp/repo")
	want := "cursor-agent --resume chat-123 --workspace /tmp/repo"
	if got != want {
		t.Fatalf("CursorOperatorCommand() = %q, want %q", got, want)
	}
}

func TestCodexRemoteThreadTitle(t *testing.T) {
	if got := CodexRemoteThreadTitle("CORE-80", "Set meaningful thread titles", "work-core-80"); got != "CORE-80 · Set meaningful thread titles" {
		t.Fatalf("CodexRemoteThreadTitle() = %q", got)
	}
}
