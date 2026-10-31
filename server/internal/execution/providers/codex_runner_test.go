package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// TestFakeCodexAppServer is not a test: it is the body of a fake `codex
// app-server` child process. The runner tests re-execute the test binary
// with FLEET_FAKE_CODEX set, so the process speaks real JSON-RPC over real
// pipes without needing the codex CLI.
func TestFakeCodexAppServer(t *testing.T) {
	mode := os.Getenv("FLEET_FAKE_CODEX")
	if mode == "" {
		t.Skip("only runs as a child process of the runner tests")
	}
	if mode == "crash" {
		_, _ = os.Stderr.WriteString("codex: fatal: not signed in\n")
		os.Exit(3)
	}
	in := bufio.NewScanner(os.Stdin)
	out := os.Stdout
	reply := func(id json.RawMessage, result string) {
		_, _ = out.WriteString(`{"id":` + string(id) + `,"result":` + result + "}\n")
	}
	for in.Scan() {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if json.Unmarshal(in.Bytes(), &req) != nil || req.Method == "" || len(req.ID) == 0 {
			continue
		}
		switch req.Method {
		case "initialize", "thread/name/set":
			reply(req.ID, `{}`)
		case "thread/start":
			reply(req.ID, `{"thread":{"id":"thr-1"}}`)
		case "turn/start":
			reply(req.ID, `{"turn":{"id":"turn-1"}}`)
			result := `{"outcome":"completed","summary":"done by fake codex"}`
			delta, _ := json.Marshal(result)
			_, _ = out.WriteString(`{"method":"item/agentMessage/delta","params":{"delta":` + string(delta) + "}}\n")
			_, _ = out.WriteString(`{"method":"turn/completed","params":{"turn":{"id":"turn-1","status":"completed"}}}` + "\n")
			os.Exit(0)
		}
	}
	os.Exit(0)
}

func runCodexChild(t *testing.T, mode string) (RuntimeSession, *recorder) {
	t.Helper()
	t.Setenv("FLEET_FAKE_CODEX", mode)
	session := testSession(t.TempDir())
	session.Command = []string{os.Args[0], "-test.run=^TestFakeCodexAppServer$"}
	rec := newRecorder()
	opts := rec.options(t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	RunCodexAppServerSession(ctx, session, testTask(), opts)
	return rec.lastSession(t), rec
}

func TestRunCodexAppServerSessionDrivesARealChildProcessToCompletion(t *testing.T) {
	session, rec := runCodexChild(t, "ok")

	if session.Status != "exited" || session.ExecutionStatus != "succeeded" {
		t.Fatalf("session = %q / %q (error %q)", session.Status, session.ExecutionStatus, session.ErrorMessage)
	}
	if session.CodexThreadID != "thr-1" || session.CodexTurnID != "turn-1" {
		t.Fatalf("ids = %q / %q", session.CodexThreadID, session.CodexTurnID)
	}
	if session.Result == nil || session.Result.Summary != "done by fake codex" {
		t.Fatalf("result = %#v", session.Result)
	}
	if session.ProcessID == 0 || session.ExitedAt == "" {
		t.Fatalf("process bookkeeping missing: %#v", session)
	}
	if len(rec.started) != 1 || len(rec.completed) != 1 {
		t.Fatalf("started %d, completed %d; each must be reported exactly once", len(rec.started), len(rec.completed))
	}
	// The control channel is registered for the live session and released after.
	select {
	case <-rec.registered:
	default:
		t.Fatal("the session never registered a control channel")
	}
	select {
	case id := <-rec.unregistered:
		if id != "claim-1" {
			t.Fatalf("unregistered %q", id)
		}
	default:
		t.Fatal("the control channel was never released")
	}
}

func TestRunCodexAppServerSessionReportsAChildThatCrashes(t *testing.T) {
	session, rec := runCodexChild(t, "crash")

	if session.Status != "failed" {
		t.Fatalf("status = %q, want failed", session.Status)
	}
	if !strings.Contains(session.ErrorMessage, "closed") {
		t.Fatalf("error = %q", session.ErrorMessage)
	}
	if len(rec.classified) == 0 {
		t.Fatal("a failed session must be offered to the provider-error classifier")
	}
}

func TestRunCodexAppServerSessionFailsLaunchWithoutACommand(t *testing.T) {
	rec := newRecorder()
	RunCodexAppServerSession(context.Background(), testSession(t.TempDir()), testTask(), rec.options(t.TempDir()))
	if len(rec.failed) != 1 || !strings.Contains(rec.failed[0].Error(), "empty") {
		t.Fatalf("failed = %v", rec.failed)
	}
}

func TestReadCodexStdoutParsesLinesAndSkipsGarbage(t *testing.T) {
	input := strings.Join([]string{
		`{"id":1,"result":{"ok":true}}`,
		``,
		`this is not json`,
		`{"method":"turn/started","params":{"turnId":"u"}}`,
	}, "\n")
	messages := make(chan CodexRPCMessage, 8)
	done := make(chan error, 1)
	var log bytes.Buffer
	go readCodexStdout(strings.NewReader(input), &log, messages, done)

	var got []CodexRPCMessage
	for msg := range messages {
		got = append(got, msg)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || codexMessageID(got[0]) != 1 || got[1].Method != "turn/started" {
		t.Fatalf("messages = %#v", got)
	}
	if !strings.Contains(log.String(), "could not parse") {
		t.Fatalf("the unparseable line was not logged:\n%s", log.String())
	}
}

func TestCopyCodexStderrLogsEveryLine(t *testing.T) {
	var log bytes.Buffer
	copyCodexStderr(io.NopCloser(strings.NewReader("warn one\nwarn two\n")), &log)
	if !strings.Contains(log.String(), "[codex stderr] warn one") || !strings.Contains(log.String(), "[codex stderr] warn two") {
		t.Fatalf("log = %q", log.String())
	}
}

func TestCodexClientWritesOneJSONLinePerMessage(t *testing.T) {
	var wire, log bytes.Buffer
	client := NewCodexClient(&wire, &log)
	id1, _ := client.request("initialize", codexInitializeParams())
	id2, _ := client.request("thread/start", nil)
	_ = client.notify("initialized", nil)
	_ = client.respondError(json.RawMessage(`7`), -32601, "nope")

	if id1 != 1 || id2 != 2 {
		t.Fatalf("ids = %d, %d; they must count up per request", id1, id2)
	}
	lines := strings.Split(strings.TrimSpace(wire.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("wrote %d lines, want 4:\n%s", len(lines), wire.String())
	}
	var errMsg struct {
		ID    int `json:"id"`
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(lines[3]), &errMsg); err != nil || errMsg.ID != 7 || errMsg.Error.Code != -32601 {
		t.Fatalf("error response = %q (%v)", lines[3], err)
	}
	if strings.Contains(lines[2], "params") {
		t.Fatalf("a notification with no params carried one: %s", lines[2])
	}
	if !strings.Contains(log.String(), "[core -> codex]") {
		t.Fatalf("outgoing messages are not logged:\n%s", log.String())
	}
}

func TestCodexParamsAndMessageHelpers(t *testing.T) {
	resume := codexThreadResumeParams(RuntimeSession{WorkingDir: "/w", CodexSessionDetails: CodexSessionDetails{CodexThreadID: "t9"}})
	if resume["threadId"] != "t9" || resume["cwd"] != "/w" {
		t.Fatalf("resume params = %#v", resume)
	}
	if init := codexInitializeParams(); init["clientInfo"] == nil || init["capabilities"] == nil {
		t.Fatalf("initialize params = %#v", init)
	}
	if codexMessageID(CodexRPCMessage{}) != 0 || codexMessageID(CodexRPCMessage{ID: json.RawMessage(`"not-a-number"`)}) != 0 {
		t.Fatal("a missing or non-numeric id must read as 0")
	}
	long := json.RawMessage(`{"delta":"` + strings.Repeat("a", 400) + `"}`)
	if got := len(CodexMessageText(long)); got != 240 {
		t.Fatalf("display text is %d bytes, want 240", got)
	}
	if got := len(CodexMessageFullText(long)); got != 400 {
		t.Fatalf("full text is %d bytes, want 400", got)
	}
}

func TestUpdateCodexSessionKeepsACancelledSessionCancelled(t *testing.T) {
	session := &RuntimeSession{ExecutionStatus: "cancelled"}
	UpdateCodexSessionFromMessage(session, rpcNotify("item/agentMessage/delta", `{"delta":"late output"}`))
	if session.ExecutionStatus != "cancelled" {
		t.Fatalf("a late notification revived a cancelled session: %q", session.ExecutionStatus)
	}
	if UpdateCodexSessionFromMessage(session, CodexRPCMessage{}) {
		t.Fatal("a message with no method must not count as an update")
	}
}
