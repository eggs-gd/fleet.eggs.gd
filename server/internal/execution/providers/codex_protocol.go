package providers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	execution "github.com/eggs-gd/fleet.eggs.gd/internal/executionapi"
	"github.com/eggs-gd/fleet.eggs.gd/internal/tasklifecycle"
)

type CodexRPCMessage struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *CodexRPCError  `json:"error,omitempty"`
}

type CodexRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type CodexClient struct {
	stdin   io.Writer
	logFile io.Writer
	mu      sync.Mutex
	nextID  int
}

func NewCodexClient(stdin io.Writer, logFile io.Writer) *CodexClient {
	return &CodexClient{stdin: stdin, logFile: logFile}
}

func CodexThreadStartParams(session RuntimeSession) map[string]any {
	return codexThreadStartParams(session)
}

func CodexThreadSetNameParams(threadID string, name string) map[string]any {
	return codexThreadSetNameParams(threadID, name)
}

func CodexTurnStartParams(threadID string, prompt string) map[string]any {
	return codexTurnStartParams(threadID, prompt)
}

func CodexTurnStatus(raw json.RawMessage) string {
	return codexTurnStatus(raw)
}

func UpdateCodexSessionFromMessage(session *RuntimeSession, msg CodexRPCMessage) bool {
	return updateCodexSessionFromMessage(session, msg, RunnerOptions{})
}

func (client *CodexClient) request(method string, params any) (int, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.nextID++
	msg := map[string]any{"id": client.nextID, "method": method}
	if params != nil {
		msg["params"] = params
	}
	return client.nextID, client.writeLocked(msg)
}

func (client *CodexClient) notify(method string, params any) error {
	client.mu.Lock()
	defer client.mu.Unlock()
	msg := map[string]any{"method": method}
	if params != nil {
		msg["params"] = params
	}
	return client.writeLocked(msg)
}

func (client *CodexClient) respondError(id json.RawMessage, code int, message string) error {
	client.mu.Lock()
	defer client.mu.Unlock()
	msg := map[string]any{
		"id":    json.RawMessage(id),
		"error": map[string]any{"code": code, "message": message},
	}
	return client.writeLocked(msg)
}

func (client *CodexClient) writeLocked(msg map[string]any) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if _, err := client.stdin.Write(append(payload, '\n')); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(client.logFile, "[core -> codex] %s\n", payload)
	return nil
}

func CodexThreadID(raw json.RawMessage) string {
	var payload struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
		ThreadID string `json:"threadId"`
	}
	_ = json.Unmarshal(raw, &payload)
	return firstNonEmpty(payload.Thread.ID, payload.ThreadID)
}

func CodexTurnID(raw json.RawMessage) string {
	var payload struct {
		Turn struct {
			ID string `json:"id"`
		} `json:"turn"`
		TurnID string `json:"turnId"`
	}
	_ = json.Unmarshal(raw, &payload)
	return firstNonEmpty(payload.Turn.ID, payload.TurnID)
}

func CodexMessageFullText(raw json.RawMessage) string {
	var payload struct {
		Delta string `json:"delta"`
		Item  struct {
			Text string `json:"text"`
		} `json:"item"`
	}
	_ = json.Unmarshal(raw, &payload)
	return firstNonEmpty(payload.Delta, payload.Item.Text)
}

func CodexMessageText(raw json.RawMessage) string {
	text := CodexMessageFullText(raw)
	if len(text) > 240 {
		return text[:240]
	}
	return text
}

func updateCodexSessionFromMessage(session *RuntimeSession, msg CodexRPCMessage, opts RunnerOptions) bool {
	if msg.Method == "" {
		return false
	}
	session.LastEvent = msg.Method
	switch msg.Method {
	case "turn/started", "item/started", "item/completed", "item/agentMessage/delta", "thread/status/changed":
		if session.ExecutionStatus != "cancelled" {
			session.ExecutionStatus = "running"
			session.BlockingReason = ""
		}
	}
	if strings.HasPrefix(msg.Method, "item/") {
		mergeSessionToolCalls(session, execution.ExtractToolCallsFromCodexParams(msg.Method, msg.Params, time.Now()))
	}
	if text := CodexMessageText(msg.Params); text != "" {
		session.LastMessage = truncateDisplayText(text)
	}
	if fullText := CodexMessageFullText(msg.Params); fullText != "" {
		if result := tasklifecycle.ParseWorkerResult(fullText); result != nil {
			session.Result = result
			session.LastEvent = "worker_result_reported"
			if summary := strings.TrimSpace(result.Summary); summary != "" {
				session.LastMessage = truncateDisplayText(summary)
			}
		}
	}
	if turnID := CodexTurnID(msg.Params); turnID != "" {
		session.CodexTurnID = turnID
	}
	if threadID := CodexThreadID(msg.Params); threadID != "" {
		session.CodexThreadID = threadID
	}
	upsertSession(opts, *session)
	return true
}

func readCodexStdout(stdout io.Reader, logFile io.Writer, messages chan<- CodexRPCMessage, done chan<- error) {
	defer close(messages)
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		_, _ = fmt.Fprintf(logFile, "[codex -> core] %s\n", line)
		var msg CodexRPCMessage
		if err := json.Unmarshal(line, &msg); err == nil {
			messages <- msg
		} else {
			_, _ = fmt.Fprintf(logFile, "[core] could not parse codex JSON-RPC line: %s\n", err)
		}
	}
	done <- scanner.Err()
}

func copyCodexStderr(stderr io.Reader, logFile io.Writer) {
	scanner := bufio.NewScanner(stderr)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		_, _ = fmt.Fprintf(logFile, "[codex stderr] %s\n", scanner.Text())
	}
}

func codexInitializeParams() map[string]any {
	return map[string]any{
		"clientInfo":   map[string]any{"name": "core_launcher", "title": "Core Agent Launcher", "version": "0.1.0"},
		"capabilities": map[string]any{"experimentalApi": true},
	}
}

func codexThreadStartParams(session RuntimeSession) map[string]any {
	return map[string]any{"cwd": session.WorkingDir, "approvalPolicy": "never", "sandbox": "workspace-write"}
}

func codexThreadResumeParams(session RuntimeSession) map[string]any {
	return map[string]any{"threadId": session.CodexThreadID, "cwd": session.WorkingDir}
}

func codexThreadSetNameParams(threadID string, name string) map[string]any {
	return map[string]any{"threadId": threadID, "name": name}
}

func codexTurnStartParams(threadID string, prompt string) map[string]any {
	return map[string]any{"threadId": threadID, "input": []map[string]string{{"type": "text", "text": prompt}}}
}

func codexTurnStatus(raw json.RawMessage) string {
	var payload struct {
		Turn struct {
			Status string `json:"status"`
		} `json:"turn"`
	}
	_ = json.Unmarshal(raw, &payload)
	return strings.TrimSpace(payload.Turn.Status)
}

func codexMessageID(msg CodexRPCMessage) int {
	if len(msg.ID) == 0 {
		return 0
	}
	var id int
	_ = json.Unmarshal(msg.ID, &id)
	return id
}
