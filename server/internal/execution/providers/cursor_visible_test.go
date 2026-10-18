package providers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// cursorScript is a fake `cursor-agent`: `create-chat` prints $CHAT_OUT, any
// other invocation records its arguments and runs $RUN_BODY.
func cursorScript(t *testing.T, createBody, runBody string) (bin, argsFile string) {
	t.Helper()
	dir := t.TempDir()
	argsFile = filepath.Join(dir, "args")
	bin = writeScript(t, dir, "cursor-agent", `if [ "$1" = "create-chat" ]; then
`+createBody+`
else
echo "$@" > "`+argsFile+`"
`+runBody+`
fi`)
	return bin, argsFile
}

func runCursor(t *testing.T, bin string, mutate func(*RuntimeSession)) (RuntimeSession, *recorder) {
	t.Helper()
	session := testSession(t.TempDir())
	session.Agent = "cursor"
	session.Backend = BackendCursorVisible
	session.Command = []string{bin, "--print", "the prompt"}
	if mutate != nil {
		mutate(&session)
	}
	rec := newRecorder()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	RunCursorVisibleSession(ctx, session, testTask(), rec.options(t.TempDir()))
	return rec.lastSession(t), rec
}

func TestRunCursorVisibleSessionCreatesAChatThenResumesIt(t *testing.T) {
	chat := "0b0a6ed0-1111-4222-8333-444455556666"
	bin, argsFile := cursorScript(t, `echo "`+chat+`"`, `echo done`)

	session, rec := runCursor(t, bin, nil)

	if session.CursorChatID != chat || session.BackgroundID != chat {
		t.Fatalf("chat id = %q / %q", session.CursorChatID, session.BackgroundID)
	}
	if session.Status != "exited" || session.ExecutionStatus != "succeeded" {
		t.Fatalf("status = %q / %q", session.Status, session.ExecutionStatus)
	}
	args, _ := os.ReadFile(argsFile)
	if got := strings.TrimSpace(string(args)); got != "--resume "+chat+" --print the prompt" {
		t.Fatalf("agent args = %q", got)
	}
	if !strings.Contains(session.OperatorCommand, chat) {
		t.Fatalf("operator command = %q; the operator could not pick the chat up", session.OperatorCommand)
	}
	if len(rec.comments) == 0 {
		t.Fatal("the live chat was never announced on the task")
	}
}

func TestRunCursorVisibleSessionResumesAnExistingChat(t *testing.T) {
	bin, argsFile := cursorScript(t, `echo "must not be called" >&2; exit 9`, `echo ok`)

	session, _ := runCursor(t, bin, func(s *RuntimeSession) { s.CursorChatID = "existing-chat" })

	if !session.ResumeAttempted || session.ResumeOutcome != "resumed" {
		t.Fatalf("resume = %v / %q", session.ResumeAttempted, session.ResumeOutcome)
	}
	args, _ := os.ReadFile(argsFile)
	if !strings.HasPrefix(string(args), "--resume existing-chat ") {
		t.Fatalf("agent args = %q", args)
	}
}

func TestRunCursorVisibleSessionFailsLaunchWhenChatCreationBreaks(t *testing.T) {
	cases := map[string]string{
		"create-chat exits non-zero": `echo boom >&2; exit 3`,
		"create-chat prints no id":   `echo "no id"`,
	}
	for name, body := range cases {
		bin, argsFile := cursorScript(t, body, `echo ok`)
		rec := newRecorder()
		session := testSession(t.TempDir())
		session.Command = []string{bin, "--print", "p"}
		RunCursorVisibleSession(context.Background(), session, testTask(), rec.options(t.TempDir()))
		if len(rec.failed) != 1 {
			t.Errorf("%s: failLaunch called %d times, want once", name, len(rec.failed))
		}
		if _, err := os.Stat(argsFile); err == nil {
			t.Errorf("%s: the agent ran without a chat", name)
		}
	}
}

func TestRunCursorVisibleSessionReportsAFailingAgent(t *testing.T) {
	bin, _ := cursorScript(t, `echo "0b0a6ed0-1111-4222-8333-444455556666"`, `echo "it broke" >&2; exit 4`)

	session, _ := runCursor(t, bin, nil)
	if session.ExitCode != 4 || session.ErrorMessage == "" {
		t.Fatalf("exit code %d, error %q", session.ExitCode, session.ErrorMessage)
	}
	if session.Status != "failed" && session.ExecutionStatus != "provider_error" {
		t.Fatalf("status = %q / %q", session.Status, session.ExecutionStatus)
	}
}

func TestRunCursorVisibleSessionKeepsAWorkerQuestionOpen(t *testing.T) {
	bin, _ := cursorScript(t, `echo "0b0a6ed0-1111-4222-8333-444455556666"`,
		`echo '{"outcome":"needs_input","question":"which one?"}'`)

	session, _ := runCursor(t, bin, nil)
	if session.Status != "waiting_input" || session.ExecutionStatus != "waiting_input" {
		t.Fatalf("status = %q / %q", session.Status, session.ExecutionStatus)
	}
	if session.ExitedAt != "" {
		t.Fatal("a session waiting for the operator must not look exited")
	}
}
