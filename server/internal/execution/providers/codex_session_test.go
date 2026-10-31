package providers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func driveFake(t *testing.T, fake *fakeCodex, session *RuntimeSession, idle time.Duration, controls chan SessionControlRequest) (*recorder, error) {
	t.Helper()
	rec := newRecorder()
	return rec, driveWith(t, rec, fake, session, idle, controls)
}

func driveWith(t *testing.T, rec *recorder, fake *fakeCodex, session *RuntimeSession, idle time.Duration, controls chan SessionControlRequest) error {
	t.Helper()
	opts := rec.options(t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if controls == nil {
		controls = make(chan SessionControlRequest, 4)
	}
	client := NewCodexClient(fake, &strings.Builder{})
	return (&codexTurn{ctx: ctx, cancel: cancel, client: client, messages: fake.messages, controls: controls, session: session, task: testTask(), idleTimeout: idle, opts: opts}).drive()
}

func TestDriveCodexTurnRunsTheHandshakeAndOneTurn(t *testing.T) {
	fake := newFakeCodex(happyCodex("completed",
		rpcNotify("item/agentMessage/delta", `{"threadId":"t1","turnId":"u1","delta":"working on it"}`)))
	session := testSession(t.TempDir())

	rec, err := driveFake(t, fake, &session, time.Minute, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := fake.sent(), "initialize,initialized,thread/start,thread/name/set,turn/start"; got != want {
		t.Fatalf("requests = %s, want %s", got, want)
	}
	if session.CodexThreadID != "t1" || session.CodexTurnID != "u1" {
		t.Fatalf("ids = %q/%q", session.CodexThreadID, session.CodexTurnID)
	}
	if session.ExecutionStatus != "succeeded" {
		t.Fatalf("execution status = %q", session.ExecutionStatus)
	}
	if session.LastMessage != "working on it" {
		t.Fatalf("last message = %q", session.LastMessage)
	}
	if session.CodexThreadTitle != "CORE-1 · Do the thing" {
		t.Fatalf("thread title = %q", session.CodexThreadTitle)
	}
	if len(rec.upserts) == 0 {
		t.Fatal("the session was never reported")
	}
}

func TestDriveCodexTurnPicksUpAWorkerResultFromTheStream(t *testing.T) {
	result := `{"outcome":"completed","summary":"all done"}`
	fake := newFakeCodex(happyCodex("completed",
		rpcNotify("item/agentMessage/delta", `{"delta":`+jsonString(result)+`}`)))
	session := testSession(t.TempDir())

	if _, err := driveFake(t, fake, &session, time.Minute, nil); err != nil {
		t.Fatal(err)
	}
	if session.Result == nil || session.Result.Outcome != "completed" || session.Result.Summary != "all done" {
		t.Fatalf("result = %#v", session.Result)
	}
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestDriveCodexTurnResumesAKnownThread(t *testing.T) {
	fake := newFakeCodex(happyCodex("completed"))
	session := testSession(t.TempDir())
	session.CodexThreadID = "t1"

	if _, err := driveFake(t, fake, &session, time.Minute, nil); err != nil {
		t.Fatal(err)
	}
	if got, want := fake.sent(), "initialize,initialized,thread/resume,thread/name/set,turn/start"; got != want {
		t.Fatalf("requests = %s, want %s", got, want)
	}
	if session.ResumeOutcome != "resumed" {
		t.Fatalf("resume outcome = %q", session.ResumeOutcome)
	}
}

func TestDriveCodexTurnFallsBackToANewThreadWhenResumeFails(t *testing.T) {
	happy := happyCodex("completed")
	fake := newFakeCodex(func(method string, id, params json.RawMessage) []CodexRPCMessage {
		if method == "thread/resume" {
			return []CodexRPCMessage{{ID: id, Error: &CodexRPCError{Code: -1, Message: "thread not found"}}}
		}
		return happy(method, id, params)
	})
	session := testSession(t.TempDir())
	session.CodexThreadID = "gone"

	if _, err := driveFake(t, fake, &session, time.Minute, nil); err != nil {
		t.Fatal(err)
	}
	if got, want := fake.sent(), "initialize,initialized,thread/resume,thread/start,thread/name/set,turn/start"; got != want {
		t.Fatalf("requests = %s, want %s", got, want)
	}
	if session.ResumeOutcome != "fallback_new" || session.CodexThreadID != "t1" {
		t.Fatalf("resume outcome = %q, thread = %q", session.ResumeOutcome, session.CodexThreadID)
	}
}

func TestDriveCodexTurnRejectsAThreadStartWithoutAnID(t *testing.T) {
	fake := newFakeCodex(func(method string, id, _ json.RawMessage) []CodexRPCMessage {
		if method == "thread/start" {
			return []CodexRPCMessage{rpcResult(id, `{}`)}
		}
		return []CodexRPCMessage{rpcResult(id, `{}`)}
	})
	session := testSession(t.TempDir())
	_, err := driveFake(t, fake, &session, time.Minute, nil)
	if err == nil || !strings.Contains(err.Error(), "thread.id") {
		t.Fatalf("err = %v", err)
	}
}

func TestDriveCodexTurnReportsHowATurnEnded(t *testing.T) {
	cases := []struct {
		status, wantExec string
	}{
		{"failed", ""},
		{"interrupted", "cancelled"},
	}
	for _, c := range cases {
		fake := newFakeCodex(happyCodex(c.status))
		session := testSession(t.TempDir())
		_, err := driveFake(t, fake, &session, time.Minute, nil)
		if err == nil || !strings.Contains(err.Error(), c.status) {
			t.Fatalf("%s: err = %v", c.status, err)
		}
		if session.ExecutionStatus != c.wantExec && c.wantExec != "" {
			t.Fatalf("%s: execution status = %q, want %q", c.status, session.ExecutionStatus, c.wantExec)
		}
	}
}

func TestDriveCodexTurnAnswersServerRequestsWithAnErrorAndWaitsForInput(t *testing.T) {
	happy := happyCodex("completed")
	fake := newFakeCodex(func(method string, id, params json.RawMessage) []CodexRPCMessage {
		if method == "turn/start" {
			out := happy(method, id, params)
			// The server asks the client something mid-turn, before completing.
			ask := CodexRPCMessage{ID: json.RawMessage(`99`), Method: "item/commandExecution/requestApproval", Params: json.RawMessage(`{}`)}
			return append([]CodexRPCMessage{out[0], ask}, out[1:]...)
		}
		return happy(method, id, params)
	})
	session := testSession(t.TempDir())
	if _, err := driveFake(t, fake, &session, time.Minute, nil); err != nil {
		t.Fatal(err)
	}
	var answered bool
	fake.mu.Lock()
	for _, msg := range fake.raw {
		if msg["id"] == float64(99) {
			if e, ok := msg["error"].(map[string]any); ok && e["code"] == float64(-32601) {
				answered = true
			}
		}
	}
	fake.mu.Unlock()
	if !answered {
		t.Fatal("the server-initiated request got no error response")
	}
}

func TestDriveCodexTurnFailsWhenStdoutClosesEarly(t *testing.T) {
	var fake *fakeCodex
	fake = newFakeCodex(func(method string, id, _ json.RawMessage) []CodexRPCMessage {
		close(fake.messages) // the app-server dies right after the first request
		return nil
	})
	session := testSession(t.TempDir())
	_, err := driveFake(t, fake, &session, time.Minute, nil)
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("err = %v", err)
	}
}

func TestDriveCodexTurnAsksForAttentionOnceWhenIdle(t *testing.T) {
	handler := happyCodex("completed")
	fake := newFakeCodex(func(method string, id, params json.RawMessage) []CodexRPCMessage {
		if method == "turn/start" {
			return handler(method, id, params)[:1] // accept the turn, never finish it
		}
		return handler(method, id, params)
	})
	session := testSession(t.TempDir())
	rec := newRecorder()
	opts := rec.options(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- (&codexTurn{ctx: ctx, cancel: cancel, client: NewCodexClient(fake, &strings.Builder{}), messages: fake.messages, controls: make(chan SessionControlRequest, 1), session: &session, task: testTask(), idleTimeout: 40 * time.Millisecond, opts: opts}).drive()
	}()
	time.Sleep(250 * time.Millisecond) // several idle periods
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.attention) != 1 {
		t.Fatalf("operator was asked %d times, want once: %v", len(rec.attention), rec.attention)
	}
	if session.ExecutionStatus != "operator_attention" || session.LastEvent != "idle_operator_attention" {
		t.Fatalf("session = %q / %q", session.ExecutionStatus, session.LastEvent)
	}
}

func TestDriveCodexTurnObeysOperatorControls(t *testing.T) {
	handler := happyCodex("completed")
	fake := newFakeCodex(func(method string, id, params json.RawMessage) []CodexRPCMessage {
		if method == "turn/start" {
			return handler(method, id, params)[:1]
		}
		return handler(method, id, params)
	})
	session := testSession(t.TempDir())
	controls := make(chan SessionControlRequest, 4)
	done := make(chan error, 1)
	rec := newRecorder()
	go func() { done <- driveWith(t, rec, fake, &session, time.Minute, controls) }()

	// Wait until the turn is running, then continue it, then cancel it.
	waitFor(t, func() bool { return strings.HasSuffix(fake.sent(), "turn/start") })
	cont := SessionControlRequest{Action: "continue", Input: "keep going", Done: make(chan error, 1)}
	controls <- cont
	if err := <-cont.Done; err != nil {
		t.Fatal(err)
	}
	// The new turn's id arrives with its first notification; cancel needs it.
	fake.messages <- rpcNotify("turn/started", `{"threadId":"t1","turnId":"u2"}`)
	waitFor(t, func() bool {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		return len(rec.upserts) > 0 && rec.upserts[len(rec.upserts)-1].CodexTurnID == "u2"
	})
	cancelReq := SessionControlRequest{Action: "cancel", Done: make(chan error, 1)}
	controls <- cancelReq
	if err := <-cancelReq.Done; err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if got := fake.sent(); !strings.Contains(got, "turn/start,turn/start,turn/interrupt") {
		t.Fatalf("requests = %s (want a second turn/start for continue, then turn/interrupt for cancel)", got)
	}
	if session.ExecutionStatus != "cancelled" {
		t.Fatalf("execution status = %q", session.ExecutionStatus)
	}
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not reached in time")
}

func TestHandleCodexSessionControlValidatesWhatItNeeds(t *testing.T) {
	fake := newFakeCodex(nil)
	client := NewCodexClient(fake, &strings.Builder{})
	cases := []struct {
		name    string
		session RuntimeSession
		control SessionControlRequest
		wantErr string
	}{
		{"interrupt needs ids", RuntimeSession{}, SessionControlRequest{Action: "interrupt"}, "thread id and turn id"},
		{"input needs a thread", RuntimeSession{}, SessionControlRequest{Action: "input", Input: "x"}, "thread id"},
		{"input needs text", RuntimeSession{CodexSessionDetails: CodexSessionDetails{CodexThreadID: "t"}}, SessionControlRequest{Action: "input", Input: "  "}, "non-empty"},
		{"unknown action", RuntimeSession{}, SessionControlRequest{Action: "explode"}, "unsupported"},
	}
	for _, c := range cases {
		session := c.session
		err := HandleCodexSessionControl(client, &session, c.control)
		if err == nil || !strings.Contains(err.Error(), c.wantErr) {
			t.Errorf("%s: err = %v, want %q", c.name, err, c.wantErr)
		}
	}
	if fake.sent() != "" {
		t.Fatalf("a rejected control still wrote %q", fake.sent())
	}

	session := RuntimeSession{CodexSessionDetails: CodexSessionDetails{CodexThreadID: "t", CodexTurnID: "u"}}
	if err := HandleCodexSessionControl(client, &session, SessionControlRequest{Action: "interrupt"}); err != nil {
		t.Fatal(err)
	}
	if session.LastEvent != "operator_interrupt" || session.ExecutionStatus != "running" {
		t.Fatalf("after interrupt: %q / %q", session.LastEvent, session.ExecutionStatus)
	}
}
