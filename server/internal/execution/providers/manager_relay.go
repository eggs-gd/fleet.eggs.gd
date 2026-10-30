package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// RelayManagerResult is the outcome of one manager-chat round trip against a
// project-less agent thread (see the Manager session binding in
// internal/settings). Unlike RunCodexAppServerSession this is bounded and
// one-shot: spawn app-server, resume (or start) the given thread, send
// exactly one turn, capture the final agent text, exit. There is no
// RuntimeSession and no task here — a manager session belongs to neither,
// so none of the finalizer/session-registry machinery in runtime.go applies.
type RelayManagerResult struct {
	ThreadID  string
	ReplyText string
}

// RelayCodexManagerMessage forwards one message into the already bound Codex
// thread and returns that turn's final text. Resume failure stays a failure:
// a busy or missing binding must not start a second unnamed thread. The
// process is torn down on success, on error, and on timeout.
func RelayCodexManagerMessage(ctx context.Context, workingDir, threadID, prompt string, timeout time.Duration, logFile io.Writer) (RelayManagerResult, error) {
	if strings.TrimSpace(prompt) == "" {
		return RelayManagerResult{}, errors.New("prompt is required")
	}
	resolvedThreadID := strings.TrimSpace(threadID)
	server, err := startCodexAppServer(ctx, workingDir, timeout, logFile, "Fleet manager relay does not service server-initiated requests")
	if err != nil {
		return RelayManagerResult{}, err
	}
	defer server.close()
	server.timedOut = "manager relay timed out waiting for codex"
	server.trackReply = true

	if err := server.handshake(); err != nil {
		return RelayManagerResult{}, err
	}
	if resolvedThreadID == "" {
		return RelayManagerResult{}, errors.New("manager session is not bound")
	}
	if err := server.resumeThread(workingDir, resolvedThreadID); err != nil {
		return RelayManagerResult{}, err
	}
	id, err := server.client.request(methodTurnStart, codexTurnStartParams(resolvedThreadID, prompt))
	if err != nil {
		return RelayManagerResult{}, err
	}
	if _, err := server.waitForID(id); err != nil {
		return RelayManagerResult{}, err
	}
	reply, err := server.drainTurn()
	return RelayManagerResult{ThreadID: resolvedThreadID, ReplyText: reply}, err
}

// codexAppServer is a short-lived `codex app-server --stdio` child and the
// request/response plumbing around it.
type codexAppServer struct {
	ctx      context.Context
	client   *CodexClient
	messages chan CodexRPCMessage
	stop     func()

	// timedOut prefixes the error when the deadline passes while waiting.
	timedOut string
	// refusal answers a request the server sends us; a relay services none.
	refusal string
	// trackReply keeps the latest agent text seen in replyText.
	trackReply bool
	replyText  string
}

// startCodexAppServer spawns the app-server. The process is torn down by
// close, which the caller defers.
func startCodexAppServer(ctx context.Context, workingDir string, timeout time.Duration, logFile io.Writer, refusal string) (*codexAppServer, error) {
	if logFile == nil {
		logFile = io.Discard
	}
	codexBinary, err := resolveCodexBinary()
	if err != nil {
		return nil, fmt.Errorf("resolve codex binary: %w", err)
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	cmd := exec.CommandContext(runCtx, codexBinary, "app-server", "--stdio")
	cmd.Dir = workingDir
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("open codex stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("open codex stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("open codex stderr: %w", err)
	}
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start codex app-server: %w", err)
	}

	messages := make(chan CodexRPCMessage, 64)
	go readCodexStdout(stdout, logFile, messages, make(chan error, 1))
	go copyCodexStderr(stderr, logFile)
	return &codexAppServer{
		ctx:      runCtx,
		client:   NewCodexClient(stdin, logFile),
		messages: messages,
		stop: func() {
			cancel()
			_ = cmd.Wait()
		},
		timedOut: "codex timed out",
		refusal:  refusal,
	}, nil
}

func (a *codexAppServer) close() { a.stop() }

// next returns the next message that is not a server-initiated request.
func (a *codexAppServer) next(timedOut string, closed string) (CodexRPCMessage, error) {
	for {
		select {
		case <-a.ctx.Done():
			return CodexRPCMessage{}, fmt.Errorf("%s: %w", timedOut, a.ctx.Err())
		case msg, ok := <-a.messages:
			if !ok {
				return CodexRPCMessage{}, errors.New(closed)
			}
			if msg.Method != "" && len(msg.ID) > 0 {
				_ = a.client.respondError(msg.ID, -32601, a.refusal)
				continue
			}
			if a.trackReply {
				a.noteReply(msg)
			}
			return msg, nil
		}
	}
}

func (a *codexAppServer) noteReply(msg CodexRPCMessage) {
	if text := CodexMessageFullText(msg.Params); text != "" {
		a.replyText = text
	} else if text := CodexMessageText(msg.Params); text != "" {
		a.replyText = text
	}
}

func (a *codexAppServer) waitForID(id int) (CodexRPCMessage, error) {
	for {
		msg, err := a.next(a.timedOut, "codex app-server stdout closed before response")
		if err != nil {
			return CodexRPCMessage{}, err
		}
		if codexMessageID(msg) != id {
			continue
		}
		if msg.Error != nil {
			return CodexRPCMessage{}, errors.New(msg.Error.Message)
		}
		return msg, nil
	}
}

func (a *codexAppServer) handshake() error {
	id, err := a.client.request("initialize", codexInitializeParams())
	if err != nil {
		return err
	}
	if _, err := a.waitForID(id); err != nil {
		return err
	}
	return a.client.notify("initialized", nil)
}

// resumeThread resumes the bound thread. A busy or missing binding stays a
// failure rather than starting a second unnamed thread.
func (a *codexAppServer) resumeThread(workingDir string, threadID string) error {
	resumeTarget := RuntimeSession{WorkingDir: workingDir}
	resumeTarget.CodexThreadID = threadID
	id, err := a.client.request("thread/resume", codexThreadResumeParams(resumeTarget))
	if err != nil {
		return err
	}
	response, err := a.waitForID(id)
	if err != nil {
		return fmt.Errorf("bound manager session %s: %w", threadID, err)
	}
	if fresh := CodexThreadID(response.Result); fresh != "" && fresh != threadID {
		return fmt.Errorf("bound manager session %s resumed as %s", threadID, fresh)
	}
	return nil
}

// drainTurn reads the streamed reply after turn/start. The start response only
// carries the turn id; the text arrives as item/agentMessage events, ended by
// a turn/completed notification (no id — waitForID would block forever
// waiting for a reply to a request nobody sent).
func (a *codexAppServer) drainTurn() (string, error) {
	for {
		msg, err := a.next("manager relay timed out waiting for codex turn to complete", "codex app-server stdout closed before turn/completed")
		if err != nil {
			return "", err
		}
		if msg.Method != "turn/completed" {
			continue
		}
		status := codexTurnStatus(msg.Params)
		if status != "" && status != "completed" {
			return a.replyText, fmt.Errorf("codex turn completed with status %q", status)
		}
		return a.replyText, nil
	}
}

// CodexThreadSummary is one entry from `thread/list`, trimmed to what the
// Manager session picker needs — not the full Thread record the protocol
// returns (which also carries turn history, model settings, git info, and
// the message preview text, none of which belong in a binding picker).
type CodexThreadSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Cwd       string `json:"cwd,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// ListCodexManagerThreads asks a short-lived codex app-server for the most
// recently updated threads (`thread/list`, sorted newest first) so Settings
// > Manager can offer a picker instead of asking the operator to go find and
// paste a raw thread UUID themselves.
func ListCodexManagerThreads(ctx context.Context, timeout time.Duration, logFile io.Writer) ([]CodexThreadSummary, error) {
	server, err := startCodexAppServer(ctx, "", timeout, logFile, "Fleet thread listing does not service server-initiated requests")
	if err != nil {
		return nil, err
	}
	defer server.close()
	server.timedOut = "codex thread/list timed out"

	if err := server.handshake(); err != nil {
		return nil, err
	}
	id, err := server.client.request("thread/list", map[string]any{
		"limit":         50,
		"sortKey":       "updated_at",
		"sortDirection": "desc",
	})
	if err != nil {
		return nil, err
	}
	response, err := server.waitForID(id)
	if err != nil {
		return nil, err
	}
	return parseCodexThreadListResult(response.Result)
}

// parseCodexThreadListResult decodes a thread/list response into deduped
// CodexThreadSummary rows, newest first. codex can list the same thread id
// more than once (e.g. a rollout split across history entries) — results
// arrive newest-first, so keep only the first (most recent) occurrence per
// id. A picker with a duplicate id would also break Svelte's keyed #each on
// the frontend.
func parseCodexThreadListResult(raw json.RawMessage) ([]CodexThreadSummary, error) {
	var parsed struct {
		Data []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Cwd       string `json:"cwd"`
			Preview   string `json:"preview"`
			UpdatedAt int64  `json:"updatedAt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode thread/list response: %w", err)
	}

	seen := make(map[string]bool, len(parsed.Data))
	threads := make([]CodexThreadSummary, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		if seen[item.ID] {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(item.Preview), "Fleet event") {
			continue
		}
		seen[item.ID] = true
		summary := CodexThreadSummary{ID: item.ID, Name: strings.TrimSpace(item.Name), Cwd: item.Cwd}
		if item.UpdatedAt > 0 {
			summary.UpdatedAt = time.Unix(item.UpdatedAt, 0).UTC().Format(time.RFC3339)
		}
		threads = append(threads, summary)
	}
	return threads, nil
}
