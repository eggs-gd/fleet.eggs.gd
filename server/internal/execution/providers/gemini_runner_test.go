package providers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const geminiInit = `{"event":"init","conversation_id":"abc","init":{"cwd":"/tmp"}}`

func geminiResult(status, response, extra string) string {
	body := `{"event":"result","result":{"conversation_id":"abc","status":"` + status + `","response":` + jsonString(response) + extra + `}}`
	return body
}

// runGemini runs RunGeminiHeadlessSession against a shell script standing in
// for the `agy` binary and returns what the runner reported.
func runGemini(t *testing.T, script string, mutate func(*RuntimeSession)) (RuntimeSession, *recorder, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args.txt")
	bin := writeScript(t, dir, "agy", `echo "$@" > "`+argsFile+`"`+"\n"+script)
	session := testSession(dir)
	session.Agent = "gemini"
	session.Backend = BackendGeminiHeadless
	session.GeminiProjectID = "proj-1"
	session.Command = []string{bin}
	if mutate != nil {
		mutate(&session)
	}
	rec := newRecorder()
	RunGeminiHeadlessSession(context.Background(), session, testTask(), rec.options(t.TempDir()))
	args, _ := os.ReadFile(argsFile)
	return rec.lastSession(t), rec, strings.TrimSpace(string(args))
}

func TestRunGeminiHeadlessSessionSucceedsAndKeepsTheConversation(t *testing.T) {
	result := `{"outcome":"completed","summary":"shipped it"}`
	session, rec, args := runGemini(t, "printf '%s\\n' '"+geminiInit+"' '"+geminiResult("SUCCESS", result, "")+"'", nil)

	if args != "--project proj-1" {
		t.Fatalf("agy args = %q", args)
	}
	if session.Status != "exited" || session.ExecutionStatus != "succeeded" {
		t.Fatalf("status = %q / %q", session.Status, session.ExecutionStatus)
	}
	if session.GeminiConversationID != "abc" {
		t.Fatalf("conversation = %q", session.GeminiConversationID)
	}
	if session.Result == nil || session.Result.Summary != "shipped it" || session.LastMessage != "shipped it" {
		t.Fatalf("result = %#v, last message = %q", session.Result, session.LastMessage)
	}
	if !strings.Contains(session.OperatorCommand, "abc") {
		t.Fatalf("operator resume command = %q", session.OperatorCommand)
	}
	if len(rec.started) != 1 || len(rec.completed) != 1 {
		t.Fatalf("started %d, completed %d; each must be reported exactly once", len(rec.started), len(rec.completed))
	}
	if session.ProcessID == 0 || session.ExitedAt == "" || session.StartedAt == "" {
		t.Fatalf("process bookkeeping missing: %#v", session)
	}
}

func TestRunGeminiHeadlessSessionResumesAKnownConversation(t *testing.T) {
	session, _, args := runGemini(t, "printf '%s\\n' '"+geminiResult("SUCCESS", "ok", "")+"'", func(s *RuntimeSession) {
		s.GeminiConversationID = "conv-9"
	})
	if args != "--project proj-1 --conversation conv-9" {
		t.Fatalf("agy args = %q", args)
	}
	if !session.ResumeAttempted {
		t.Fatal("a resume was attempted but not recorded")
	}
}

func TestRunGeminiHeadlessSessionRegistersANewProjectOnFirstLaunch(t *testing.T) {
	_, _, args := runGemini(t, "printf '%s\\n' '"+geminiResult("SUCCESS", "ok", "")+"'", func(s *RuntimeSession) {
		s.GeminiProjectID = ""
	})
	if args != "--new-project" {
		t.Fatalf("agy args = %q, want --new-project", args)
	}
}

func TestRunGeminiHeadlessSessionMapsEveryTerminalStatus(t *testing.T) {
	cases := []struct {
		status        string
		wantStatus    string
		wantExecution string
	}{
		{"WAITING", "waiting_input", "waiting_input"},
		{"CANCELED", "exited", "cancelled"},
		{"INTERRUPTED", "exited", "cancelled"},
		{"ERROR", "exited", "failed"},
		{"WEIRD", "exited", "failed"},
	}
	for _, c := range cases {
		session, _, _ := runGemini(t, "printf '%s\\n' '"+geminiResult(c.status, "", "")+"'", nil)
		if session.Status != c.wantStatus || session.ExecutionStatus != c.wantExecution {
			t.Errorf("%s: status = %q / %q, want %q / %q", c.status, session.Status, session.ExecutionStatus, c.wantStatus, c.wantExecution)
		}
	}
}

func TestRunGeminiHeadlessSessionPausesWhenTheWorkerNeedsInput(t *testing.T) {
	result := `{"outcome":"needs_input","question":"which one?"}`
	session, _, _ := runGemini(t, "printf '%s\\n' '"+geminiResult("SUCCESS", result, "")+"'", nil)
	if session.ExecutionStatus != "waiting_input" || session.Status != "waiting_input" || session.ExitedAt != "" {
		t.Fatalf("a worker question must hold the session: %q / %q / exited=%q", session.Status, session.ExecutionStatus, session.ExitedAt)
	}
}

func TestRunGeminiHeadlessSessionFailsOnANonZeroExit(t *testing.T) {
	session, rec, _ := runGemini(t, "echo boom >&2; exit 3", nil)
	if session.Status != "failed" || session.ExitCode != 3 || session.ExecutionStatus != "failed" {
		t.Fatalf("session = %q / exit %d / %q", session.Status, session.ExitCode, session.ExecutionStatus)
	}
	if !strings.Contains(strings.Join(rec.classified, ","), "agy output") {
		t.Fatalf("the failure was not classified: %v", rec.classified)
	}
}

func TestRunGeminiHeadlessSessionFlagsAStreamWithoutAResult(t *testing.T) {
	session, rec, _ := runGemini(t, "printf '%s\\n' '"+geminiInit+"'", nil)
	if session.ExecutionStatus != "provider_error" || !strings.Contains(session.ErrorMessage, "without a terminal") {
		t.Fatalf("session = %q / %q", session.ExecutionStatus, session.ErrorMessage)
	}
	if len(rec.classified) != 1 {
		t.Fatalf("classified %d times, want once: %v", len(rec.classified), rec.classified)
	}
}

func TestRunGeminiHeadlessSessionFailsLaunchWithoutAUsableCommand(t *testing.T) {
	for name, command := range map[string][]string{"empty": nil, "missing binary": {"/no/such/agy"}} {
		rec := newRecorder()
		session := testSession(t.TempDir())
		session.Command = command
		RunGeminiHeadlessSession(context.Background(), session, testTask(), rec.options(t.TempDir()))
		if len(rec.failed) != 1 {
			t.Errorf("%s: failLaunch called %d times, want once", name, len(rec.failed))
		}
	}
}
