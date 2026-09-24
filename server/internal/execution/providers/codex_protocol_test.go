package providers

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestCodexThreadStartParamsUseVerifiedNonInteractivePolicy(t *testing.T) {
	params := CodexThreadStartParams(RuntimeSession{WorkingDir: "/tmp/repo"})

	if params["cwd"] != "/tmp/repo" {
		t.Fatalf("cwd = %v", params["cwd"])
	}
	if params["approvalPolicy"] != "never" {
		t.Fatalf("approvalPolicy = %v, want never", params["approvalPolicy"])
	}
	if params["sandbox"] != "workspace-write" {
		t.Fatalf("sandbox = %v, want workspace-write", params["sandbox"])
	}
	if _, ok := params["metadata"]; ok {
		t.Fatal("thread/start params should not include undocumented metadata fields")
	}
}

func TestCodexThreadSetNameParamsUseProtocolMethodShape(t *testing.T) {
	params := CodexThreadSetNameParams("thr_123", "CORE-80 · Set meaningful Codex Remote thread titles")

	if params["threadId"] != "thr_123" {
		t.Fatalf("threadId = %v", params["threadId"])
	}
	if params["name"] != "CORE-80 · Set meaningful Codex Remote thread titles" {
		t.Fatalf("name = %v", params["name"])
	}
}

func TestCodexTurnStartParamsSendTextInput(t *testing.T) {
	params := CodexTurnStartParams("thr_123", "do the task")

	if params["threadId"] != "thr_123" {
		t.Fatalf("threadId = %v", params["threadId"])
	}
	input, ok := params["input"].([]map[string]string)
	if !ok || len(input) != 1 {
		t.Fatalf("input = %#v, want one text input item", params["input"])
	}
	if input[0]["type"] != "text" || input[0]["text"] != "do the task" {
		t.Fatalf("input[0] = %#v", input[0])
	}
}

func TestCodexProtocolExtractorsReadThreadTurnAndStatus(t *testing.T) {
	threadResult := json.RawMessage(`{"thread":{"id":"thr_123","status":{"type":"idle"}}}`)
	turnResult := json.RawMessage(`{"turn":{"id":"turn_456","status":"completed"}}`)
	changed := json.RawMessage(`{"threadId":"thr_123","turnId":"turn_456","delta":"hello"}`)

	if got := CodexThreadID(threadResult); got != "thr_123" {
		t.Fatalf("thread id = %q", got)
	}
	if got := CodexTurnID(turnResult); got != "turn_456" {
		t.Fatalf("turn id = %q", got)
	}
	if got := CodexTurnStatus(turnResult); got != "completed" {
		t.Fatalf("turn status = %q", got)
	}
	if got := CodexThreadID(changed); got != "thr_123" {
		t.Fatalf("notification thread id = %q", got)
	}
	if got := CodexTurnID(changed); got != "turn_456" {
		t.Fatalf("notification turn id = %q", got)
	}
	if got := CodexMessageText(changed); got != "hello" {
		t.Fatalf("message text = %q", got)
	}
}

// TestUpdateCodexSessionFromMessageParsesResultFromUntruncatedText covers a
// CORE-97 fix: session.LastMessage is intentionally truncated to 240 chars
// for display, but a worker result payload with several artifacts/tests
// entries can easily exceed that. Before this fix, updateCodexSessionFromMessage
// parsed the truncated display text and silently dropped results that ran
// past the cutoff.
func TestUpdateCodexSessionFromMessageParsesResultFromUntruncatedText(t *testing.T) {
	longSummary := strings.Repeat("implemented the thing thoroughly and carefully. ", 8)
	payload := `{"outcome":"completed","summary":"` + longSummary + `","artifacts":["a.go","b.go"],"tests":["go test ./..."]}`
	if len(payload) <= 240 {
		t.Fatalf("test payload must exceed the 240-char display truncation to be meaningful, got %d bytes", len(payload))
	}
	text := "Done.\n\n" + payload
	raw := json.RawMessage(`{"delta":` + marshalJSONString(t, text) + `}`)

	session := &RuntimeSession{}
	UpdateCodexSessionFromMessage(session, CodexRPCMessage{Method: "item/agentMessage/delta", Params: raw})

	if session.Result == nil {
		t.Fatal("session.Result was not parsed from untruncated message text")
	}
	if session.Result.Outcome != "completed" {
		t.Fatalf("outcome = %q, want completed", session.Result.Outcome)
	}
	if len(session.Result.Artifacts) != 2 || session.Result.Artifacts[1] != "b.go" {
		t.Fatalf("artifacts = %#v", session.Result.Artifacts)
	}
	if len(session.LastMessage) > 240 {
		t.Fatalf("LastMessage should stay truncated for display, got %d bytes", len(session.LastMessage))
	}
}

func TestCodexSessionControlsInterruptAndCancelExplicitly(t *testing.T) {
	var stdin bytes.Buffer
	var log bytes.Buffer
	client := NewCodexClient(&stdin, &log)
	session := RuntimeSession{
		CodexSessionDetails: CodexSessionDetails{
			CodexThreadID: "thread-1",
			CodexTurnID:   "turn-1",
		},
		ExecutionStatus: "operator_attention",
		BlockingReason:  "idle",
	}

	if err := HandleCodexSessionControl(client, &session, SessionControlRequest{Action: "interrupt"}); err != nil {
		t.Fatal(err)
	}
	if session.ExecutionStatus != "running" || session.BlockingReason != "" || session.LastEvent != "operator_interrupt" {
		t.Fatalf("interrupt session = %#v, want running operator interrupt", session)
	}
	if !strings.Contains(stdin.String(), `"method":"turn/interrupt"`) {
		t.Fatalf("interrupt did not write turn/interrupt request: %s", stdin.String())
	}

	if err := HandleCodexSessionControl(client, &session, SessionControlRequest{Action: "cancel"}); err != nil {
		t.Fatal(err)
	}
	if session.ExecutionStatus != "cancelled" || session.LastEvent != "operator_cancel" {
		t.Fatalf("cancel session = %#v, want explicit cancelled state", session)
	}
}

func marshalJSONString(t *testing.T, value string) string {
	t.Helper()
	out, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}
