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

// RelayCodexManagerMessage forwards a single Manager Bar message into an
// existing (or brand-new, if threadID is empty) Codex thread and returns the
// agent's final reply text for that turn. It reuses the same JSON-RPC
// sequence RunCodexAppServerSession uses (initialize -> thread/resume or
// thread/start -> turn/start -> wait for turn/completed), just without the
// session/task bookkeeping around it, and always tears the process down —
// on success, on error, and on the timeout context — so a bad handshake
// cannot leak a codex app-server process.
func RelayCodexManagerMessage(ctx context.Context, workingDir, threadID, prompt string, timeout time.Duration, logFile io.Writer) (RelayManagerResult, error) {
	if strings.TrimSpace(prompt) == "" {
		return RelayManagerResult{}, errors.New("prompt is required")
	}
	if logFile == nil {
		logFile = io.Discard
	}
	codexBinary, err := resolveCodexBinary()
	if err != nil {
		return RelayManagerResult{}, fmt.Errorf("resolve codex binary: %w", err)
	}

	relayCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(relayCtx, codexBinary, "app-server", "--stdio")
	cmd.Dir = workingDir
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return RelayManagerResult{}, fmt.Errorf("open codex stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return RelayManagerResult{}, fmt.Errorf("open codex stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return RelayManagerResult{}, fmt.Errorf("open codex stderr: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return RelayManagerResult{}, fmt.Errorf("start codex app-server: %w", err)
	}
	defer func() {
		cancel()
		_ = cmd.Wait()
	}()

	messages := make(chan CodexRPCMessage, 64)
	go readCodexStdout(stdout, logFile, messages, make(chan error, 1))
	go copyCodexStderr(stderr, logFile)

	client := NewCodexClient(stdin, logFile)
	var replyText string

	waitForID := func(id int) (CodexRPCMessage, error) {
		for {
			select {
			case <-relayCtx.Done():
				return CodexRPCMessage{}, fmt.Errorf("manager relay timed out waiting for codex: %w", relayCtx.Err())
			case msg, ok := <-messages:
				if !ok {
					return CodexRPCMessage{}, errors.New("codex app-server stdout closed before response")
				}
				if msg.Method != "" && len(msg.ID) > 0 {
					_ = client.respondError(msg.ID, -32601, "Core manager relay does not service server-initiated requests")
					continue
				}
				if text := CodexMessageFullText(msg.Params); text != "" {
					replyText = text
				} else if text := CodexMessageText(msg.Params); text != "" {
					replyText = text
				}
				if codexMessageID(msg) == id {
					if msg.Error != nil {
						return CodexRPCMessage{}, errors.New(msg.Error.Message)
					}
					return msg, nil
				}
			}
		}
	}

	id, err := client.request("initialize", codexInitializeParams())
	if err != nil {
		return RelayManagerResult{}, err
	}
	if _, err := waitForID(id); err != nil {
		return RelayManagerResult{}, err
	}
	if err := client.notify("initialized", nil); err != nil {
		return RelayManagerResult{}, err
	}

	resolvedThreadID := strings.TrimSpace(threadID)
	if resolvedThreadID != "" {
		resumeTarget := RuntimeSession{WorkingDir: workingDir}
		resumeTarget.CodexThreadID = resolvedThreadID
		id, err = client.request("thread/resume", codexThreadResumeParams(resumeTarget))
		if err != nil {
			return RelayManagerResult{}, err
		}
		response, resumeErr := waitForID(id)
		if resumeErr == nil {
			if fresh := CodexThreadID(response.Result); fresh != "" {
				resolvedThreadID = fresh
			}
		} else {
			// Resume failed (thread gone/expired) — fall through and start a
			// fresh thread instead of failing the whole manager message.
			resolvedThreadID = ""
		}
	}
	if resolvedThreadID == "" {
		id, err = client.request("thread/start", codexThreadStartParams(RuntimeSession{WorkingDir: workingDir}))
		if err != nil {
			return RelayManagerResult{}, err
		}
		response, err := waitForID(id)
		if err != nil {
			return RelayManagerResult{}, err
		}
		resolvedThreadID = CodexThreadID(response.Result)
		if resolvedThreadID == "" {
			return RelayManagerResult{}, errors.New("codex app-server thread/start response did not include thread.id")
		}
	}

	id, err = client.request("turn/start", codexTurnStartParams(resolvedThreadID, prompt))
	if err != nil {
		return RelayManagerResult{}, err
	}
	if _, err := waitForID(id); err != nil {
		return RelayManagerResult{}, err
	}

	// turn/start's own response only carries the turn id; the actual reply
	// text streams in afterwards as item/agentMessage events, terminated by
	// a turn/completed notification (no id — waitForID would block forever
	// waiting for a reply to a request nobody sent). Drain those directly.
	for {
		select {
		case <-relayCtx.Done():
			return RelayManagerResult{}, fmt.Errorf("manager relay timed out waiting for codex turn to complete: %w", relayCtx.Err())
		case msg, ok := <-messages:
			if !ok {
				return RelayManagerResult{}, errors.New("codex app-server stdout closed before turn/completed")
			}
			if msg.Method != "" && len(msg.ID) > 0 {
				_ = client.respondError(msg.ID, -32601, "Core manager relay does not service server-initiated requests")
				continue
			}
			if text := CodexMessageFullText(msg.Params); text != "" {
				replyText = text
			} else if text := CodexMessageText(msg.Params); text != "" {
				replyText = text
			}
			if msg.Method == "turn/completed" {
				status := codexTurnStatus(msg.Params)
				if status != "" && status != "completed" {
					return RelayManagerResult{ThreadID: resolvedThreadID, ReplyText: replyText}, fmt.Errorf("codex turn completed with status %q", status)
				}
				return RelayManagerResult{ThreadID: resolvedThreadID, ReplyText: replyText}, nil
			}
		}
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
	if logFile == nil {
		logFile = io.Discard
	}
	codexBinary, err := resolveCodexBinary()
	if err != nil {
		return nil, fmt.Errorf("resolve codex binary: %w", err)
	}

	listCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(listCtx, codexBinary, "app-server", "--stdio")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open codex stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open codex stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("open codex stderr: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start codex app-server: %w", err)
	}
	defer func() {
		cancel()
		_ = cmd.Wait()
	}()

	messages := make(chan CodexRPCMessage, 64)
	go readCodexStdout(stdout, logFile, messages, make(chan error, 1))
	go copyCodexStderr(stderr, logFile)

	client := NewCodexClient(stdin, logFile)

	waitForID := func(id int) (CodexRPCMessage, error) {
		for {
			select {
			case <-listCtx.Done():
				return CodexRPCMessage{}, fmt.Errorf("codex thread/list timed out: %w", listCtx.Err())
			case msg, ok := <-messages:
				if !ok {
					return CodexRPCMessage{}, errors.New("codex app-server stdout closed before response")
				}
				if msg.Method != "" && len(msg.ID) > 0 {
					_ = client.respondError(msg.ID, -32601, "Core thread listing does not service server-initiated requests")
					continue
				}
				if codexMessageID(msg) == id {
					if msg.Error != nil {
						return CodexRPCMessage{}, errors.New(msg.Error.Message)
					}
					return msg, nil
				}
			}
		}
	}

	id, err := client.request("initialize", codexInitializeParams())
	if err != nil {
		return nil, err
	}
	if _, err := waitForID(id); err != nil {
		return nil, err
	}
	if err := client.notify("initialized", nil); err != nil {
		return nil, err
	}

	id, err = client.request("thread/list", map[string]any{
		"limit":         50,
		"sortKey":       "updated_at",
		"sortDirection": "desc",
	})
	if err != nil {
		return nil, err
	}
	response, err := waitForID(id)
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
		seen[item.ID] = true
		summary := CodexThreadSummary{ID: item.ID, Name: strings.TrimSpace(item.Name), Cwd: item.Cwd}
		if item.UpdatedAt > 0 {
			summary.UpdatedAt = time.Unix(item.UpdatedAt, 0).UTC().Format(time.RFC3339)
		}
		threads = append(threads, summary)
	}
	return threads, nil
}
