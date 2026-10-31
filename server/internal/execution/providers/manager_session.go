package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

const managerSessionPrompt = "You are the Fleet manager for this workspace. The Fleet board is available through the manager MCP server. Talk with the operator in this app. Fleet does not send chat messages into this session."

// StartCodexManagerThread creates a Codex thread with cwd set to the data root
// and then exits app-server, so Codex Desktop can open that thread. It does
// not send a turn.
func StartCodexManagerThread(ctx context.Context, workingDir string, logFile io.Writer) (string, error) {
	server, err := startCodexAppServer(ctx, workingDir, 0, logFile, "Fleet manager setup does not service server-initiated requests")
	if err != nil {
		return "", err
	}
	defer server.close()
	server.timedOut = "codex manager setup stopped"

	if err := server.handshake(); err != nil {
		return "", err
	}
	id, err := server.client.request("thread/start", codexThreadStartParams(RuntimeSession{WorkingDir: workingDir}))
	if err != nil {
		return "", err
	}
	response, err := server.waitForID(id)
	if err != nil {
		return "", err
	}
	threadID := CodexThreadID(response.Result)
	if threadID == "" {
		return "", errors.New("codex thread/start did not include thread.id")
	}
	if nameID, nameErr := server.client.request("thread/name/set", codexThreadSetNameParams(threadID, "Fleet Manager")); nameErr == nil {
		_, _ = server.waitForID(nameID)
	}
	return threadID, nil
}

// StartClaudeManagerSession dispatches a visible remote-control session whose
// working directory is the data root. The returned id is the background id.
func StartClaudeManagerSession(ctx context.Context, workingDir string) (string, error) {
	binary, err := resolveClaudeBinary()
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, binary, "--bg", "--remote-control", "--name", "Fleet Manager", "--permission-mode", "acceptEdits", managerSessionPrompt)
	cmd.Dir = workingDir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude manager session: %w: %s", err, strings.TrimSpace(out.String()))
	}
	match := backgroundSessionIDPattern.FindStringSubmatch(out.String())
	if len(match) < 2 {
		return "", fmt.Errorf("claude manager session did not print a background id: %s", strings.TrimSpace(out.String()))
	}
	return match[1], nil
}

// StartGeminiManagerSession registers the data root as an Antigravity project
// when needed and runs one headless turn, the same launch workers use.
// The returned id is the conversation id, or the project id if that is all
// the CLI recorded.
func StartGeminiManagerSession(ctx context.Context, workingDir string) (string, error) {
	binary, err := resolveGeminiBinary()
	if err != nil {
		return "", err
	}
	projectID := findGeminiProjectID(workingDir)
	args := []string{}
	if projectID != "" {
		args = append(args, "--project", projectID)
	} else {
		args = append(args, "--new-project")
	}
	args = append(args, "--print", managerSessionPrompt, "--output-format", "stream-json")
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = workingDir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	runErr := cmd.Run()
	if id := geminiConversationID(out.Bytes()); id != "" {
		return id, nil
	}
	if projectID == "" {
		projectID = findGeminiProjectID(workingDir)
	}
	if runErr != nil {
		return "", fmt.Errorf("gemini manager session: %w: %s", runErr, strings.TrimSpace(out.String()))
	}
	if projectID == "" {
		return "", errors.New("gemini manager session did not return a conversation or project id")
	}
	return projectID, nil
}

func geminiConversationID(raw []byte) string {
	for _, line := range bytes.Split(raw, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var event struct {
			ConversationID string `json:"conversation_id"`
		}
		if json.Unmarshal(line, &event) == nil && strings.TrimSpace(event.ConversationID) != "" {
			return strings.TrimSpace(event.ConversationID)
		}
	}
	return ""
}
