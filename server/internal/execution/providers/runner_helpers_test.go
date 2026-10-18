package providers

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

// recorder captures every callback a runner makes, so a test can assert on
// what the runner reported instead of on its internals.
type recorder struct {
	mu           sync.Mutex
	upserts      []RuntimeSession
	attention    []string
	failed       []error
	started      []string
	completed    []string
	comments     []string
	classified   []string
	registered   chan chan SessionControlRequest
	unregistered chan string
}

func newRecorder() *recorder {
	return &recorder{registered: make(chan chan SessionControlRequest, 4), unregistered: make(chan string, 4)}
}

func (r *recorder) options(root string) RunnerOptions {
	return RunnerOptions{
		RuntimeRoot:    root,
		SessionTimeout: time.Minute,
		Callbacks: RunnerCallbacks{
			ClassifyProviderError: func(provider, backend, operation string, err error, output string) *tasklifecycle.ProviderError {
				r.mu.Lock()
				defer r.mu.Unlock()
				r.classified = append(r.classified, provider+"/"+operation)
				return nil
			},
			FailLaunchSession: func(_ RuntimeSession, _ tasklifecycle.Task, err error) {
				r.mu.Lock()
				defer r.mu.Unlock()
				r.failed = append(r.failed, err)
			},
			RecordProviderSessionStart: func(session RuntimeSession, _ tasklifecycle.Task, message string, _ map[string]any) {
				r.mu.Lock()
				defer r.mu.Unlock()
				r.started = append(r.started, message)
				r.upserts = append(r.upserts, session)
			},
			CompleteProviderSession: func(session RuntimeSession, _ tasklifecycle.Task, message string, _ map[string]any) {
				r.mu.Lock()
				defer r.mu.Unlock()
				r.completed = append(r.completed, message)
				r.upserts = append(r.upserts, session)
			},
			UpsertSession: func(session RuntimeSession) {
				r.mu.Lock()
				defer r.mu.Unlock()
				r.upserts = append(r.upserts, session)
			},
			AddTaskComment: func(_ string, comment string) error {
				r.mu.Lock()
				defer r.mu.Unlock()
				r.comments = append(r.comments, comment)
				return nil
			},
			MarkSessionOperatorAttention: func(_ RuntimeSession, _ tasklifecycle.Task, reason string) {
				r.mu.Lock()
				defer r.mu.Unlock()
				r.attention = append(r.attention, reason)
			},
			RegisterSessionControl:   func(_ string, ch chan SessionControlRequest) { r.registered <- ch },
			UnregisterSessionControl: func(claimID string) { r.unregistered <- claimID },
		},
	}
}

func (r *recorder) lastSession(t *testing.T) RuntimeSession {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.upserts) == 0 {
		t.Fatal("the runner never reported a session")
	}
	return r.upserts[len(r.upserts)-1]
}

func testSession(workingDir string) RuntimeSession {
	return RuntimeSession{
		ClaimID:    "claim-1",
		TaskRef:    "CORE-1",
		TaskID:     "work-1",
		Agent:      "codex",
		Backend:    BackendCodexAppServer,
		LogPath:    "sessions/claim-1.log",
		WorkingDir: workingDir,
	}
}

func testTask() tasklifecycle.Task {
	task := tasklifecycle.Task{ID: "work-1", Ref: "CORE-1", Title: "Do the thing", RelativePath: "Work/p/tasks/t.md"}
	task.LaunchEvaluation.Prompt = "do the thing"
	return task
}

// writeScript writes an executable POSIX shell script and returns its path.
// Runner tests drive real child processes with it; they are skipped on
// Windows, where the repository only checks that the code builds.
func writeScript(t *testing.T, dir, name, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell-script fakes need a POSIX shell")
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// fakeCodex is the other end of the Codex JSON-RPC pipe: the client writes
// its requests here, and the handler decides which messages come back.
type fakeCodex struct {
	messages chan CodexRPCMessage
	handler  func(method string, id json.RawMessage, params json.RawMessage) []CodexRPCMessage
	mu       sync.Mutex
	methods  []string
	raw      []map[string]any
}

func newFakeCodex(handler func(method string, id json.RawMessage, params json.RawMessage) []CodexRPCMessage) *fakeCodex {
	return &fakeCodex{messages: make(chan CodexRPCMessage, 64), handler: handler}
}

func (f *fakeCodex) Write(p []byte) (int, error) {
	for _, line := range bytes.Split(bytes.TrimSpace(p), []byte("\n")) {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}
		var generic map[string]any
		_ = json.Unmarshal(line, &generic)
		f.mu.Lock()
		f.raw = append(f.raw, generic)
		if req.Method != "" {
			f.methods = append(f.methods, req.Method)
		}
		f.mu.Unlock()
		if req.Method == "" || f.handler == nil {
			continue // a response from the client (e.g. to a server request)
		}
		for _, msg := range f.handler(req.Method, req.ID, req.Params) {
			f.messages <- msg
		}
	}
	return len(p), nil
}

func (f *fakeCodex) sent() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return strings.Join(f.methods, ",")
}

func rpcResult(id json.RawMessage, result string) CodexRPCMessage {
	return CodexRPCMessage{ID: id, Result: json.RawMessage(result)}
}

func rpcNotify(method, params string) CodexRPCMessage {
	return CodexRPCMessage{Method: method, Params: json.RawMessage(params)}
}

// happyCodex answers the whole handshake and runs one turn to completion.
func happyCodex(finalStatus string, extra ...CodexRPCMessage) func(string, json.RawMessage, json.RawMessage) []CodexRPCMessage {
	return func(method string, id json.RawMessage, _ json.RawMessage) []CodexRPCMessage {
		switch method {
		case "initialize", "thread/name/set":
			return []CodexRPCMessage{rpcResult(id, `{}`)}
		case "thread/start", "thread/resume":
			return []CodexRPCMessage{rpcResult(id, `{"thread":{"id":"t1"}}`)}
		case "turn/start":
			out := []CodexRPCMessage{rpcResult(id, `{"turn":{"id":"u1"}}`)}
			out = append(out, extra...)
			return append(out, rpcNotify("turn/completed", `{"turn":{"id":"u1","status":"`+finalStatus+`"}}`))
		}
		return nil
	}
}
