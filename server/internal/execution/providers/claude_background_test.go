package providers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeClaude is a directory-backed stand-in for the `claude` CLI: the test
// writes what each subcommand should print, the script prints it, and the
// script records what it was asked to do.
type fakeClaude struct {
	dir string
	bin string
}

func newFakeClaude(t *testing.T) *fakeClaude {
	t.Helper()
	dir := t.TempDir()
	f := &fakeClaude{dir: dir}
	f.bin = writeScript(t, dir, "claude", `D="`+dir+`"
case "$1" in
--bg)
  echo "$@" > "$D/dispatch_args"
  [ -f "$D/dispatch_fail" ] && { echo "dispatch exploded" >&2; exit 1; }
  cat "$D/dispatch_out" 2>/dev/null ;;
agents) cat "$D/agents.json" 2>/dev/null ;;
logs) cat "$D/logs.txt" 2>/dev/null ;;
stop) echo "$2" > "$D/stopped" ;;
esac`)
	f.set("dispatch_out", "backgrounded · bg-1 · ready\n")
	f.set("agents.json", `[{"id":"bg-1","kind":"bg","status":"running","state":"working","sessionId":"sess-uuid-1"}]`)
	return f
}

func (f *fakeClaude) set(name, body string) {
	_ = os.WriteFile(filepath.Join(f.dir, name), []byte(body), 0o644)
}

func (f *fakeClaude) read(name string) string {
	b, _ := os.ReadFile(filepath.Join(f.dir, name))
	return strings.TrimSpace(string(b))
}

func (f *fakeClaude) stopped() bool {
	_, err := os.Stat(filepath.Join(f.dir, "stopped"))
	return err == nil
}

const remoteControlLog = "rendered chrome - Continue here, on your phone, or at https://claude.ai/code/session_01Np7y5FZ7tMmTgixAbmyXGb\n"

func runBackground(t *testing.T, ctx context.Context, f *fakeClaude, mutate func(*RuntimeSession)) (RuntimeSession, *recorder) {
	t.Helper()
	oldPoll, oldReady := BackgroundRemotePollInterval(), BackgroundRemoteControlReadyTimeout()
	SetBackgroundRemotePollInterval(10 * time.Millisecond)
	t.Cleanup(func() {
		SetBackgroundRemotePollInterval(oldPoll)
		SetBackgroundRemoteControlReadyTimeout(oldReady)
	})
	session := testSession(t.TempDir())
	session.Agent = "claude"
	session.Backend = BackendBackgroundRemote
	session.Command = []string{f.bin, "--bg", "the prompt"}
	if mutate != nil {
		mutate(&session)
	}
	rec := newRecorder()
	opts := rec.options(t.TempDir())
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	RunBackgroundRemoteSession(ctx, session, testTask(), opts)
	return rec.lastSession(t), rec
}

func TestRunBackgroundRemoteSessionStopsAndFinishesOnceTheWorkerReportsAnOutcome(t *testing.T) {
	f := newFakeClaude(t)
	f.set("logs.txt", remoteControlLog+`{"outcome":"completed","summary":"claude done"}`+"\n")

	session, rec := runBackground(t, context.Background(), f, nil)

	if session.BackgroundID != "bg-1" || session.SessionID != "sess-uuid-1" {
		t.Fatalf("ids = %q / %q (the full session uuid is what --resume needs)", session.BackgroundID, session.SessionID)
	}
	if !strings.Contains(session.RemoteControlURL, "session_01Np7y5FZ7tMmTgixAbmyXGb") {
		t.Fatalf("remote control url = %q", session.RemoteControlURL)
	}
	if session.Status != "exited" || session.ExecutionStatus != "succeeded" {
		t.Fatalf("status = %q / %q", session.Status, session.ExecutionStatus)
	}
	if session.Result == nil || session.Result.Summary != "claude done" {
		t.Fatalf("result = %#v", session.Result)
	}
	if !f.stopped() || f.read("stopped") != "bg-1" {
		t.Fatal("the background session was not stopped, so the next task could not resume it cleanly")
	}
	if len(rec.completed) != 1 || !strings.Contains(rec.completed[0], "resumable") {
		t.Fatalf("completed = %v", rec.completed)
	}
	if len(rec.comments) == 0 {
		t.Fatal("the live remote-control session was never announced on the task")
	}
}

func TestRunBackgroundRemoteSessionKeepsAWorkerQuestionOpen(t *testing.T) {
	f := newFakeClaude(t)
	f.set("logs.txt", remoteControlLog+`{"outcome":"needs_input","question":"which?"}`+"\n")
	f.set("agents.json", `[{"id":"bg-1","state":"working","sessionId":"s"}]`)

	// Nothing ends the session on its own here, so end it by cancelling after
	// the question has been seen.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for f.read("stopped") == "" {
			time.Sleep(5 * time.Millisecond)
			if _, err := os.Stat(filepath.Join(f.dir, "dispatch_args")); err == nil {
				time.Sleep(200 * time.Millisecond)
				cancel()
				return
			}
		}
	}()
	session, _ := runBackground(t, ctx, f, nil)
	if session.Result == nil || session.Result.Outcome != "needs_input" {
		t.Fatalf("result = %#v", session.Result)
	}
	// It was stopped only because the runtime was cancelled, never because of
	// the question: the operator may answer in the live session.
	if session.LastEvent == "" || session.ErrorMessage != "runtime shutting down" {
		t.Fatalf("last event %q, error %q", session.LastEvent, session.ErrorMessage)
	}
}

func TestRunBackgroundRemoteSessionFinishesWhenTheSessionDisappears(t *testing.T) {
	f := newFakeClaude(t)
	f.set("logs.txt", remoteControlLog)
	f.set("agents.json", `[]`)

	session, _ := runBackground(t, context.Background(), f, nil)
	if session.Status != "exited" || !strings.Contains(session.ErrorMessage, "no longer listed") {
		t.Fatalf("status = %q, error = %q", session.Status, session.ErrorMessage)
	}
}

func TestRunBackgroundRemoteSessionFailsWhenTheAgentEndsInError(t *testing.T) {
	f := newFakeClaude(t)
	f.set("logs.txt", remoteControlLog)
	f.set("agents.json", `[{"id":"bg-1","state":"error","sessionId":"s"}]`)

	session, _ := runBackground(t, context.Background(), f, nil)
	if session.Status != "failed" || !strings.Contains(session.ErrorMessage, "state error") {
		t.Fatalf("status = %q, error = %q", session.Status, session.ErrorMessage)
	}
}

func TestRunBackgroundRemoteSessionGivesUpOnAHollowSession(t *testing.T) {
	f := newFakeClaude(t)
	f.set("logs.txt", "no url here\n") // dispatched, but Claude never publishes a remote-control URL
	SetBackgroundRemoteControlReadyTimeout(60 * time.Millisecond)

	session, _ := runBackground(t, context.Background(), f, nil)
	if session.Status != "failed" || session.ProviderError == nil || session.ProviderError.Kind != "remote_control_unavailable" {
		t.Fatalf("status = %q, provider error = %#v", session.Status, session.ProviderError)
	}
	if !f.stopped() {
		t.Fatal("a hollow session was left running")
	}
}

func TestRunBackgroundRemoteSessionDoesNotStartOnADeadContext(t *testing.T) {
	f := newFakeClaude(t)
	rec := newRecorder()
	session := testSession(t.TempDir())
	session.Command = []string{f.bin, "--bg", "p"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	RunBackgroundRemoteSession(ctx, session, testTask(), rec.options(t.TempDir()))
	if f.read("dispatch_args") != "" {
		t.Fatalf("a cancelled runtime still dispatched a background session: %q", f.read("dispatch_args"))
	}
}

func TestRunBackgroundRemoteSessionRewritesAResumeToTheFullSessionID(t *testing.T) {
	f := newFakeClaude(t)
	f.set("logs.txt", remoteControlLog+`{"outcome":"completed","summary":"ok"}`+"\n")

	runBackground(t, context.Background(), f, func(s *RuntimeSession) {
		s.SessionID = "sess-uuid-1"
		s.BackgroundID = "bg-1"
		s.Command = []string{f.bin, "--bg", "--remote-control", "--name", "x", "the prompt"}
	})
	if got := f.read("dispatch_args"); got != "--bg --resume sess-uuid-1 the prompt" {
		t.Fatalf("dispatch args = %q; a resume must carry only the session id and the prompt, or Claude forks a copy", got)
	}
}

func TestRunBackgroundRemoteSessionFailsLaunchWhenDispatchFails(t *testing.T) {
	cases := map[string]func(*fakeClaude){
		"dispatch exits non-zero": func(f *fakeClaude) { f.set("dispatch_fail", "1") },
		"dispatch prints no id":   func(f *fakeClaude) { f.set("dispatch_out", "something unexpected\n") },
	}
	for name, setup := range cases {
		f := newFakeClaude(t)
		setup(f)
		rec := newRecorder()
		session := testSession(t.TempDir())
		session.Command = []string{f.bin, "--bg", "p"}
		RunBackgroundRemoteSession(context.Background(), session, testTask(), rec.options(t.TempDir()))
		if len(rec.failed) != 1 {
			t.Errorf("%s: failLaunch called %d times, want once", name, len(rec.failed))
		}
		if len(rec.classified) == 0 {
			t.Errorf("%s: the failure was not offered to the classifier", name)
		}
	}
}

func TestBackgroundAgentStatusQueries(t *testing.T) {
	f := newFakeClaude(t)
	status, found, err := QueryBackgroundAgentStatus(f.bin, "bg-1")
	if err != nil || !found || status.SessionID != "sess-uuid-1" {
		t.Fatalf("status = %#v, %v, %v", status, found, err)
	}
	if _, found, _ := QueryBackgroundAgentStatus(f.bin, "someone-else"); found {
		t.Fatal("found a session that is not listed")
	}
	f.set("agents.json", "not json")
	if _, _, err := QueryBackgroundAgentStatus(f.bin, "bg-1"); err == nil {
		t.Fatal("unparseable agent list reported success")
	}

	for state, want := range map[string]bool{"done": true, "STOPPED": true, "error": true, "failed": true, "working": false, "": false} {
		if got := IsTerminalBackgroundState(BackgroundAgentStatus{State: state}); got != want {
			t.Errorf("IsTerminalBackgroundState(%q) = %v, want %v", state, got, want)
		}
	}
	if logs, err := FetchBackgroundSessionLogs("", "x"); logs != "" || err != nil {
		t.Fatalf("no binary must be a quiet no-op, got %q, %v", logs, err)
	}
	if err := StopBackgroundSession(f.bin, "bg-9"); err != nil || f.read("stopped") != "bg-9" {
		t.Fatalf("stop err = %v, recorded %q", err, f.read("stopped"))
	}
}
